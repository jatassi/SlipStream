package indexers

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/cardigann"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers/internal/db"
	"github.com/jatassi/slipstream/internal/newznab"
	"github.com/jatassi/slipstream/internal/storage"
)

var (
	ErrNoDefinition = errors.New("no definition has that id")
	// ErrCatalogUnavailable is a catalog definition asked for before any
	// refresh has read the catalog.
	ErrCatalogUnavailable = errors.New("the Indexer catalog hasn't been downloaded")
	ErrDefinitionIDTaken  = errors.New("the Indexer catalog already has a definition with this id")
	ErrDefinitionInUse    = errors.New("an Indexer runs this definition")
)

// CatalogUnreachableError is an address that serves no catalog SlipStream
// can read.
type CatalogUnreachableError struct{ Err error }

func (e *CatalogUnreachableError) Error() string {
	return "no Indexer catalog at that address: " + e.Err.Error()
}

func (e *CatalogUnreachableError) Unwrap() error { return e.Err }

// CatalogView is the Indexer catalog as the last good refresh listed it,
// with the custom definitions.
type CatalogView struct {
	// Refreshed is zero until a refresh has read the catalog.
	Refreshed time.Time
	// Next is when the refresh runs next, zero when it isn't waiting.
	Next      time.Time
	Address   string
	IsDefault bool
	// Notice says why the list holds no catalog site yet, empty once it
	// does.
	Notice  string
	Entries []Entry
}

// Entry is one definition an Admin can add an Indexer from.
type Entry struct {
	ID, File, Name, Description, Language string
	Privacy                               cardigann.Privacy
	// Categories are the Newznab categories the definition maps to.
	Categories []newznab.ID
	Origin     Origin
	// Reason is why SlipStream can't run the definition, empty when it
	// can.
	Reason string
	// ReplacedBy is a catalog entry whose id a custom definition has.
	ReplacedBy bool
	// Version is when SlipStream fetched this version.
	Version time.Time
}

// Catalog lists the catalog's definitions by name, then the custom ones.
func (x *Indexers) Catalog(ctx context.Context, _ identity.Admin) (CatalogView, error) {
	var c CatalogView
	err := x.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		q := db.New(r)
		state, err := q.GetCatalogState(ctx)
		if err != nil {
			return err
		}
		c.Refreshed = state.RefreshedAt.Time()
		c.Address, c.IsDefault = deref(state.Address), state.Address == nil
		if c.IsDefault {
			c.Address = x.cfg.Endpoints.Catalog.String()
		}
		custom, err := customEntries(ctx, q)
		if err != nil {
			return err
		}
		replaced := map[string]bool{}
		for _, e := range custom {
			replaced[e.ID] = true
		}
		rows, err := q.ListCatalog(ctx)
		if err != nil {
			return err
		}
		for _, row := range rows {
			e := Entry{
				ID: string(row.ID), File: row.File, Name: row.Name, Description: row.Description, Language: row.Language,
				Privacy: cardigann.Privacy(row.Privacy), Origin: FromCatalog, Reason: deref(row.Refusal), ReplacedBy: replaced[string(row.ID)],
				Version: row.FetchedAt.Time(),
			}
			if err := decode(row.Categories, &e.Categories); err != nil {
				return fmt.Errorf("catalog entry %s: %w", row.ID, err)
			}
			c.Entries = append(c.Entries, e)
		}
		c.Entries = append(c.Entries, custom...)
		return nil
	})
	if err != nil {
		return CatalogView{}, err
	}
	if c.Next, err = x.cfg.CatalogNext(ctx); err != nil {
		return CatalogView{}, err
	}
	if c.Refreshed.IsZero() {
		c.Notice = x.notDownloaded(c.Next)
	}
	return c, nil
}

func (x *Indexers) notDownloaded(next time.Time) string {
	if next.IsZero() {
		return "SlipStream hasn't downloaded the Indexer catalog yet. SlipStream retries automatically."
	}
	return "SlipStream hasn't downloaded the Indexer catalog yet. SlipStream retries at " + x.clock(next) + "."
}

