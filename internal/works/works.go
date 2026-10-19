// The core stores and compares each node's key and never parses it. A
// refresh keeps every id: a node its source stops listing is withdrawn,
// with a time, never deleted.
package works

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"regexp"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works/internal/db"
)

type file = storage.Library

//go:embed schema/*.sql
var schemaFiles embed.FS

func Schema() storage.Schema[file] { return storage.NewSchema[file]("works", schemaFiles) }

type (
	WorkID  = db.WorkID
	LevelID = db.LevelID
	ItemID  = db.ItemID
	TypeID  = db.TypeID
	// Key is a node's identity at its source, 1 to 200 bytes, unique among
	// one Work's levels and among its Items. A Work's key never changes.
	Key = db.Key
)

const maxKey = 200

func ParseKey(s string) (Key, error) {
	if s == "" || len(s) > maxKey {
		return "", fmt.Errorf("a key is 1 to %d bytes, not %d", maxKey, len(s))
	}
	return Key(s), nil
}

type Timings struct {
	// Fetch bounds one Search or Fetch of a metadata source.
	Fetch time.Duration `yaml:"fetch"`
}

func DefaultTimings() Timings { return Timings{Fetch: 30 * time.Second} }

type WorkChanged struct {
	WorkID string `json:"workId" format:"uuid"`
}

type ContentTypesChanged struct {
	ContentType string `json:"contentType" pattern:"^[a-z][a-z0-9]{1,15}$"`
}

var (
	ChangedEvent             = live.AdminEvent[WorkChanged]("works.changed")
	ContentTypesChangedEvent = live.EveryoneEvent[ContentTypesChanged]("works.content-types-changed")
)

type Publisher[T any] interface {
	Publish(ctx context.Context, v T)
}

var (
	// ErrNotFound and ErrUnavailable are what a Content Type wraps: its
	// source has no record with the key, or didn't answer.
	ErrNotFound    = errors.New("the metadata source has no record with that key")
	ErrUnavailable = errors.New("the metadata source didn't answer")
	ErrInvalid     = errors.New("the metadata source sent a record SlipStream can't use")
	ErrRefused     = errors.New("the metadata source refused SlipStream's key")

	ErrNoType       = errors.New("no Content Type has that id")
	ErrTypeOff      = errors.New("that Content Type is turned off")
	ErrTypeInUse    = errors.New("the Content Type still holds Works")
	ErrNoWork       = errors.New("no Work has that id")
	ErrGoneUpstream = errors.New("the metadata source no longer has this Work")
)

// SourceError is a metadata call that failed at its source. Err is
// ErrNotFound, ErrRefused, ErrUnavailable or ErrInvalid.
type SourceError struct {
	// Source names the provider in words, such as "TMDB".
	Source string
	// Until is when a rate-limited source takes calls again.
	Until time.Time
	Words string
	Err   error
}

func (e *SourceError) Error() string {
	if e.Words == "" {
		return e.Source + ": " + e.Err.Error()
	}
	return e.Source + ": " + e.Words
}

func (e *SourceError) Unwrap() error { return e.Err }

type Config struct {
	Library      *storage.DB[file]
	Catalogs     map[TypeID]Catalog
	Timings      Timings
	Changed      Publisher[WorkChanged]
	TypesChanged Publisher[ContentTypesChanged]
	Log          *slog.Logger
}

type Works struct {
	cfg   Config
	ids   []TypeID
	types map[TypeID]registered
}

type registered struct {
	catalog Catalog
	desc    Description
}

// Open describes every Content Type, panicking on a description that
// breaks a rule, and refuses a library holding Works of a type this binary
// does not register with a *storage.RefusedError.
func Open(ctx context.Context, cfg Config) (*Works, error) {
	w := &Works{cfg: cfg, ids: slices.Sorted(maps.Keys(cfg.Catalogs)), types: map[TypeID]registered{}}
	for _, id := range w.ids {
		c := cfg.Catalogs[id]
		d := c.Describe()
		if err := validate(d); err != nil {
			panic(fmt.Sprintf("works: content type %s: %v", id, err))
		}
		w.types[id] = registered{catalog: c, desc: d}
	}
	held, err := db.New(cfg.Library).HeldTypes(ctx)
	if err != nil {
		return nil, fmt.Errorf("open works: %w", err)
	}
	for _, id := range held {
		if _, ok := w.types[id]; !ok {
			return nil, &storage.RefusedError{Err: fmt.Errorf("library.db holds Works of Content Type %s, which this binary does not register", id)}
		}
	}
	return w, nil
}

var fieldKey = regexp.MustCompile(`^[a-z][a-z0-9_]{0,31}$`)

func validate(d Description) error {
	if d.Name == "" {
		return errors.New("its name is empty")
	}
	if iconNames[d.Icon] == "" {
		return errors.New("it has no icon")
	}
	nouns := append([]Noun{d.Work, d.Item}, d.Levels...)
	for _, n := range nouns {
		if n.One == "" || n.Many == "" {
			return fmt.Errorf("noun %+v needs both One and Many", n)
		}
	}
	seen := map[Tier]map[string]bool{}
	for _, f := range d.Fields {
		switch {
		case !fieldKey.MatchString(f.Key):
			return fmt.Errorf("field key %s is not %s", f.Key, fieldKey)
		case f.Label == "":
			return fmt.Errorf("field %s has no label", f.Key)
		case f.Tier != TierWork && f.Tier != TierItem && (f.Tier < 1 || int(f.Tier) > len(d.Levels)):
			return fmt.Errorf("field %s sits at %s, which the type does not declare", f.Key, f.Tier)
		case kindNames[f.Kind] == "":
			return fmt.Errorf("field %s has no kind", f.Key)
		case seen[f.Tier][f.Key]:
			return fmt.Errorf("field %s is declared twice on %s", f.Key, f.Tier)
		}
		if seen[f.Tier] == nil {
			seen[f.Tier] = map[string]bool{}
		}
		seen[f.Tier][f.Key] = true
	}
	return nil
}

func Enabled(ctx context.Context, r storage.Reader[file], t TypeID) (bool, error) {
	off, err := db.New(r).IsDisabled(ctx, t)
	return !off, err
}
