package indexers

import (
	"cmp"
	"context"
	"errors"
	"slices"

	"github.com/jatassi/slipstream/internal/cardigann"
	"github.com/jatassi/slipstream/internal/indexers/internal/db"
	"github.com/jatassi/slipstream/internal/searching"
	"github.com/jatassi/slipstream/internal/storage"
)

var (
	// ErrNotALink is a base address the definition doesn't list.
	ErrNotALink = errors.New("the definition doesn't list that address")
	// ErrCookieNeeded is an Indexer added without a browser cookie for a
	// definition that signs in with one.
	ErrCookieNeeded = errors.New("the definition signs in with a browser cookie")
)

// ValueError is a setting value the definition doesn't take.
type ValueError struct {
	Field string
	// Sentence says what the definition takes, for the form.
	Sentence string
}

func (e *ValueError) Error() string { return "setting " + e.Field + " doesn't fit the definition" }

// drafted is d as an unsaved Indexer at revision 1 with its secrets
// sealed, and the session pasted for it, nil for none.
func (x *Indexers) drafted(ctx context.Context, d Draft) (Indexer, *searching.Session, error) {
	home, err := x.homeOf(d.Home)
	if err != nil {
		return Indexer{}, nil, err
	}
	idx := Indexer{
		ID: ID{UUID: storage.NewUUID()}, Name: d.Name, Kind: cmp.Or(d.Kind, Torznab), Home: home, On: d.On, AutoSearch: d.AutoSearch, RSS: d.RSS,
		AllowFailover: d.AllowFailover, Priority: cmp.Or(d.Priority, DefaultPriority), Categories: d.Categories, Media: d.Media,
		Searches: d.Searches, Grabs: d.Grabs, Revision: 1, Created: storage.Now().Time(),
	}
	secrets := Secrets{Address: d.Address, APIKey: d.APIKey}
	if idx.Kind == Catalog {
		def, fields, err := x.catalogDraft(ctx, d)
		if err != nil {
			return Indexer{}, nil, err
		}
		idx.Source, secrets = def, Secrets{Fields: fields}
	} else {
		idx.Source, idx.FeedHost = Feed{Tracker: cmp.Or(d.Tracker, d.Name), AllowNoSize: d.AllowNoSize}, hostOf(d.Address)
	}
	if err := checkMedia(idx, x.cfg.Types); err != nil {
		return Indexer{}, nil, err
	}
	session, err := pasted(d.Cookie, siteHost(idx))
	if err != nil {
		return Indexer{}, nil, err
	}
	if idx.Secrets, err = x.seal(idx.ID, secrets); err != nil {
		return Indexer{}, nil, err
	}
	return idx, session, nil
}

// catalogDraft is the definition version d pins, with its plain settings,
// and its sealed ones.
func (x *Indexers) catalogDraft(ctx context.Context, d Draft) (Definition, map[string]string, error) {
	var row db.IndexersDefinition
	var s cardigann.Summary
	err := x.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		row, s, err = runnable(ctx, db.New(r), d.Definition)
		return err
	})
	if err != nil {
		return Definition{}, nil, err
	}
	def := Definition{ID: string(row.ID), SHA: row.Sha, BaseAddress: d.BaseAddress, Login: s.Login, Captcha: s.Captcha}
	if def.BaseAddress == "" && len(s.Links) > 0 {
		def.BaseAddress = s.Links[0]
	}
	if !slices.Contains(s.Links, def.BaseAddress) {
		return Definition{}, nil, ErrNotALink
	}
	if s.Uses.Has(cardigann.LoginCookie) && d.Cookie == nil {
		return Definition{}, nil, ErrCookieNeeded
	}
	plain, sealed, err := settingsOf(s, d.Values, nil, nil)
	if err != nil {
		return Definition{}, nil, err
	}
	def.Plain = plain
	return def, sealed, nil
}

