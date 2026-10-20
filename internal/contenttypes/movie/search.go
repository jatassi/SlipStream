package movie

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/metadata"
	"github.com/jatassi/slipstream/internal/metadata/artwork"
	"github.com/jatassi/slipstream/internal/providers/tmdb"
	"github.com/jatassi/slipstream/internal/works"
)

const maxResults = 25

// Search reads TMDB's pages in ask's language until it has 25 movies or
// none are left. A query ending in a year is read twice, first as the
// title before it filtered to that year, then as typed, so "Heat 1995"
// answers that year's film first and "Blade Runner 2049" still finds the
// film with that name. A movie both read is answered once.
func (s *Source) Search(ctx context.Context, ask metadata.Ask, query string) ([]works.Candidate, error) {
	type read struct {
		query string
		year  int
	}
	reads := []read{{query: query}}
	if title, y, ok := trailingYear(query, time.Now()); ok {
		reads = []read{{title, y}, {query: query}}
	}
	var out []works.Candidate
	seen := map[int64]bool{}
	for _, r := range reads {
		for page, pages := 1, 1; page <= pages && len(out) < maxResults; page++ {
			p, err := s.tmdb.SearchMovies(ctx, ask, r.query, r.year, page)
			if err != nil {
				return nil, err
			}
			pages = p.TotalPages
			for _, m := range p.Results {
				if !seen[m.ID] && len(out) < maxResults {
					seen[m.ID] = true
					out = append(out, candidateOf(m))
				}
			}
		}
	}
	return out, nil
}

// firstFilm is the year of the earliest film TMDB lists.
const firstFilm = 1874

// trailingYear splits "Heat 1995" into "Heat" and 1995. A query that is
// only a year, such as "1917", has no title to filter and isn't split, nor
// is one whose year falls before the first film or more than ten years
// after now.
func trailingYear(query string, now time.Time) (title string, year int, ok bool) {
	words := strings.Fields(query)
	if len(words) < 2 {
		return "", 0, false
	}
	last := words[len(words)-1]
	y, err := strconv.Atoi(last)
	if err != nil || len(last) != 4 || y < firstFilm || y > now.Year()+10 {
		return "", 0, false
	}
	return strings.Join(words[:len(words)-1], " "), y, true
}

func candidateOf(m tmdb.MovieResult) works.Candidate {
	c := works.Candidate{Key: works.Key(strconv.FormatInt(m.ID, 10)), Label: m.Title}
	if m.OriginalTitle != m.Title {
		c.Original = m.OriginalTitle
	}
	c.Overview, _, _ = strings.Cut(strings.TrimSpace(m.Overview), "\n")
	if m.PosterPath != "" {
		c.Poster = imageRef(artwork.ShapePoster, m.PosterPath)
	}
	if y := year(m.ReleaseDate); y != nil {
		c.Fields = []works.FieldValue{works.NumberField("year", float64(*y))}
	}
	return c
}
