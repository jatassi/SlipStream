package targets

import (
	"context"
	"fmt"
	"slices"

	"github.com/jatassi/slipstream/internal/language"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets/internal/db"
	"github.com/jatassi/slipstream/internal/works"
)

// HI is how a Wanted Language ranks hearing-impaired subtitles: Avoid
// takes one only when nothing else fits, Prefer takes one first.
type HI = db.HI

const (
	Avoid  = db.Avoid
	Allow  = db.Allow
	Prefer = db.Prefer
)

// AudioCondition is when a Wanted Language is wanted: Always, or NoAudio,
// only when no audio track is in the language.
type AudioCondition = db.AudioCondition

const (
	Always  = db.Always
	NoAudio = db.NoAudio
)

// WantedLanguage is one of a Profile's subtitle wants. A Forced one wants
// only the lines in another language, such as signs.
type WantedLanguage struct {
	Language language.Code
	Forced   bool
	HI       HI
	Audio    AudioCondition
}

// NoSubtitlesError is a Profile listing Wanted Languages for a Content
// Type that has no subtitles. Type is the type's name.
type NoSubtitlesError struct{ Type string }

func (e *NoSubtitlesError) Error() string {
	return e.Type + " has no subtitles, so its Profiles can't list subtitle languages"
}

// DuplicateLanguageError is a spec listing two Wanted Languages of one
// language and kind. Index is the second.
type DuplicateLanguageError struct {
	Index  int
	Wanted WantedLanguage
}

func (e *DuplicateLanguageError) Error() string {
	return fmt.Sprintf("a Profile lists %s %s subtitles twice", kindWord(e.Wanted.Forced), e.Wanted.Language.Name())
}

func kindWord(forced bool) string {
	if forced {
		return "forced-only"
	}
	return "full"
}

// checkLanguages refuses Wanted Languages that typ can't take or that
// name one language and kind twice.
func (t *Targets) checkLanguages(typ works.TypeID, ls []WantedLanguage) error {
	if len(ls) > 0 && !t.cfg.Subtitled(typ) {
		d, _ := t.cfg.Works.Describe(typ)
		return &NoSubtitlesError{Type: d.Name}
	}
	for i, l := range ls {
		if slices.ContainsFunc(ls[:i], func(o WantedLanguage) bool { return o.Language == l.Language && o.Forced == l.Forced }) {
			return &DuplicateLanguageError{Index: i, Wanted: l}
		}
	}
	return nil
}

type languageKey struct {
	language string
	forced   bool
}

// writeLanguages replaces Profile id's Wanted Languages with ls. One the
// Profile already listed keeps when it was added.
func writeLanguages(ctx context.Context, q *db.Queries, id ProfileID, ls []WantedLanguage) error {
	rows, err := q.ListLanguagesOfProfile(ctx, id)
	if err != nil {
		return err
	}
	added := make(map[languageKey]storage.Time, len(rows))
	for _, row := range rows {
		added[languageKey{row.Language, row.Forced}] = row.AddedAt
	}
	if err := q.DeleteProfileLanguages(ctx, id); err != nil {
		return err
	}
	now := storage.Now()
	for i, l := range ls {
		at, ok := added[languageKey{l.Language.String(), l.Forced}]
		if !ok {
			at = now
		}
		err := q.InsertProfileLanguage(ctx, db.InsertProfileLanguageParams{
			ProfileID: id, Position: int64(i + 1), Language: l.Language.String(), Forced: l.Forced, Hi: l.HI, AudioCondition: l.Audio, AddedAt: at,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func wantedOf(row db.TargetsProfileLanguage) (WantedLanguage, error) {
	c, err := language.Parse(row.Language)
	if err != nil {
		return WantedLanguage{}, fmt.Errorf("read the Wanted Languages of Profile %s: %w", row.ProfileID, err)
	}
	return WantedLanguage{Language: c, Forced: row.Forced, HI: row.Hi, Audio: row.AudioCondition}, nil
}

// LanguagesIn is Profile p's Wanted Languages in r, in the Admin's order.
func (t *Targets) LanguagesIn(ctx context.Context, r storage.Reader[file], p ProfileID) ([]WantedLanguage, error) {
	rows, err := db.New(r).ListLanguagesOfProfile(ctx, p)
	if err != nil {
		return nil, err
	}
	out := make([]WantedLanguage, len(rows))
	for i, row := range rows {
		if out[i], err = wantedOf(row); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// LanguagesOf is every Profile's Wanted Languages in r. A Profile that
// lists none has no entry.
func (t *Targets) LanguagesOf(ctx context.Context, r storage.Reader[file]) (map[ProfileID][]WantedLanguage, error) {
	return languagesOf(ctx, db.New(r))
}

func languagesOf(ctx context.Context, q *db.Queries) (map[ProfileID][]WantedLanguage, error) {
	rows, err := q.ListProfileLanguages(ctx)
	if err != nil {
		return nil, err
	}
	out := map[ProfileID][]WantedLanguage{}
	for _, row := range rows {
		l, err := wantedOf(row)
		if err != nil {
			return nil, err
		}
		out[row.ProfileID] = append(out[row.ProfileID], l)
	}
	return out, nil
}

// Wanting is the Profile a Work wants from and the Profile's Content
// Type.
type Wanting struct {
	Profile ProfileID
	Type    works.TypeID
}

// WantingOf reads, in r, the Profile each of ws wants from.
func WantingOf(ctx context.Context, r storage.Reader[file], ws []works.WorkID) (map[works.WorkID]Wanting, error) {
	rows, err := db.New(r).ListProfilesOfWorks(ctx, ws)
	if err != nil {
		return nil, err
	}
	out := make(map[works.WorkID]Wanting, len(rows))
	for _, row := range rows {
		out[row.WorkID] = Wanting{Profile: row.ProfileID, Type: row.ContentType}
	}
	return out, nil
}
