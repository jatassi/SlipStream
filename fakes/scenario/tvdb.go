package scenario

import (
	"fmt"
	"iter"
	"regexp"
	"slices"
	"time"
)

// TVDB is the catalog fake-tvdb serves: the series it lists, then the ones
// it generates.
type TVDB struct {
	Series    []TVDBSeries    `yaml:"series,omitempty"`
	Generated []TVDBGenerated `yaml:"generated,omitempty"`
	// TokenLifetime is how long a token fake-tvdb issues lasts; zero is
	// TVDB's own month. The tvdb-expire-tokens cue ends every live token
	// at once, which is what tests use.
	TokenLifetime time.Duration `jsonschema:"minimum=0" yaml:"token_lifetime,omitempty"`
}

// TVDBSeries is one series as TVDB lists it in Aired order, specials in
// season 0. Names are fictional, except in a Scenario recorded from TVDB.
// Each recorded field left out is one fake-tvdb makes up.
type TVDBSeries struct {
	ID      int64    `jsonschema:"required,minimum=1"                            yaml:"id"`
	Name    string   `jsonschema:"required"                                      yaml:"name"`
	English string   `yaml:"english,omitempty"`
	Aliases []string `yaml:"aliases,omitempty"`
	Status  string   `jsonschema:"required,enum=continuing,enum=ended,enum=upcoming" yaml:"status"`
	Network string   `yaml:"network,omitempty"`
	IMDb    string   `jsonschema:"pattern=^tt[0-9]{7\\,}$" yaml:"imdb,omitempty"`
	// TMDB is the series' TMDB id in TVDB's remote ids; zero names none.
	TMDB int64 `jsonschema:"minimum=0" yaml:"tmdb,omitempty"`
	// Overview is the series' English overview.
	Overview string   `yaml:"overview,omitempty"`
	Genres   []string `yaml:"genres,omitempty"`
	// OriginalLanguage and OriginalCountry are TVDB's three-letter codes,
	// such as kor.
	OriginalLanguage string `jsonschema:"pattern=^[a-z]{3}$" yaml:"original_language,omitempty"`
	OriginalCountry  string `jsonschema:"pattern=^[a-z]{3}$" yaml:"original_country,omitempty"`
	// Image is the TVDB artwork path of the series' recorded poster in the
	// artwork store, such as /banners/v4/series/1/posters/a.jpg.
	Image   string       `jsonschema:"pattern=^/banners/([A-Za-z0-9_-]+/)*[A-Za-z0-9_-]+\\.(jpg|jpeg|png)$" yaml:"image,omitempty"`
	Seasons []TVDBSeason `yaml:"seasons,omitempty"`
	// DVD and Absolute renumber episodes by id; an episode an order omits
	// isn't in it, and an order left out isn't one the series has.
	DVD      map[int64]TVDBNumber `yaml:"dvd,omitempty"`
	Absolute map[int64]int        `yaml:"absolute,omitempty"`
}

// TVDBSeason is one Aired season. Name and Image are as on TVDBSeries.
type TVDBSeason struct {
	Number   int           `jsonschema:"minimum=0"                                                          yaml:"number"`
	Name     string        `yaml:"name,omitempty"`
	Image    string        `jsonschema:"pattern=^/banners/([A-Za-z0-9_-]+/)*[A-Za-z0-9_-]+\\.(jpg|jpeg|png)$" yaml:"image,omitempty"`
	Episodes []TVDBEpisode `yaml:"episodes,omitempty"`
}

// TVDBEpisode is one episode. An empty Aired is announced with no date.
// Overview and Image are as on TVDBSeries, Image being a still.
type TVDBEpisode struct {
	ID       int64  `jsonschema:"required,minimum=1"                                                 yaml:"id"`
	Number   int    `jsonschema:"required,minimum=0"                                                 yaml:"number"`
	Name     string `jsonschema:"required"                                                           yaml:"name"`
	Aired    string `jsonschema:"pattern=^[0-9]{4}-[0-9]{2}-[0-9]{2}$"                                yaml:"aired,omitempty"`
	Runtime  int    `jsonschema:"minimum=0"                                                          yaml:"runtime,omitempty"`
	Overview string `yaml:"overview,omitempty"`
	Image    string `jsonschema:"pattern=^/banners/([A-Za-z0-9_-]+/)*[A-Za-z0-9_-]+\\.(jpg|jpeg|png)$" yaml:"image,omitempty"`
}

