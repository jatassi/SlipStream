package scenario

import (
	"fmt"
	"hash/fnv"
	"iter"
	"strconv"
	"time"
)

// TMDBGenerated is a catalog of Count movies made from Seed. Movie i is a
// pure function of the seed and i, so fake-tmdb holds none of them.
type TMDBGenerated struct {
	Count int    `jsonschema:"required,minimum=1,maximum=50000" yaml:"count"`
	Seed  string `jsonschema:"required"                         yaml:"seed"`
}

// GeneratedFirstID is the first generated movie's TMDB id. Generated ids
// run from it for MaxGenerated ids, a block no listed movie may use.
const (
	GeneratedFirstID int64 = 50_000_001
	MaxGenerated     int   = 50_000
)

func (g TMDBGenerated) ID(i int) int64 { return GeneratedFirstID + int64(i) }

func (g TMDBGenerated) Index(id int64) (int, bool) {
	i := id - GeneratedFirstID
	if i < 0 || i >= int64(g.Count) {
		return 0, false
	}
	return int(i), true
}

func inGeneratedBlock(id int64) bool {
	return id >= GeneratedFirstID && id < GeneratedFirstID+int64(MaxGenerated)
}

var (
	adjectives = []string{
		"Quiet", "Hollow", "Silver", "Last", "Burning", "Northern", "Hidden", "Broken", "Golden", "Distant",
		"Crimson", "Restless", "Frozen", "Wandering", "Bright", "Secret", "Fallen", "Electric", "Painted", "Endless",
		"Lonely", "Midnight", "Paper", "Iron", "Velvet", "Salt", "Glass", "Winter", "Summer", "Second",
	}
	nouns = []string{
		"Orchard", "Harbor", "Lantern", "River", "Garden", "Signal", "Mountain", "Station", "Engine", "Archive",
		"Meadow", "Lighthouse", "Kingdom", "Machine", "Island", "Bridge", "Tide", "Circus", "Frontier", "Forest",
		"Letter", "Window", "Comet", "Valley", "Mirror", "Orbit", "Canyon", "Voyage", "Echo", "Parade",
	}
	accented = []string{
		"Château", "Café", "Fiancée", "Señorita", "Résumé", "Façade", "Naïveté", "Øresund", "Crème", "Déjà Vu",
		"Piñata", "Mañana", "Smörgåsbord", "Über", "Jalapeño", "Protégé", "Soirée", "Doppelgänger", "Ångström", "Belle Époque",
	}
	places = []string{
		"the North", "Avalon", "the Deep", "Saturn", "the Plains", "Kestrel Bay", "the Old Town", "Marrow Hill",
		"the Coast", "Windmere", "the Delta", "Ashford",
	}
	genres = []string{
		"Action", "Adventure", "Animation", "Comedy", "Crime", "Documentary", "Drama", "Family", "Fantasy", "History",
		"Horror", "Music", "Mystery", "Romance", "Science Fiction", "Thriller", "War", "Western",
	}
	certifications = []string{"G", "PG", "PG-13", "R", "NC-17", ""}
	firstDay       = time.Date(1960, time.January, 1, 0, 0, 0, 0, time.UTC)
	lastDay        = time.Date(2030, time.December, 31, 0, 0, 0, 0, time.UTC)
)

// Movie is generated movie i: a title from the word lists, accented in one
// movie of every twenty and with a British alternative title in one of
// every ten, a US cinema date from 1960 to 2030 with a certification, a US
// digital date after it, a runtime, two genres, and a poster and a
// backdrop.
func (g TMDBGenerated) Movie(i int) TMDBMovie {
	r := mix(g.Seed, i)
	id := g.ID(i)
	noun := nouns[r.n(len(nouns))]
	if i%20 == 0 {
		noun = accented[r.n(len(accented))]
	}
	title := "The " + adjectives[r.n(len(adjectives))] + " " + noun
	if r.n(3) == 0 {
		title += " of " + places[r.n(len(places))]
	}
	cinema := firstDay.AddDate(0, 0, r.n(int(lastDay.Sub(firstDay).Hours()/24)+1))
	first := r.n(len(genres))
	m := TMDBMovie{
		ID:          id,
		IMDbID:      "tt" + strconv.FormatInt(id, 10),
		Title:       title,
		Released:    cinema.AddDate(0, 0, 45+r.n(120)).Format(time.DateOnly),
		Runtime:     time.Duration(80+r.n(100)) * time.Minute,
		Dates:       []TMDBDate{{Country: "US", Kind: "cinema", Date: cinema.Format(time.DateOnly), Certification: certifications[r.n(len(certifications))]}},
		Overview:    fmt.Sprintf("A story set in %s over one %s.", places[r.n(len(places))], []string{"summer", "winter", "night", "year"}[r.n(4)]),
		Genres:      []string{genres[first], genres[(first+1+r.n(len(genres)-1))%len(genres)]},
		Images:      []TMDBImage{{Shape: "poster", Language: "en", Votes: 5.5}, {Shape: "backdrop", Votes: 5.5}},
		VoteAverage: float64(40+r.n(51)) / 10,
		VoteCount:   int64(10 + r.n(5000)),
	}
	if i%10 == 0 {
		m.Titles = []TMDBTitle{{Country: "GB", Title: adjectives[r.n(len(adjectives))] + " " + noun}}
	}
	return m
}

// All is every movie the fake serves: the listed ones, then the generated
// ones in id order.
func (c TMDB) All() iter.Seq[TMDBMovie] {
	return func(yield func(TMDBMovie) bool) {
		for _, m := range c.Movies {
			if !yield(m) {
				return
			}
		}
		if c.Generated == nil {
			return
		}
		for i := range c.Generated.Count {
			if !yield(c.Generated.Movie(i)) {
				return
			}
		}
	}
}

func (c TMDB) Movie(id int64) (TMDBMovie, bool) {
	for _, m := range c.Movies {
		if m.ID == id {
			return m, true
		}
	}
	if c.Generated != nil {
		if i, ok := c.Generated.Index(id); ok {
			return c.Generated.Movie(i), true
		}
	}
	return TMDBMovie{}, false
}

// stream is splitmix64 from a hash of the seed and index, so each draw is
// fixed by them alone.
type stream struct{ state uint64 }

func mix(seed string, i int) *stream {
	h := fnv.New64a()
	_, _ = h.Write([]byte(seed))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write(strconv.AppendInt(nil, int64(i), 10))
	return &stream{state: h.Sum64()}
}

func (s *stream) n(bound int) int {
	s.state += 0x9e3779b97f4a7c15
	z := s.state
	z = (z ^ (z >> 30)) * 0xbf58476d1ce4e5b9
	z = (z ^ (z >> 27)) * 0x94d049bb133111eb
	z ^= z >> 31
	return int(z % uint64(bound)) //nolint:gosec // G115: bound is a word list's length, so the remainder fits an int
}
