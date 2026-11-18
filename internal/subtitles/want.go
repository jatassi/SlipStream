package subtitles

import (
	"fmt"
	"slices"
	"strings"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/language"
	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/subtitling"
	"github.com/jatassi/slipstream/internal/targets"
)

// Want is a Wanted Language's language and kind: full, or forced only.
type Want struct {
	Language language.Code
	Forced   bool
}

const forcedSuffix = "-forced"

// String is w's URL form: "en", "pt-BR", "en-forced".
func (w Want) String() string {
	if w.Forced {
		return w.Language.String() + forcedSuffix
	}
	return w.Language.String()
}

// key is w as the embedded column lists it: ",en-forced,".
func (w Want) key() string { return "," + w.String() + "," }

// ParseWant reads a want's URL form.
func ParseWant(s string) (Want, error) {
	tag, forced := strings.CutSuffix(s, forcedSuffix)
	l, err := language.Parse(tag)
	if err != nil || l.String() != tag {
		return Want{}, fmt.Errorf("%q is not a subtitle language and kind", s)
	}
	return Want{Language: l, Forced: forced}, nil
}

// WantKey is one want of one library file, a subtitles_want row's key.
type WantKey struct {
	File files.ID
	Want
}

// StateKind is what a want reads as.
type StateKind uint8

const (
	// NotNeeded is a want whose condition doesn't hold: audio in the
	// language, or a full want of the language that applies.
	NotNeeded StateKind = iota + 1
	// Embedded is a want a text track in the file meets.
	Embedded
	// Met is a want a Sidecar meets.
	Met
	// Missing is a want nothing meets yet.
	Missing
)

// NotNeededWhy is why a want is Not needed.
type NotNeededWhy uint8

const (
	AudioInLanguage NotNeededWhy = iota + 1
	FullCovers
)

// facts is what a file's wants read of it, the columns putFacts copies to
// subtitles_file. A file with no Kind has no subtitles.
type facts struct {
	Kind     subtitling.Kind
	Wantless bool
	// Audio is the languages of its tagged audio tracks.
	Audio []release.Language
	// Embedded is each want a text track meets and the track's ordinal in
	// the media record's subtitle tracks, ignored tracks left out.
	Embedded map[Want]int
}

// ignore is a "Don't count this track" row: the track's ordinal and its
// fingerprint.
type ignore struct {
	Ordinal  int
	Codec    string
	Language release.Language
	Forced   bool
}

// holds is whether the ignore still names the track at its ordinal.
func (i ignore) holds(text []media.Subtitle) bool {
	if i.Ordinal >= len(text) {
		return false
	}
	t := text[i.Ordinal]
	return t.Codec == i.Codec && t.Language == i.Language && t.Forced == i.Forced
}

// factsOf is s's facts as wants read them, kind its type's.
func factsOf(s files.Subject, kind subtitling.Kind, ignored []ignore) facts {
	f := facts{Kind: kind, Wantless: s.Kept || s.Gone, Embedded: map[Want]int{}}
	for _, a := range s.Audio {
		if a != "" && !slices.Contains(f.Audio, a) {
			f.Audio = append(f.Audio, a)
		}
	}
	slices.Sort(f.Audio)
	for i, t := range s.Text {
		if t.Image || slices.ContainsFunc(ignored, func(ig ignore) bool { return ig.Ordinal == i && ig.holds(s.Text) }) {
			continue
		}
		l, err := language.Parse(string(t.Language))
		if err != nil {
			continue
		}
		if _, ok := f.Embedded[Want{l, t.Forced}]; !ok {
			f.Embedded[Want{l, t.Forced}] = i
		}
	}
	return f
}

// audioColumn and embeddedColumn are f's lists as subtitles_file stores
// them: ",en,fr," and ",en,en-forced,".
func (f facts) audioColumn() string {
	parts := make([]string, len(f.Audio))
	for i, a := range f.Audio {
		parts[i] = string(a)
	}
	return listColumn(parts)
}

func (f facts) embeddedColumn() string {
	parts := make([]string, 0, len(f.Embedded))
	for w := range f.Embedded {
		parts = append(parts, w.String())
	}
	slices.Sort(parts)
	return listColumn(parts)
}

func listColumn(parts []string) string {
	if len(parts) == 0 {
		return ","
	}
	return "," + strings.Join(parts, ",") + ","
}

// want is one of a file's wants and what it reads as from facts and the
// file's Sidecar rows alone.
type want struct {
	targets.WantedLanguage
	Kind  StateKind
	Why   NotNeededWhy
	Track int
}

func (w want) Want() Want { return Want{Language: w.Language, Forced: w.Forced} }

// searchable is whether the want needs a row: Missing, or Met by a Sidecar.
func (w want) searchable() bool { return w.Kind == Missing || w.Kind == Met }

// wantsOf is every want langs make of a file with f, in Profile order. A
// file with no subtitles, a Kept one and a gone one have none. met is the
// wants a Sidecar row of the file meets.
func wantsOf(f facts, langs []targets.WantedLanguage, met func(Want) bool) []want {
	if f.Kind == 0 || f.Wantless {
		return nil
	}
	out := make([]want, 0, len(langs))
	for _, l := range langs {
		w := want{WantedLanguage: l}
		track, embedded := f.Embedded[w.Want()]
		switch {
		case audible(f, l):
			w.Kind, w.Why = NotNeeded, AudioInLanguage
		case l.Forced && slices.ContainsFunc(langs, func(o targets.WantedLanguage) bool {
			return !o.Forced && o.Language == l.Language && !audible(f, o)
		}):
			w.Kind, w.Why = NotNeeded, FullCovers
		case embedded:
			w.Kind, w.Track = Embedded, track
		case met(w.Want()):
			w.Kind = Met
		default:
			w.Kind = Missing
		}
		out = append(out, w)
	}
	return out
}

// audible is whether l is wanted only without audio in its language, and
// f has audio in it.
func audible(f facts, l targets.WantedLanguage) bool {
	return l.Audio == targets.NoAudio && l.Language.Spoken(f.Audio)
}
