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
	"github.com/jatassi/slipstream/internal/metadata/artwork"
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

type TypeChanged struct {
	ContentType string `json:"contentType" pattern:"^[a-z][a-z0-9]{1,15}$"`
}

var (
	ChangedEvent             = live.AdminEvent[WorkChanged]("works.changed")
	ContentTypesChangedEvent = live.EveryoneEvent[ContentTypesChanged]("works.content-types-changed")
	// TypeChangedEvent means every Work of the type may read differently,
	// so a page refetches the type's Works instead of hearing one
	// works.changed per Work.
	TypeChangedEvent = live.AdminEvent[TypeChanged]("works.type-changed")
)

type Publisher[T any] interface {
	Publish(ctx context.Context, v T)
}

var (
	// ErrNotFound and ErrUnavailable are what a Content Type wraps: its
	// source has no record with the key, or didn't answer.
	ErrNotFound    = errors.New("the metadata source has no record with that key")
	ErrUnavailable = errors.New("the metadata source didn't respond")
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
	Library  *storage.DB[file]
	Catalogs map[TypeID]Catalog
	Source   Source
	// Artwork fetches the images a new Work's kinds prefetch, after the add
	// commits.
	Artwork interface {
		Prefetch(wants []artwork.Want)
	}
	Timings      Timings
	Changed      Publisher[WorkChanged]
	TypesChanged Publisher[ContentTypesChanged]
	TypeChanged  Publisher[TypeChanged]
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
			return nil, &storage.RefusedError{Err: fmt.Errorf("library.db holds Works of Content Type %s, which this binary doesn't register", id)}
		}
	}
	return w, nil
}

var fieldKey = regexp.MustCompile(`^[a-z][a-z0-9_]{0,31}$`)

func validate(d Description) error {
	switch {
	case d.Name == "":
		return errors.New("its name is empty")
	case iconNames[d.Icon] == "":
		return errors.New("it has no icon")
	case d.Source == "":
		return errors.New("it names no source")
	case itemsNames[d.Items] == "":
		return errors.New("it has no Items shape")
	case d.Items == ItemsOne && len(d.Levels) > 0:
		return errors.New("it holds one Item and declares levels")
	}
	nouns := append([]Noun{d.Work, d.Item}, d.Levels...)
	for _, n := range nouns {
		if n.One == "" || n.Many == "" {
			return fmt.Errorf("noun %+v needs both One and Many", n)
		}
	}
	declared, err := validateFields(d)
	if err != nil {
		return err
	}
	for _, err := range []error{validateArtwork(d.Artwork), validatePlaces(d.Fields), validateFieldRefs(d, declared), validateArt(d), validateRatings(d.Ratings)} {
		if err != nil {
			return err
		}
	}
	return nil
}

// validateFields answers each tier's fields by key.
func validateFields(d Description) (map[Tier]map[string]FieldSpec, error) {
	declared := map[Tier]map[string]FieldSpec{}
	for _, f := range d.Fields {
		switch {
		case !fieldKey.MatchString(f.Key):
			return nil, fmt.Errorf("field key %s is not %s", f.Key, fieldKey)
		case f.Label == "":
			return nil, fmt.Errorf("field %s has no label", f.Key)
		case f.Tier != TierWork && f.Tier != TierItem && (f.Tier < 1 || int(f.Tier) > len(d.Levels)):
			return nil, fmt.Errorf("field %s sits at %s, which the type does not declare", f.Key, f.Tier)
		case kindNames[f.Kind] == "":
			return nil, fmt.Errorf("field %s has no kind", f.Key)
		}
		if _, twice := declared[f.Tier][f.Key]; twice {
			return nil, fmt.Errorf("field %s is declared twice on %s", f.Key, f.Tier)
		}
		if declared[f.Tier] == nil {
			declared[f.Tier] = map[string]FieldSpec{}
		}
		declared[f.Tier][f.Key] = f
	}
	return declared, nil
}

// validateFieldRefs holds each field's list flags, With and Art to what
// the type declares.
func validateFieldRefs(d Description, declared map[Tier]map[string]FieldSpec) error {
	for _, f := range d.Fields {
		if err := validateListing(f); err != nil {
			return err
		}
		if with, ok := declared[f.Tier][f.With]; f.With != "" && (!ok || with.Kind != KindText || f.With == f.Key) {
			return fmt.Errorf("field %s is shown with %s, which is no other text field of %s", f.Key, f.With, f.Tier)
		}
		if f.Art != "" && !hasWorkKind(d.Artwork, f.Art, "") {
			return fmt.Errorf("field %s shows art %s, which is no Work-tier artwork kind", f.Key, f.Art)
		}
	}
	return nil
}