func customEntries(ctx context.Context, q *db.Queries) ([]Entry, error) {
	rows, err := q.ListCustom(ctx)
	if err != nil {
		return nil, err
	}
	var out []Entry
	for _, row := range rows {
		e, err := customEntry(row)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, nil
}

// customEntry is a custom definition as the catalog lists it. A custom
// definition always loads, since AddCustom refuses one that doesn't.
func customEntry(row db.IndexersDefinition) (Entry, error) {
	s, _, err := summaryOf(row)
	if err != nil {
		return Entry{}, err
	}
	return Entry{
		ID: string(row.ID), File: row.File, Name: s.Name, Description: s.Description, Language: s.Language, Privacy: s.Privacy,
		Categories: s.Caps.Categories, Origin: FromCustom, Version: row.FetchedAt.Time(),
	}, nil
}

// Form is one definition's fields for the add form.
type Form struct {
	Entry
	Links  []string
	Fields []FormField
	// Session is a definition whose Indexers hold a login session.
	Session bool
	Captcha bool
	// Solver is a definition that asks for a Challenge solver.
	Solver bool
}

type FormField struct {
	cardigann.Field
	// Sealed is a value SlipStream never shows again once saved.
	Sealed bool
}

// Form describes definition id: the custom definition when one has the
// id, else the catalog's.
func (x *Indexers) Form(ctx context.Context, _ identity.Admin, id string) (Form, error) {
	var f Form
	err := x.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		q := db.New(r)
		def, err := q.GetCustom(ctx, db.DefinitionID(id))
		switch {
		case err == nil:
			f.Entry, err = customEntry(def)
		case errors.Is(err, storage.ErrNoRows):
			f.Entry, def, err = catalogEntry(ctx, q, id)
		}
		if err != nil {
			return err
		}
		s, ok, err := summaryOf(def)
		if err != nil || !ok {
			return err
		}
		f.Links, f.Session, f.Captcha = s.Links, s.Login == cardigann.SessionLogin, s.Captcha
		for _, field := range s.Fields {
			f.Fields = append(f.Fields, FormField{Field: field, Sealed: Sealed(field)})
			f.Solver = f.Solver || field.Type == cardigann.InfoFlareSolverrField
		}
		return nil
	})
	return f, err
}

// catalogEntry is the catalog's entry id and its stored version:
// ErrNoDefinition when the catalog doesn't list it, ErrCatalogUnavailable
// before any refresh has read the catalog.
func catalogEntry(ctx context.Context, q *db.Queries, id string) (Entry, db.IndexersDefinition, error) {
	row, err := q.GetCatalogEntry(ctx, db.DefinitionID(id))
	if errors.Is(err, storage.ErrNoRows) {
		state, err := q.GetCatalogState(ctx)
		switch {
		case err != nil:
			return Entry{}, db.IndexersDefinition{}, err
		case state.RefreshedAt.IsZero():
			return Entry{}, db.IndexersDefinition{}, ErrCatalogUnavailable
		}
		return Entry{}, db.IndexersDefinition{}, ErrNoDefinition
	}
	if err != nil {
		return Entry{}, db.IndexersDefinition{}, err
	}
	def, err := q.GetDefinition(ctx, db.GetDefinitionParams{ID: row.ID, Sha: row.Sha})
	if err != nil {
		return Entry{}, db.IndexersDefinition{}, fmt.Errorf("read definition %s: %w", id, err)
	}
	e := Entry{
		ID: string(row.ID), File: row.File, Name: row.Name, Description: row.Description, Language: row.Language,
		Privacy: cardigann.Privacy(row.Privacy), Origin: FromCatalog, Reason: deref(def.Refusal), Version: def.FetchedAt.Time(),
	}
	err = decode(row.Categories, &e.Categories)
	return e, def, err
}

// SetCatalogAddress reads the schema 12 index at addr from the Controller
// and saves addr only when it reads one. nil resets the address to the
// default. The next refresh reads the whole index again.
func (x *Indexers) SetCatalogAddress(ctx context.Context, _ identity.Admin, addr *string) error {
	if addr != nil {
		base := strings.TrimSuffix(*addr, "/")
		u, err := url.Parse(base)
		if err != nil || u.Scheme != "http" && u.Scheme != "https" || u.Host == "" {
			return &CatalogUnreachableError{Err: errors.New("it isn't an http or https address")}
		}
		src := x.catalogSource(base)
		if _, err := src.fetch(ctx, "", func(IndexEntry) bool { return false }); err != nil {
			return &CatalogUnreachableError{Err: err}
		}
		addr = &base
	}
	err := x.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		return db.New(tx).SetCatalogAddress(ctx, addr)
	})
	if err != nil {
		return err
	}
	x.catalogChanged(ctx)
	return nil
}
