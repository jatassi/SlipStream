package scenario

import (
	"regexp"
	"slices"
	"time"
)

// TMDB is the catalog fake-tmdb serves.
type TMDB struct {
	Movies []TMDBMovie `jsonschema:"required,minItems=1" yaml:"movies"`
}

type TMDBMovie struct {
	ID       int64         `jsonschema:"required,minimum=1"                         yaml:"id"`
	IMDbID   string        `jsonschema:"required,pattern=^tt[0-9]{7\\,}$"            yaml:"imdb_id"`
	Title    string        `jsonschema:"required"                                   yaml:"title"`
	Released string        `jsonschema:"required,pattern=^[0-9]{4}-[0-9]{2}-[0-9]{2}$" yaml:"released"`
	Runtime  time.Duration `jsonschema:"required"                                   yaml:"runtime"`
}

var (
	imdbID  = regexp.MustCompile(`^tt\d{7,}$`)
	dayDate = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
)

func (c TMDB) check(add func(format string, args ...any)) {
	if len(c.Movies) == 0 {
		add("tmdb lists no movie")
	}
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
		if _, err := time.Parse(time.DateOnly, m.Released); err != nil || !dayDate.MatchString(m.Released) {
			add("tmdb movie %d released %q is not YYYY-MM-DD", m.ID, m.Released)
		}
		if m.Runtime%time.Minute != 0 || m.Runtime < 0 {
			add("tmdb movie %d runtime %s is not whole minutes", m.ID, m.Runtime)
		}
	}
}