// runnable is the version of definition id an Indexer added now pins: the
// custom definition with the id, else the catalog's. It fails with
// ErrNoDefinition, ErrCatalogUnavailable or *cardigann.Unrunnable.
func runnable(ctx context.Context, q *db.Queries, id string) (db.IndexersDefinition, cardigann.Summary, error) {
	_, row, err := lookup(ctx, q, id)
	if err != nil {
		return db.IndexersDefinition{}, cardigann.Summary{}, err
	}
	s, ok, err := summaryOf(row)
	switch {
	case err != nil:
		return db.IndexersDefinition{}, cardigann.Summary{}, err
	case !ok:
		return db.IndexersDefinition{}, cardigann.Summary{}, &cardigann.Unrunnable{Reason: deref(row.Refusal)}
	}
	return row, s, nil
}

// pinnedSummary is the summary of the version idx pins, which fails with
// *cardigann.Unrunnable when that version doesn't load.
func pinnedSummary(ctx context.Context, r storage.Reader[file], d Definition) (cardigann.Summary, error) {
	row, err := db.New(r).GetDefinition(ctx, db.GetDefinitionParams{ID: db.DefinitionID(d.ID), Sha: d.SHA})
	if err != nil {
		return cardigann.Summary{}, err
	}
	s, ok, err := summaryOf(row)
	if err == nil && !ok {
		err = &cardigann.Unrunnable{Reason: deref(row.Refusal)}
	}
	return s, err
}

// settable reports whether an Indexer of a definition with summary s
// stores a value for f. A definition that signs in with a pasted cookie
// takes it with the browser's User-Agent in place of its cookie setting.
func settable(s cardigann.Summary, f cardigann.Field) bool {
	return stored(f) && (f.Name != "cookie" || !s.Uses.Has(cardigann.LoginCookie))
}

// settingsOf splits values over s's settings into the plain ones and the
// sealed ones, on top of the plain and sealed values saved before. A
// setting values leaves out keeps its saved value, as does a sealed one
// sent empty. A setting with no value takes its default when the Node
// runs it.
func settingsOf(s cardigann.Summary, values, plainBefore, sealedBefore map[string]string) (plain, sealed map[string]string, err error) {
	plain, sealed = map[string]string{}, map[string]string{}
	byName := map[string]cardigann.Field{}
	for _, f := range s.Fields {
		if !settable(s, f) {
			continue
		}
		byName[f.Name] = f
		if v, ok := plainBefore[f.Name]; ok && !Sealed(f) {
			plain[f.Name] = v
		}
		if v, ok := sealedBefore[f.Name]; ok && Sealed(f) {
			sealed[f.Name] = v
		}
	}
	for name, v := range values {
		f, ok := byName[name]
		if !ok {
			return nil, nil, &ValueError{Field: name, Sentence: "The definition has no setting named " + name + "."}
		}
		if err := fits(f, v); err != nil {
			return nil, nil, err
		}
		switch {
		case !Sealed(f):
			plain[name] = v
		case v != "":
			sealed[name] = v
		}
	}
	return plain, sealed, nil
}

// fits refuses v for a select that doesn't offer it and a checkbox that
// isn't true or false.
func fits(f cardigann.Field, v string) error {
	switch f.Type {
	case cardigann.SelectField:
		if !slices.ContainsFunc(f.Options, func(o cardigann.Option) bool { return o.Value == v }) {
			return &ValueError{Field: f.Name, Sentence: "Choose one of the options for " + f.Label + "."}
		}
	case cardigann.CheckboxField:
		if v != "true" && v != "false" {
			return &ValueError{Field: f.Name, Sentence: f.Label + " is true or false."}
		}
	case cardigann.TextField, cardigann.PasswordField, cardigann.InfoField, cardigann.InfoCookieField, cardigann.InfoFlareSolverrField,
		cardigann.InfoUserAgentField, cardigann.InfoCategory8000Field, cardigann.CaptchaField:
	}
	return nil
}
