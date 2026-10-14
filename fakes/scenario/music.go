package scenario

import (
	"regexp"
	"slices"
	"strconv"
	"time"
)

type Music struct {
	Artists []MusicArtist `jsonschema:"required,minItems=1" yaml:"artists"`
	Faults  []MusicFault  `yaml:"faults,omitempty"`
}

// MusicArtist is one artist as its source has it at each lookup: the k-th
// lookup serves revision k, and the last revision from then on.
type MusicArtist struct {
	ID        string          `jsonschema:"required,pattern=^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$" yaml:"id"`
	Revisions []MusicRevision `jsonschema:"required,minItems=1"                                                             yaml:"revisions"`
}

type MusicRevision struct {
	Name           string       `jsonschema:"required" yaml:"name"`
	SortName       string       `jsonschema:"required" yaml:"sort_name"`
	Country        string       `yaml:"country,omitempty"`
	Disambiguation string       `yaml:"disambiguation,omitempty"`
	Albums         []MusicAlbum `yaml:"albums,omitempty"`
}

type MusicAlbum struct {
	ID           string       `jsonschema:"required,pattern=^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$" yaml:"id"`
	Release      string       `jsonschema:"required,pattern=^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$" yaml:"release"`
	Title        string       `jsonschema:"required"                                                                        yaml:"title"`
	Kind         string       `jsonschema:"required,enum=Album,enum=EP,enum=Single"                                          yaml:"kind"`
	FirstRelease string       `jsonschema:"pattern=^[0-9]{4}(-[0-9]{2}(-[0-9]{2})?)?$"                                      yaml:"first_release,omitempty"`
	Tracks       []MusicTrack `yaml:"tracks,omitempty"`
}

// MusicTrack is one track of an album's release, in disc order. A zero
// length is unknown.
type MusicTrack struct {
	Recording string        `jsonschema:"required,pattern=^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$" yaml:"recording"`
	Title     string        `jsonschema:"required"                                                                        yaml:"title"`
	Length    time.Duration `yaml:"length,omitempty"`
	Disc      int           `jsonschema:"required,minimum=1"                                                              yaml:"disc"`
}

// MusicFault makes the first Times lookups of an artist answer Answer.
type MusicFault struct {
	Artist string `jsonschema:"required"                                          yaml:"artist"`
	Answer string `jsonschema:"required,enum=not_found,enum=unavailable,enum=slow" yaml:"answer"`
	Times  int    `jsonschema:"required,minimum=1"                                yaml:"times"`
}

// ContentTypes is every Content Type the Scenario needs the binary to
// register, in id order.
func (s Scenario) ContentTypes() []string {
	var out []string
	if s.Music != nil {
		out = append(out, "music")
	}
	if s.Library != nil {
		out = append(out, s.Library.contentTypes()...)
	}
	slices.Sort(out)
	return slices.Compact(out)
}

const (
	// AnswerNotFound answers 404, AnswerUnavailable 503 with Retry-After: 1,
	// and AnswerSlow holds the response until the client gives up.
	AnswerNotFound    = "not_found"
	AnswerUnavailable = "unavailable"
	AnswerSlow        = "slow"
)

var (
	mbid         = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	partialDate  = regexp.MustCompile(`^\d{4}(-\d{2}(-\d{2})?)?$`)
	musicKinds   = []string{"Album", "EP", "Single"}
	musicAnswers = []string{AnswerNotFound, AnswerUnavailable, AnswerSlow}
)

func (m Music) check(add func(format string, args ...any)) {
	if len(m.Artists) == 0 {
		add("music lists no artist")
	}
	owners := owners{artistOf: map[string]string{}, albumOf: map[string]string{}}
	for i, a := range m.Artists {
		if !mbid.MatchString(a.ID) {
			add("music artist %q is not a UUID", a.ID)
		}
		if slices.ContainsFunc(m.Artists[:i], func(b MusicArtist) bool { return b.ID == a.ID }) {
			add("music artist %s is listed twice", a.ID)
		}
		if len(a.Revisions) == 0 {
			add("music artist %s has no revision", a.ID)
		}
		for r, rev := range a.Revisions {
			owners.check(add, a.ID, "music artist "+a.ID+" revision "+strconv.Itoa(r), rev.Albums)
		}
	}
	for i, f := range m.Faults {
		f.check(add, m.Artists, m.Faults[:i])
	}
}

type owners struct {
	artistOf, albumOf map[string]string
}

func (o owners) check(add func(format string, args ...any), artist, at string, albums []MusicAlbum) {
	seen := map[string]bool{}
	for _, al := range albums {
		al.check(add, at)
		if seen[al.ID] {
			add("%s lists album %s twice", at, al.ID)
		}
		seen[al.ID] = true
		if other, ok := o.artistOf[al.ID]; ok && other != artist {
			add("music album %s belongs to artists %s and %s", al.ID, other, artist)
		}
		o.artistOf[al.ID] = artist
		if other, ok := o.albumOf[al.Release]; ok && other != al.ID {
			add("music release %s belongs to albums %s and %s", al.Release, other, al.ID)
		}
		o.albumOf[al.Release] = al.ID
	}
}

func (f MusicFault) check(add func(format string, args ...any), artists []MusicArtist, earlier []MusicFault) {
	if !slices.ContainsFunc(artists, func(a MusicArtist) bool { return a.ID == f.Artist }) {
		add("music fault names artist %s, which the catalog does not list", f.Artist)
	}
	if slices.ContainsFunc(earlier, func(g MusicFault) bool { return g.Artist == f.Artist }) {
		add("music faults name artist %s twice", f.Artist)
	}
	if !slices.Contains(musicAnswers, f.Answer) {
		add("music fault answer %q is not one of not_found, unavailable, slow", f.Answer)
	}
	if f.Times < 1 {
		add("music fault for artist %s has times %d; it is at least 1", f.Artist, f.Times)
	}
}

func (al MusicAlbum) check(add func(format string, args ...any), at string) {
	for _, id := range []string{al.ID, al.Release} {
		if !mbid.MatchString(id) {
			add("%s album %q has an id %q that is not a UUID", at, al.Title, id)
		}
	}
	if !slices.Contains(musicKinds, al.Kind) {
		add("%s album %s kind %q is not one of Album, EP, Single", at, al.ID, al.Kind)
	}
	if al.FirstRelease != "" && !partialDate.MatchString(al.FirstRelease) {
		add("%s album %s first_release %q is not YYYY, YYYY-MM or YYYY-MM-DD", at, al.ID, al.FirstRelease)
	}
	recordings := map[string]bool{}
	for _, t := range al.Tracks {
		if !mbid.MatchString(t.Recording) {
			add("%s album %s recording %q is not a UUID", at, al.ID, t.Recording)
		}
		if recordings[t.Recording] {
			add("%s album %s lists recording %s twice", at, al.ID, t.Recording)
		}
		recordings[t.Recording] = true
		if t.Disc < 1 {
			add("%s album %s track %s is on disc %d; discs start at 1", at, al.ID, t.Recording, t.Disc)
		}
	}
}
