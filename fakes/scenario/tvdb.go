package scenario

import (
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
// season 0. Names are fictional.
type TVDBSeries struct {
	ID      int64    `jsonschema:"required,minimum=1"                            yaml:"id"`
	Name    string   `jsonschema:"required"                                      yaml:"name"`
	English string   `yaml:"english,omitempty"`
	Aliases []string `yaml:"aliases,omitempty"`
	Status  string   `jsonschema:"required,enum=continuing,enum=ended,enum=upcoming" yaml:"status"`
	Network string   `yaml:"network,omitempty"`
	IMDb    string   `jsonschema:"pattern=^tt[0-9]{7\\,}$" yaml:"imdb,omitempty"`
	// TMDB is the series' TMDB id in TVDB's remote ids; zero names none.
	TMDB    int64        `jsonschema:"minimum=0" yaml:"tmdb,omitempty"`
	Seasons []TVDBSeason `yaml:"seasons,omitempty"`
	// DVD and Absolute renumber episodes by id; an episode an order omits
	// isn't in it, and an order left out isn't one the series has.
	DVD      map[int64]TVDBNumber `yaml:"dvd,omitempty"`
	Absolute map[int64]int        `yaml:"absolute,omitempty"`
}

type TVDBSeason struct {
	Number   int           `jsonschema:"minimum=0" yaml:"number"`
	Episodes []TVDBEpisode `yaml:"episodes,omitempty"`
}

// TVDBEpisode is one episode. An empty Aired is announced with no date.
type TVDBEpisode struct {
	ID      int64  `jsonschema:"required,minimum=1"                  yaml:"id"`
	Number  int    `jsonschema:"required,minimum=0"                  yaml:"number"`
	Name    string `jsonschema:"required"                            yaml:"name"`
	Aired   string `jsonschema:"pattern=^[0-9]{4}-[0-9]{2}-[0-9]{2}$" yaml:"aired,omitempty"`
	Runtime int    `jsonschema:"minimum=0"                           yaml:"runtime,omitempty"`
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

// TVDBGeneratedFirstID is the first id generated series, seasons and
// episodes take, each from its own block: series from here, seasons from
// a million on, episodes from two million on.
const TVDBGeneratedFirstID int64 = 60_000_001

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
	for _, g := range c.Generated {
		if g.Count < 1 || g.Seasons < 1 || g.Episodes < 1 || g.Seed == "" {
			add("tvdb generated has count %d, %d seasons of %d episodes and seed %q; want each at least 1 and a seed", g.Count, g.Seasons, g.Episodes, g.Seed)
		}
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
	for j, season := range s.Seasons {
		if slices.ContainsFunc(s.Seasons[:j], func(o TVDBSeason) bool { return o.Number == season.Number }) {
			add("tvdb series %d lists season %d twice", s.ID, season.Number)
		}
		for _, e := range season.Episodes {
			if slices.Contains(seen, e.ID) {
				add("tvdb episode %d is listed twice", e.ID)
			}
			seen = append(seen, e.ID)
			if e.Aired != "" && !isDay(e.Aired) {
				add("tvdb episode %d aired %q is not YYYY-MM-DD", e.ID, e.Aired)
			}
		}
	}
	return seen
}