func validateArt(d Description) error {
	for _, role := range []struct {
		name, key string
		shape     artwork.Shape
	}{{"poster", d.Art.Poster, artwork.ShapePoster}, {"backdrop", d.Art.Backdrop, artwork.ShapeBackdrop}, {"title", d.Art.Title, artwork.ShapeLogo}} {
		if role.key != "" && !hasWorkKind(d.Artwork, role.key, role.shape) {
			return fmt.Errorf("its %s art %s is no Work-tier artwork kind of shape %s", role.name, role.key, role.shape)
		}
	}
	return nil
}

func validateRatings(sources []RatingSource) error {
	for i, r := range sources {
		if _, ok := RatingSources[r]; !ok {
			return fmt.Errorf("it shows ratings from %s, which RatingSources doesn't list", r)
		}
		if slices.Contains(sources[:i], r) {
			return fmt.Errorf("it shows ratings from %s twice", r)
		}
	}
	return nil
}

// validatePlaces holds every field to a place that takes it: tagline and
// overview take one text field each, credits take people only and facts
// take no people.
func validatePlaces(fields []FieldSpec) error {
	held := map[Place]string{}
	for _, f := range fields {
		if len(f.In) == 0 {
			return fmt.Errorf("field %s is in no place", f.Key)
		}
		for i, p := range f.In {
			single := p == PlaceTagline || p == PlaceOverview
			switch {
			case placeNames[p] == "":
				return fmt.Errorf("field %s is in place %d, which doesn't exist", f.Key, p)
			case slices.Contains(f.In[:i], p):
				return fmt.Errorf("field %s is in %s twice", f.Key, p)
			case single && f.Kind != KindText:
				return fmt.Errorf("field %s is %s, and %s takes text only", f.Key, f.Kind, p)
			case single && held[p] != "":
				return fmt.Errorf("fields %s and %s are both in %s, which takes one", held[p], f.Key, p)
			case p == PlaceCredits && f.Kind != KindPeople:
				return fmt.Errorf("field %s is %s, and credits takes people only", f.Key, f.Kind)
			case p == PlaceFacts && f.Kind == KindPeople:
				return fmt.Errorf("field %s is people, which facts doesn't take", f.Key)
			}
			if single {
				held[p] = f.Key
			}
		}
	}
	return nil
}

func validateListing(f FieldSpec) error {
	switch {
	case f.List&^listAll != 0:
		return fmt.Errorf("field %s has list flags %b, which aren't all flags", f.Key, f.List)
	case f.List == 0:
		return nil
	case f.Tier != TierWork:
		return fmt.Errorf("field %s sits at %s and is listed, and only Work-tier fields are", f.Key, f.Tier)
	case f.Kind == KindPeople || f.Kind == KindLink:
		return fmt.Errorf("field %s is %s, which the Library can't list", f.Key, f.Kind)
	case f.List.Has(ListSort) && f.Kind == KindTexts:
		return fmt.Errorf("field %s is texts, which the Library can't sort by", f.Key)
	}
	return nil
}

// hasWorkKind is whether kinds holds a Work-tier kind key, of shape when
// shape is set.
func hasWorkKind(kinds []artwork.Kind, key string, shape artwork.Shape) bool {
	return slices.ContainsFunc(kinds, func(k artwork.Kind) bool {
		return !k.Item && k.Key == key && (shape == "" || k.Shape == shape)
	})
}

func validateArtwork(kinds []artwork.Kind) error {
	seen := map[bool]map[string]bool{false: {}, true: {}}
	for _, k := range kinds {
		_, shapeErr := artwork.ParseShape(string(k.Shape))
		switch {
		case !fieldKey.MatchString(k.Key):
			return fmt.Errorf("artwork kind %s is not %s", k.Key, fieldKey)
		case k.Label == "":
			return fmt.Errorf("artwork kind %s has no label", k.Key)
		case shapeErr != nil:
			return fmt.Errorf("artwork kind %s: %w", k.Key, shapeErr)
		case seen[k.Item][k.Key]:
			return fmt.Errorf("artwork kind %s is declared twice", k.Key)
		}
		if k.Prefetch != "" {
			if _, err := artwork.ParseSize(string(k.Prefetch)); err != nil {
				return fmt.Errorf("artwork kind %s: %w", k.Key, err)
			}
		}
		seen[k.Item][k.Key] = true
	}
	return nil
}

func Enabled(ctx context.Context, r storage.Reader[file], t TypeID) (bool, error) {
	off, err := db.New(r).IsDisabled(ctx, t)
	return !off, err
}
