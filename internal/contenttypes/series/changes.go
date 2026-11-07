package series

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/jatassi/slipstream/internal/metadata"
	"github.com/jatassi/slipstream/internal/works"
)

// Changes reads TVDB's updates and then TMDB's series changes as one list
// whose Pages is the sum of theirs. Page reads TVDB's page of that number
// and, while it lies within TVDB's pages, TMDB's first page for its
// count. Past them, page reads TMDB's page after TVDB's last. A TMDB
// change is keyed "tmdb-<id>", which Refreshing.KeysOf maps to a series'
// key. TMDB listing no changes that far back drops only its own list.
func (s *Source) Changes(ctx context.Context, ask metadata.Ask, since, until time.Time, page int) (metadata.ChangesPage, error) {
	tv, err := s.tvdb.Updates(ctx, ask, since, page)
	if err != nil {
		return metadata.ChangesPage{}, err
	}
	tmdbPage := max(page-tv.Pages, 1)
	tm, err := s.tmdb.SeriesChanges(ctx, ask, since, until, tmdbPage)
	if err != nil && !errors.Is(err, metadata.ErrNoChanges) {
		return metadata.ChangesPage{}, err
	}
	out := metadata.ChangesPage{Pages: tv.Pages + tm.TotalPages}
	if page <= tv.Pages {
		for _, id := range tv.Series {
			out.Keys = append(out.Keys, keyOf(int64(id)))
		}
		return out, nil
	}
	for _, id := range tm.IDs {
		out.Keys = append(out.Keys, works.Key(tmdbPrefix+strconv.FormatInt(id, 10)))
	}
	return out, nil
}
