package scenario

import (
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

// TMDB is the catalog fake-tmdb serves: the movies it lists, then the
// ones it generates.
type TMDB struct {
	Movies    []TMDBMovie    `yaml:"movies,omitempty"`
	Generated *TMDBGenerated `yaml:"generated,omitempty"`
}

// TMDBMovie is one movie's whole TMDB record. Only id, imdb_id, title and
// runtime are required; a movie with neither released nor dates is
// undated.
type TMDBMovie struct {
	ID     int64  `jsonschema:"required,minimum=1"              yaml:"id"`
	IMDbID string `jsonschema:"required,pattern=^tt[0-9]{7\\,}$" yaml:"imdb_id"`
	Title  string `jsonschema:"required"                        yaml:"title"`
	// Released is one US digital release, so a Scenario that names only it
	// has its movie out at home that day.
	Released         string            `jsonschema:"pattern=^[0-9]{4}-[0-9]{2}-[0-9]{2}$" yaml:"released,omitempty"`
	Runtime          time.Duration     `jsonschema:"required"                            yaml:"runtime"`
	Dates            []TMDBDate        `yaml:"dates,omitempty"`
	OriginalTitle    string            `yaml:"original_title,omitempty"`
	OriginalLanguage string            `jsonschema:"pattern=^[a-z]{2}$" yaml:"original_language,omitempty"`
	Overview         string            `yaml:"overview,omitempty"`
	Tagline          string            `yaml:"tagline,omitempty"`
	Genres           []string          `yaml:"genres,omitempty"`
	Status           string            `jsonschema:"enum=Rumored,enum=Planned,enum=In Production,enum=Post Production,enum=Released,enum=Canceled" yaml:"status,omitempty"`
	Titles           []TMDBTitle       `yaml:"titles,omitempty"`
	Translations     []TMDBTranslation `yaml:"translations,omitempty"`
	Credits          TMDBCredits       `yaml:"credits,omitempty"`
	// Trailer is the YouTube key of the movie's one trailer.
	Trailer     string      `yaml:"trailer,omitempty"`
	Collection  *TMDBNamed  `yaml:"collection,omitempty"`
	Studio      *TMDBNamed  `yaml:"studio,omitempty"`
	Images      []TMDBImage `yaml:"images,omitempty"`
	VoteAverage float64     `jsonschema:"minimum=0,maximum=10" yaml:"vote_average,omitempty"`
	VoteCount   int64       `jsonschema:"minimum=0"            yaml:"vote_count,omitempty"`
	// Deleted is a movie TMDB no longer has: its id answers 404.
	Deleted bool `yaml:"deleted,omitempty"`
}

// TMDBDate is one release in one country.
type TMDBDate struct {
	Country       string `jsonschema:"required,pattern=^[A-Z]{2}$"                                                                    yaml:"country"`
	Kind          string `jsonschema:"required,enum=premiere,enum=limited,enum=cinema,enum=digital,enum=physical,enum=tv"             yaml:"kind"`
	Date          string `jsonschema:"required,pattern=^[0-9]{4}-[0-9]{2}-[0-9]{2}$"                                                  yaml:"date"`
	Certification string `yaml:"certification,omitempty"`
	Note          string `yaml:"note,omitempty"`
}

// TMDBDateKinds are the kinds a TMDBDate names, in TMDB's release type
// order from 1.
var TMDBDateKinds = []string{"premiere", "limited", "cinema", "digital", "physical", "tv"}

// TMDBTitle is an alternative title in one country.
type TMDBTitle struct {
	Country string `jsonschema:"required,pattern=^[A-Z]{2}$" yaml:"country"`
	Title   string `jsonschema:"required"                    yaml:"title"`
	Kind    string `yaml:"kind,omitempty"`
}

// TMDBTranslation is the movie's text in one language, such as fr-FR.
type TMDBTranslation struct {
	Language string `jsonschema:"required,pattern=^[a-z]{2}-[A-Z]{2}$" yaml:"language"`
	Title    string `yaml:"title,omitempty"`
	Overview string `yaml:"overview,omitempty"`
	Tagline  string `yaml:"tagline,omitempty"`
}

type TMDBCredits struct {
	Cast []TMDBCast `yaml:"cast,omitempty"`
	Crew []TMDBCrew `yaml:"crew,omitempty"`
}

// TMDBCast is one role, billed in list order. Names are fictional.
type TMDBCast struct {
	ID        int64  `jsonschema:"required,minimum=1" yaml:"id"`
	Name      string `jsonschema:"required"           yaml:"name"`
	Character string `yaml:"character,omitempty"`
}

type TMDBCrew struct {
	ID         int64  `jsonschema:"required,minimum=1" yaml:"id"`
	Name       string `jsonschema:"required"           yaml:"name"`
	Department string `jsonschema:"required"           yaml:"department"`
	Job        string `jsonschema:"required"           yaml:"job"`
}

// TMDBNamed is a collection or a studio.
type TMDBNamed struct {
	ID   int64  `jsonschema:"required,minimum=1" yaml:"id"`
	Name string `jsonschema:"required"           yaml:"name"`
}

// TMDBImage is one image; fake-tmdb makes up its file path. Language is
// empty for an image with no text.
type TMDBImage struct {
	Shape    string  `jsonschema:"required,enum=poster,enum=backdrop,enum=logo" yaml:"shape"`
	Language string  `jsonschema:"pattern=^[a-z]{2}$"                          yaml:"language,omitempty"`
	Votes    float64 `jsonschema:"minimum=0,maximum=10"                        yaml:"votes,omitempty"`
}

var (
	imdbID   = regexp.MustCompile(`^tt\d{7,}$`)
	dayDate  = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	country  = regexp.MustCompile(`^[A-Z]{2}$`)
	language = regexp.MustCompile(`^[a-z]{2}-[A-Z]{2}$`)
)

func (c TMDB) check(add func(format string, args ...any)) {
	c.checkGenerated(add)
	for i, m := range c.Movies {
		if m.ID < 1 {
			add("tmdb movie %q has id %d; ids start at 1", m.Title, m.ID)
		}
		if slices.ContainsFunc(c.Movies[:i], func(o TMDBMovie) bool { return o.ID == m.ID }) {
			add("tmdb movie %d is listed twice", m.ID)
		}
		if !imdbID.MatchString(m.IMDbID) {
			add("tmdb movie %d imdb_id %q is not tt and seven or more digits", m.ID, m.IMDbID)
		}
		if m.Title == "" {
			add("tmdb movie %d has no title", m.ID)
		}
		if m.Released != "" && !isDay(m.Released) {
			add("tmdb movie %d released %q is not YYYY-MM-DD", m.ID, m.Released)
		}
		if m.Runtime%time.Minute != 0 || m.Runtime < 0 {
			add("tmdb movie %d runtime %s is not whole minutes", m.ID, m.Runtime)
		}
		m.checkRecord(add)
	}
}

// checkGenerated checks the generated catalog and keeps listed movies out
// of its id block, TMDB's and IMDb's alike.
func (c TMDB) checkGenerated(add func(format string, args ...any)) {
	if len(c.Movies) == 0 && c.Generated == nil {
		add("tmdb lists no movie and generates none")
	}
	if g := c.Generated; g != nil && (g.Count < 1 || g.Count > MaxGenerated || g.Seed == "") {
		add("tmdb generated has count %d and seed %q; want a count from 1 to %d and a seed", g.Count, g.Seed, MaxGenerated)
	}
	for _, m := range c.Movies {
		if n, err := strconv.ParseInt(strings.TrimPrefix(m.IMDbID, "tt"), 10, 64); inGeneratedBlock(m.ID) || (err == nil && inGeneratedBlock(n)) {
			add("tmdb movie %d uses an id from %d to %d, which generated movies hold", m.ID, GeneratedFirstID, GeneratedFirstID+int64(MaxGenerated)-1)
		}
	}
}

func (m TMDBMovie) checkRecord(add func(format string, args ...any)) {
	for _, d := range m.Dates {
		if !country.MatchString(d.Country) {
			add("tmdb movie %d date country %q is not two capital letters", m.ID, d.Country)
		}
		if !slices.Contains(TMDBDateKinds, d.Kind) {
			add("tmdb movie %d date kind %q is not one of %v", m.ID, d.Kind, TMDBDateKinds)
		}
		if !isDay(d.Date) {
			add("tmdb movie %d date %q is not YYYY-MM-DD", m.ID, d.Date)
		}
	}
	for _, t := range m.Titles {
		if !country.MatchString(t.Country) || t.Title == "" {
			add("tmdb movie %d title %q in %q needs a title and a two-letter country", m.ID, t.Title, t.Country)
		}
	}
	for i, t := range m.Translations {
		if !language.MatchString(t.Language) {
			add("tmdb movie %d translation %q is not a language such as fr-FR", m.ID, t.Language)
		}
		if slices.ContainsFunc(m.Translations[:i], func(o TMDBTranslation) bool { return o.Language == t.Language }) {
			add("tmdb movie %d translation %s is listed twice", m.ID, t.Language)
		}
	}
	for _, im := range m.Images {
		if !slices.Contains([]string{"poster", "backdrop", "logo"}, im.Shape) {
			add("tmdb movie %d image shape %q is not poster, backdrop or logo", m.ID, im.Shape)
		}
	}
}

func isDay(s string) bool {
	_, err := time.Parse(time.DateOnly, s)
	return err == nil && dayDate.MatchString(s)
}