// TVDBNumber is an episode's place in an order.
type TVDBNumber struct {
	Season  int `jsonschema:"minimum=0" yaml:"season"`
	Episode int `jsonschema:"minimum=0" yaml:"episode"`
}

// TVDBGenerated is Count series of Seasons seasons of Episodes episodes
// each, every one a pure function of Seed and its index, with ids from
// the block TVDBGeneratedFirstID starts.
type TVDBGenerated struct {
	Count    int    `jsonschema:"required,minimum=1" yaml:"count"`
	Seasons  int    `jsonschema:"required,minimum=1" yaml:"seasons"`
	Episodes int    `jsonschema:"required,minimum=1" yaml:"episodes"`
	Seed     string `jsonschema:"required"           yaml:"seed"`
}

// TVDBGeneratedFirstID is the first id generated series and episodes
// take, each from its own block: series from here, episodes from two
// million on. No listed series or episode may use an id from here on.
const (
	TVDBGeneratedFirstID int64 = 60_000_001
	tvdbEpisodesFirstID        = TVDBGeneratedFirstID + 2_000_000
	// TVDBMaxGenerated bounds the series and the episodes generated across
	// every block, so neither block reaches the next.
	TVDBMaxGenerated = 1_000_000
)

var tvdbStatuses = []string{"continuing", "ended", "upcoming"}

func (c TVDB) check(add func(format string, args ...any)) {
	if c.TokenLifetime < 0 {
		add("tvdb token_lifetime %s is below zero", c.TokenLifetime)
	}
	var episodes []int64
	for i, s := range c.Series {
		if slices.ContainsFunc(c.Series[:i], func(o TVDBSeries) bool { return o.ID == s.ID }) {
			add("tvdb series %d is listed twice", s.ID)
		}
		episodes = s.check(add, episodes)
	}
	series, generated := 0, 0
	for _, g := range c.Generated {
		if g.Count < 1 || g.Seasons < 1 || g.Episodes < 1 || g.Seed == "" {
			add("tvdb generated has count %d, %d seasons of %d episodes and seed %q; want each at least 1 and a seed", g.Count, g.Seasons, g.Episodes, g.Seed)
		}
		series += g.Count
		generated += g.Count * g.Seasons * g.Episodes
	}
	if series > TVDBMaxGenerated || generated > TVDBMaxGenerated {
		add("tvdb generates %d series of %d episodes in all; want at most %d of each", series, generated, TVDBMaxGenerated)
	}
}

// check holds s to TVDB's shape, answering seen with s's episode ids
// added so an id another series lists is caught.
func (s TVDBSeries) check(add func(format string, args ...any), seen []int64) []int64 {
	if s.ID < 1 || s.ID >= TVDBGeneratedFirstID {
		add("tvdb series %q has id %d; ids run from 1 to %d", s.Name, s.ID, TVDBGeneratedFirstID-1)
	}
	if s.Name == "" {
		add("tvdb series %d has no name", s.ID)
	}
	if !slices.Contains(tvdbStatuses, s.Status) {
		add("tvdb series %d status %q is not one of %v", s.ID, s.Status, tvdbStatuses)
	}
	if s.IMDb != "" && !imdbID.MatchString(s.IMDb) {
		add("tvdb series %d imdb %q is not tt and seven or more digits", s.ID, s.IMDb)
	}
	checkTVDBFile(add, s.ID, "image", s.Image)
	for j, season := range s.Seasons {
		if slices.ContainsFunc(s.Seasons[:j], func(o TVDBSeason) bool { return o.Number == season.Number }) {
			add("tvdb series %d lists season %d twice", s.ID, season.Number)
		}
		checkTVDBFile(add, s.ID, fmt.Sprintf("season %d image", season.Number), season.Image)
		seen = season.check(add, s.ID, seen)
	}
	return seen
}

func (season TVDBSeason) check(add func(format string, args ...any), series int64, seen []int64) []int64 {
	for _, e := range season.Episodes {
		checkTVDBFile(add, series, fmt.Sprintf("episode %d image", e.ID), e.Image)
		if e.ID >= TVDBGeneratedFirstID {
			add("tvdb episode %d is in the generated block from %d", e.ID, TVDBGeneratedFirstID)
		}
		if slices.Contains(seen, e.ID) {
			add("tvdb episode %d is listed twice", e.ID)
		}
		seen = append(seen, e.ID)
		if e.Aired != "" && !isDay(e.Aired) {
			add("tvdb episode %d aired %q is not YYYY-MM-DD", e.ID, e.Aired)
		}
	}
	return seen
}

