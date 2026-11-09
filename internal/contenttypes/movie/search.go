package movie

import (
	"context"
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/internal/metadata"
	"github.com/jatassi/slipstream/internal/metadata/artwork"
	"github.com/jatassi/slipstream/internal/providers/tmdb"
	"github.com/jatassi/slipstream/internal/works"
)

const maxResults = 25

// Search reads TMDB's pages in ask's language for each of the query's
// metadata.SearchReads until it has 25 movies or none are left. A movie
// two reads find is answered once.
func (s *Source) Search(ctx context.Context, ask metadata.Ask, query string) ([]works.Candidate, error) {
	var out []works.Candidate
	seen := map[int64]bool{}
	for _, r := range metadata.SearchReads(query, s.now()) {
		for page, pages := 1, 1; page <= pages && len(out) < maxResults; page++ {
			p, err := s.tmdb.SearchMovies(ctx, ask, r.Query, r.Year, page)
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
