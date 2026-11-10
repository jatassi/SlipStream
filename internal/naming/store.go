package naming

import (
	"context"
	"embed"
	"errors"

	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/naming/internal/db"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

//go:embed schema/*.sql
var schemaFiles embed.FS

// Schema is each Content Type's stored naming settings in library.db.
func Schema() storage.Schema[storage.Library] {
	return storage.NewSchema[storage.Library]("naming", schemaFiles)
}

// Changed names the Content Type whose naming settings an Admin saved.
type Changed struct {
	ContentType string `json:"contentType" pattern:"^[a-z][a-z0-9]{1,15}$"`
}

var ChangedEvent = live.AdminEvent[Changed]("naming.changed")

type Publisher interface {
	Publish(ctx context.Context, c Changed)
}

type Config struct {
	Library *storage.DB[storage.Library]
	Facets  map[works.TypeID]Facet
	// Describe is a type's Description, whose nouns label its folder
	// settings.
	Describe func(works.TypeID) (works.Description, bool)
	// Budget is the smallest path budget among t's roots, which the save
	// check names samples within: roots.SmallestBudget.
	Budget  func(ctx context.Context, r storage.Reader[storage.Library], t works.TypeID) (int, error)
	Changed Publisher
}

// Naming owns each Content Type's naming: the settings an Admin stores,
// the scheme its files are named under and the values its Works name them
// with.
type Naming struct{ cfg Config }

func New(cfg Config) *Naming { return &Naming{cfg: cfg} }

// SchemeIn is t's scheme as stored, in r.
func (n *Naming) SchemeIn(ctx context.Context, r storage.Reader[storage.Library], t works.TypeID) (*Scheme, error) {
	s, err := storedIn(ctx, r, t)
	if err != nil {
		return nil, err
	}
	sc, err := newScheme(n.cfg.Facets[t].Naming(), s)
	return sc, keyed(t, err)
}

// DefaultScheme is the scheme d's templates make under the default
// switches, as a type with nothing stored names its files.
func DefaultScheme(d Declaration) (*Scheme, error) { return newScheme(d, Settings{}) }

// DraftIn is t's scheme with d applied over what is stored in r, unsaved,
// once the save check passes it. A key t has no setting for is
// *setting.UnknownKeyError, and every value its kind refuses and every
// problem the check finds is a Problem of one *RefusedError.
func (n *Naming) DraftIn(ctx context.Context, r storage.Reader[storage.Library], t works.TypeID, d setting.Patch) (*Scheme, error) {
	sc, _, err := n.draft(ctx, r, t, d)
	return sc, err
}

// ValuesIn reads ws's values through t's facet, in r.
func (n *Naming) ValuesIn(ctx context.Context, r storage.Reader[storage.Library], t works.TypeID, ws works.Among[works.WorkID]) (map[works.WorkID]Values, error) {
	return n.cfg.Facets[t].NamingValues(ctx, r, ws)
}

// draft is DraftIn with the Settings the scheme was made from, for Apply
// to store.
func (n *Naming) draft(ctx context.Context, r storage.Reader[storage.Library], t works.TypeID, d setting.Patch) (*Scheme, Settings, error) {
	ks := n.keysOf(t)
	if err := ks.known(d); err != nil {
		return nil, Settings{}, err
	}
	s, err := storedIn(ctx, r, t)
	if err != nil {
		return nil, Settings{}, err
	}
	problems, err := ks.patch(&s, d)
	if err != nil {
		return nil, Settings{}, err
	}
	if problems != nil {
		return nil, Settings{}, refusal(t, problems)
	}
	sc, err := newScheme(n.cfg.Facets[t].Naming(), s)
	if err != nil {
		return nil, Settings{}, keyed(t, err)
	}
	budget, err := n.cfg.Budget(ctx, r, t)
	if err != nil {
		return nil, Settings{}, err
	}
	if problems = check(sc, budget); problems != nil {
		return nil, Settings{}, refusal(t, problems)
	}
	return sc, s, nil
}

// refusal is a *RefusedError of problems with each Key the setting key,
// nil for none.
func refusal(t works.TypeID, problems []Problem) error {
	if problems == nil {
		return nil
	}
	return &RefusedError{Problems: prefixed(t, problems)}
}

// prefixed is problems with each Key the setting key.
func prefixed(t works.TypeID, problems []Problem) []Problem {
	var out []Problem
	for _, p := range problems {
		out = append(out, Problem{Key: prefixOf(t) + p.Key, Sentence: p.Sentence})
	}
	return out
}

// keyed is err with a *RefusedError's keys the setting keys.
func keyed(t works.TypeID, err error) error {
	if refused, ok := errors.AsType[*RefusedError](err); ok {
		return refusal(t, refused.Problems)
	}
	return err
}

func prefixOf(t works.TypeID) string { return string(t) + ".naming." }

// storedIn is t's stored overrides in r.
func storedIn(ctx context.Context, r storage.Reader[storage.Library], t works.TypeID) (Settings, error) {
	q := db.New(r)
	row, err := q.GetType(ctx, t)
	if err != nil && !errors.Is(err, storage.ErrNoRows) {
		return Settings{}, err
	}
	templates, err := q.ListTemplates(ctx, t)
	if err != nil {
		return Settings{}, err
	}
	s := Settings{
		Rename:       row.RenameFiles,
		Templates:    make(map[string]*string, len(templates)),
		Colon:        fromWord(colonWords, row.Colon),
		ColonCustom:  row.ColonCustom,
		MultiEpisode: fromWord(multiWords, row.MultiEpisode),
	}
	for _, tpl := range templates {
		s.Templates[tpl.Field] = &tpl.Template
	}
	return s, nil
}

// store replaces t's stored overrides with s.
func store(ctx context.Context, tx *storage.Tx[storage.Library], t works.TypeID, s Settings) error {
	q := db.New(tx)
	err := q.PutType(ctx, db.PutTypeParams{
		ContentType: t, RenameFiles: s.Rename, Colon: toWord(colonWords, s.Colon),
		ColonCustom: s.ColonCustom, MultiEpisode: toWord(multiWords, s.MultiEpisode),
	})
	if err != nil {
		return err
	}
	if err := q.ClearTemplates(ctx, t); err != nil {
		return err
	}
	for field, text := range s.Templates {
		if text == nil {
			continue
		}
		if err := q.PutTemplate(ctx, db.PutTemplateParams{ContentType: t, Field: field, Template: *text}); err != nil {
			return err
		}
	}
	return nil
}
