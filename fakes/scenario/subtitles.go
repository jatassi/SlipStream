package scenario

import (
	"slices"
	"strings"
	"time"

	sublanguage "github.com/jatassi/slipstream/internal/language"
)

// SubtitleProviders is what fake-subdl, fake-subsource and fake-gestdown
// serve.
type SubtitleProviders struct {
	Catalog []CatalogSubtitle `yaml:"catalog,omitempty"`
}

// CatalogSubtitle is one subtitle a provider fake lists for a Work of the
// Scenario's metadata catalogs: a tmdb film by its title, or episodes of
// a tvdb series by its name. Several episodes are a season pack. Its cues
// follow the speech of the library files Sources names, one for a film
// and one per episode, through Offset; with no Sources they follow two
// minutes of speech seeded by ID.
type CatalogSubtitle struct {
	ID       string `jsonschema:"required,pattern=^[a-z0-9][a-z0-9-]*$"                       yaml:"id"`
	Provider string `jsonschema:"required,enum=subdl,enum=subsource,enum=gestdown"              yaml:"provider"`
	Work     string `jsonschema:"required"                                                    yaml:"work"`
	Season   int    `jsonschema:"minimum=0"                                                   yaml:"season,omitempty"`
	Episodes []int  `yaml:"episodes,omitempty"`
	// Language is an ISO 639-1 code, or pt-BR.
	Language string `jsonschema:"required"                                 yaml:"language"`
	HI       bool   `yaml:"hi,omitempty"`
	// AI is a machine translation.
	AI      bool   `yaml:"ai,omitempty"`
	Release string `jsonschema:"required"                                 yaml:"release"`
	Format  string `jsonschema:"enum=srt,enum=ass,enum=ssa,enum=sup"       yaml:"format,omitempty"`
	// Sources are library files as <folder>/<path>.
	Sources []string      `yaml:"sources,omitempty"`
	Offset  time.Duration `yaml:"offset,omitempty"`
	// Unaired is listed by fake-gestdown only once the gestdown-air cue
	// airs it, and a search before then stays not found for 12 h.
	Unaired bool `yaml:"unaired,omitempty"`
}

// MediaSpeech is speech at seeded times over Length: runs of 0.3 to 4 s,
// Density of the whole.
type MediaSpeech struct {
	Seed    uint64        `jsonschema:"required"                         yaml:"seed"`
	Density float64       `jsonschema:"required,minimum=0.05,maximum=0.9" yaml:"density"`
	Length  time.Duration `jsonschema:"required"                         yaml:"length"`
}

// Speech lengths bound a speech track, so a fixture stays minutes long.
const (
	MinSpeech = 30 * time.Second
	MaxSpeech = 300 * time.Second
)

var subtitleFormats = []string{"", "srt", "ass", "ssa", "sup"}

func (s Scenario) checkSubtitleProviders(add func(format string, args ...any)) {
	files := s.speechFiles()
	var ids []string
	for _, c := range s.SubtitleProviders.Catalog {
		at := "subtitle_providers catalog " + c.ID
		if slices.Contains(ids, c.ID) {
			add("%s is listed twice", at)
		}
		ids = append(ids, c.ID)
		if !slices.Contains([]string{"subdl", "subsource", "gestdown"}, c.Provider) {
			add("%s provider is %q, not subdl, subsource or gestdown", at, c.Provider)
		}
		if _, err := sublanguage.Parse(c.Language); err != nil {
			add("%s language: %v", at, err)
		}
		if !slices.Contains(subtitleFormats, c.Format) {
			add("%s format is %q, not srt, ass, ssa or sup", at, c.Format)
		}
		if c.Unaired && c.Provider != "gestdown" {
			add("%s is unaired, which only gestdown plays", at)
		}
		members := s.checkSubtitleWork(add, at, c)
		for _, src := range c.Sources {
			if !files[src] {
				add("%s source %s is no library file with a speech track", at, src)
			}
		}
		if len(c.Sources) > 0 && len(c.Sources) != members {
			add("%s has %d sources for %d files", at, len(c.Sources), members)
		}
	}
}

