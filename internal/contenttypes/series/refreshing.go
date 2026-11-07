package series

import (
	"context"
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/internal/contenttypes/series/internal/db"
	"github.com/jatassi/slipstream/internal/providers/tvdb"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// tmdbPrefix marks a key that names a series by its TMDB id: a TMDB
// change, or a search result TVDB doesn't list.
const tmdbPrefix = "tmdb-"

// Growing answers true for each of ws TVDB lists as Continuing or
// Upcoming, which can still gain episodes.
func (*Series) Growing(ctx context.Context, r storage.Reader[file], ws []works.WorkID) (map[works.WorkID]bool, error) {
	q := db.New(r)
	rows, err := works.Read(works.Only(ws), nil, func(ids []works.WorkID) ([]db.ListStatusesOfRow, error) { return q.ListStatusesOf(ctx, ids) })
	if err != nil {
		return nil, err
	}
	out := make(map[works.WorkID]bool, len(rows))
	for _, row := range rows {
		st := tvdb.Status(row.Status)
		out[row.WorkID] = st == tvdb.Continuing || st == tvdb.Upcoming
	}
	return out, nil
}

// KeysOf maps each "tmdb-<id>" key to the key of the series in the
// library with that TMDB id. Other keys, and TMDB ids no series has, map
// to none.
func (*Series) KeysOf(ctx context.Context, r storage.Reader[file], keys []works.Key) ([]works.Key, error) {
	var ids []*db.TMDBID
	for _, k := range keys {
		rest, ok := strings.CutPrefix(string(k), tmdbPrefix)
		if !ok {
			continue
		}
		if n, err := strconv.ParseInt(rest, 10, 64); err == nil && n > 0 {
			id := db.TMDBID(n)
			ids = append(ids, &id)
		}
	}
	q := db.New(r)
	rows, err := works.Read(works.Only(ids), nil, func(ids []*db.TMDBID) ([]db.WorksByTMDBRow, error) { return q.WorksByTMDB(ctx, ids) })
	if err != nil {
		return nil, err
	}
	out := make([]works.Key, len(rows))
	for i, row := range rows {
		out[i] = keyOf(int64(row.TvdbID))
	}
	return out, nil
}
