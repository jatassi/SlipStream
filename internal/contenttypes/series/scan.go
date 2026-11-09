package series

import (
	"context"
	"fmt"
	"path"
	"strconv"

	"github.com/jatassi/slipstream/internal/contenttypes/series/internal/db"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/scanning"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// Hint reads a series folder's name, and names the Work key its tvdb tag
// gives.
func (*Series) Hint(name string) scanning.Hint {
	h := scanning.ReadName(name)
	h.Key = works.Key(h.Of(scanning.TVDB))
	return h
}

// Claims are each series' first-aired year and its TVDB, TMDB and IMDb
// ids.
func (*Series) Claims(ctx context.Context, r storage.Reader[file], ws works.Among[works.WorkID]) (map[works.WorkID]scanning.Claim, error) {
	q := db.New(r)
	rows, err := works.Read(ws, func() ([]db.ListEveryClaimRow, error) { return q.ListEveryClaim(ctx) }, func(ids []works.WorkID) ([]db.ListEveryClaimRow, error) {
		rows, err := q.ListClaimsOf(ctx, ids)
		out := make([]db.ListEveryClaimRow, len(rows))
		for i, row := range rows {
			out[i] = db.ListEveryClaimRow(row)
		}
		return out, err
	})
	if err != nil {
		return nil, err
	}
	out := make(map[works.WorkID]scanning.Claim, len(rows))
	for _, row := range rows {
		c := scanning.Claim{IDs: []scanning.ID{{Provider: scanning.TVDB, Value: strconv.FormatInt(int64(row.TvdbID), 10)}}}
		if row.TmdbID != nil {
			c.IDs = append(c.IDs, scanning.ID{Provider: scanning.TMDB, Value: strconv.FormatInt(int64(*row.TmdbID), 10)})
		}
		if row.ImdbID != nil {
			c.IDs = append(c.IDs, scanning.ID{Provider: scanning.IMDb, Value: *row.ImdbID})
		}
		if row.FirstAired != nil {
			first, err := works.ParseDate(*row.FirstAired)
			if err != nil {
				return nil, fmt.Errorf("work %s: %w", row.WorkID, err)
			}
			c.Year = first.Year
		}
		out[row.WorkID] = c
	}
	return out, nil
}

func (*Series) YearOf(c works.Candidate) int {
	for _, f := range c.Fields {
		if f.Key == "year" {
			return int(f.Number)
		}
	}
	return 0
}

// Place reads a file name's episode numbering as the series' release
// names read it, and places it at the stored episode its Expect covers.
func (s *Series) Place(ctx context.Context, r storage.Reader[file], work works.WorkID) (scanning.Placer, error) {
	ex, err := s.Expect(ctx, r, work)
	if err != nil {
		return nil, err
	}
	return func(rel string) scanning.Placement {
		n := release.Title(path.Base(rel), s, nil).Name
		switch {
		case len(n.Marks) == 0:
			return scanning.Unplaced(scanning.Reason{Code: scanning.NoNumber})
		case n.Span != release.One:
			return scanning.Unplaced(scanning.Reason{Code: scanning.SeveralItems})
		}
		covers := ex.Covers(n)
		if len(covers) == 0 {
			return scanning.Unplaced(scanning.Reason{Code: scanning.NoSuchItem, Names: n.Marks[0].Words})
		}
		return scanning.Placed(covers[0])
	}, nil
}