// checkSubtitleWork checks the Work c names and answers how many files
// its download holds.
func (s Scenario) checkSubtitleWork(add func(format string, args ...any), at string, c CatalogSubtitle) int {
	if s.TMDB != nil && slices.ContainsFunc(s.TMDB.Movies, func(m TMDBMovie) bool { return m.Title == c.Work }) {
		if c.Season != 0 || len(c.Episodes) > 0 || c.Provider == "gestdown" {
			add("%s is for the film %s, which takes no season or episodes and no gestdown", at, c.Work)
		}
		return 1
	}
	var series *TVDBSeries
	if s.TVDB != nil {
		if i := slices.IndexFunc(s.TVDB.Series, func(t TVDBSeries) bool { return t.Name == c.Work }); i >= 0 {
			series = &s.TVDB.Series[i]
		}
	}
	if series == nil {
		add("%s work %q is no tmdb movie title and no tvdb series name", at, c.Work)
		return 1
	}
	if series.IMDb == "" && c.Provider != "gestdown" {
		add("%s: %s finds a series by its IMDb id, and tvdb series %s has none", at, c.Provider, c.Work)
	}
	i := slices.IndexFunc(series.Seasons, func(t TVDBSeason) bool { return t.Number == c.Season })
	if c.Season < 1 || i < 0 || len(c.Episodes) == 0 {
		add("%s names no episodes of a season of %s", at, c.Work)
		return 1
	}
	for _, e := range c.Episodes {
		if !slices.ContainsFunc(series.Seasons[i].Episodes, func(t TVDBEpisode) bool { return t.Number == e }) {
			add("%s episode %d is not in %s season %d", at, e, c.Work, c.Season)
		}
	}
	return len(c.Episodes)
}

// speechFiles are the library files, as <folder>/<path>, whose audio has
// a speech track.
func (s Scenario) speechFiles() map[string]bool {
	out := map[string]bool{}
	if s.Library == nil {
		return out
	}
	for _, f := range s.Library.Folders {
		for _, m := range f.Files {
			if m.SpeechTrack() != nil {
				out[f.Folder+"/"+m.Path] = true
			}
		}
	}
	return out
}

// SpeechTrack is the speech of m's first speech track, nil when it has
// none.
func (m MediaFile) SpeechTrack() *MediaSpeech {
	for _, a := range m.Audio {
		if a.Speech != nil {
			return a.Speech
		}
	}
	return nil
}

// LibraryFile is the library file <folder>/<path> names.
func (s Scenario) LibraryFile(name string) (MediaFile, bool) {
	folder, file, _ := strings.Cut(name, "/")
	if s.Library == nil {
		return MediaFile{}, false
	}
	for _, f := range s.Library.Folders {
		if f.Folder != folder {
			continue
		}
		if i := slices.IndexFunc(f.Files, func(m MediaFile) bool { return m.Path == file }); i >= 0 {
			return f.Files[i], true
		}
	}
	return MediaFile{}, false
}

func (a MediaAudio) checkSpeech(add func(format string, args ...any), at string, i int) {
	switch {
	case a.Speech == nil && a.Codec == "pcm_s16le":
		add("%s audio %d is pcm_s16le, which only a speech track takes", at, i+1)
	case a.Speech == nil:
	case a.Codec != "pcm_s16le":
		add("%s audio %d has speech, which only pcm_s16le carries", at, i+1)
	case a.Speech.Length < MinSpeech || a.Speech.Length > MaxSpeech:
		add("%s audio %d speech lasts %v; it lasts %v to %v", at, i+1, a.Speech.Length, MinSpeech, MaxSpeech)
	case a.Speech.Density < 0.05 || a.Speech.Density > 0.9:
		add("%s audio %d speech density is %g; it is 0.05 to 0.9", at, i+1, a.Speech.Density)
	}
}
