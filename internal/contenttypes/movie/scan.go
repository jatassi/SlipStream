package movie

import (
	"context"
	"strconv"

	"github.com/jatassi/slipstream/internal/contenttypes/movie/internal/db"
	"github.com/jatassi/slipstream/internal/scanning"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// Hint reads a movie folder's name, and names the Work key its tmdb tag
// gives.
func (*Movie) Hint(name string) scanning.Hint {
	h := scanning.ReadName(name)
	h.Key = works.Key(h.Of(scanning.TMDB))
	return h
}

// Claims are each movie's year and its TMDB and IMDb ids.
func (*Movie) Claims(ctx context.Context, r storage.Reader[file], ws works.Among[works.WorkID]) (map[works.WorkID]scanning.Claim, error) {
	q := db.New(r)
	rows, err := works.Read(ws, func() ([]db.ListEveryClaimRow, error) { return q.ListEveryClaim(ctx) }, func(ids []works.WorkID) ([]db.ListEveryClaimRow, error) {
		rows, err := q.ListClaimsOf(ctx, ids)
		return as(rows, func(r db.ListClaimsOfRow) db.ListEveryClaimRow { return db.ListEveryClaimRow(r) }), err
	})
	if err != nil {
		return nil, err
	}
	out := make(map[works.WorkID]scanning.Claim, len(rows))
	for _, row := range rows {
		c := scanning.Claim{IDs: []scanning.ID{{Provider: scanning.TMDB, Value: strconv.FormatInt(int64(row.TmdbID), 10)}}}
		if row.ImdbID != nil {
			c.IDs = append(c.IDs, scanning.ID{Provider: scanning.IMDb, Value: string(*row.ImdbID)})
		}
		if row.Year != nil {
			c.Year = int(*row.Year)
		}
		out[row.WorkID] = c
	}
	return out, nil
}

func (*Movie) YearOf(c works.Candidate) int {
	for _, f := range c.Fields {
		if f.Key == "year" {
			return int(f.Number)
		}
	}
	return 0
}

// Place answers the movie's one Item for every file in its folder.
func (*Movie) Place(ctx context.Context, r storage.Reader[file], work works.WorkID) (scanning.Placer, error) {
	items, err := works.ItemsOf(ctx, r, works.Only([]works.WorkID{work}))
	if err != nil {
		return nil, err
	}
	item := items[work][0].ID
	return func(string) scanning.Placement { return scanning.Placed(item) }, nil
}