// TVDBArtworkFile is a TVDB artwork path the artwork store can hold, in
// the shape SlipStream keeps.
var TVDBArtworkFile = regexp.MustCompile(`^/banners/(?:[A-Za-z0-9_-]+/)*[A-Za-z0-9_-]+\.(jpg|jpeg|png)$`)

// checkTVDBFile checks a recorded TVDB file's path, when the record names
// one.
func checkTVDBFile(add func(format string, args ...any), series int64, what, p string) {
	switch {
	case p == "":
	case !TVDBArtworkFile.MatchString(p):
		add("tvdb series %d %s %q is not a TVDB artwork path such as /banners/v4/series/1/posters/a.jpg", series, what, p)
	default:
		if _, ok := artworkStore.Lookup(p); !ok {
			add("tvdb series %d %s %s is not in fakes/internal/artwork/files/tvdb", series, what, p)
		}
	}
}

// All is every series fake-tvdb serves: the listed ones, then each
// generated block's in id order.
func (c TVDB) All() iter.Seq[TVDBSeries] {
	return func(yield func(TVDBSeries) bool) {
		for _, s := range c.Series {
			if !yield(s) {
				return
			}
		}
		for b := range c.blocks() {
			for i := range b.Count {
				if !yield(b.series(i)) {
					return
				}
			}
		}
	}
}

// Find is the series whose TVDB id is id, listed or generated.
func (c TVDB) Find(id int64) (TVDBSeries, bool) {
	for _, s := range c.Series {
		if s.ID == id {
			return s, true
		}
	}
	for b := range c.blocks() {
		if i := id - b.firstSeries; i >= 0 && i < int64(b.Count) {
			return b.series(int(i)), true
		}
	}
	return TVDBSeries{}, false
}

// block is one generated block placed in the id blocks after the ones
// before it.
type block struct {
	TVDBGenerated
	firstSeries, firstEpisode int64
}

func (c TVDB) blocks() iter.Seq[block] {
	return func(yield func(block) bool) {
		series, episode := TVDBGeneratedFirstID, tvdbEpisodesFirstID
		for _, g := range c.Generated {
			if !yield(block{TVDBGenerated: g, firstSeries: series, firstEpisode: episode}) {
				return
			}
			series += int64(g.Count)
			episode += int64(g.Count * g.Seasons * g.Episodes)
		}
	}
}

var (
	networks     = []string{"Lantern One", "Harbor TV", "Northlight", "Kestrel+", "Signal Nine"}
	tvdbStarts   = time.Date(1970, time.January, 1, 0, 0, 0, 0, time.UTC)
	tvdbStartsBy = 20 * 365
)

// series is the block's series i: a title from the word lists, Seasons
// seasons of Episodes episodes from season 1, one airing each day from a
// day in 1970 to 1990, ended once every one has aired.
func (b block) series(i int) TVDBSeries {
	r := mix(b.Seed, i)
	id := b.firstSeries + int64(i)
	s := TVDBSeries{
		ID:      id,
		Name:    adjectives[r.n(len(adjectives))] + " " + nouns[r.n(len(nouns))] + " of " + places[r.n(len(places))],
		Status:  "ended",
		Network: networks[r.n(len(networks))],
		IMDb:    fmt.Sprintf("tt%d", id),
		Seasons: make([]TVDBSeason, b.Seasons),
	}
	day := tvdbStarts.AddDate(0, 0, r.n(tvdbStartsBy))
	episode := b.firstEpisode + int64(i*b.Seasons*b.Episodes)
	runtime := 22 + r.n(39)
	for n := range b.Seasons {
		season := TVDBSeason{Number: n + 1, Episodes: make([]TVDBEpisode, b.Episodes)}
		for e := range b.Episodes {
			season.Episodes[e] = TVDBEpisode{
				ID: episode, Number: e + 1, Runtime: runtime, Aired: day.Format(time.DateOnly),
				Name: "The " + adjectives[r.n(len(adjectives))] + " " + nouns[r.n(len(nouns))],
			}
			episode++
			day = day.AddDate(0, 0, 1)
		}
		s.Seasons[n] = season
	}
	return s
}
