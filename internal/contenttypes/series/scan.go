package series

import (
	"context"
	"fmt"
	"path"
	"regexp"
	"strconv"

	"github.com/jatassi/slipstream/internal/contenttypes/series/internal/db"
	"github.com/jatassi/slipstream/internal/providers/tvdb"
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

var (
	// sxxeyy is an episode marker in a file name: S02E05, or several
	// episodes as S02E05E06, S02E05-E06 or S02E05-06.
	sxxeyy = regexp.MustCompile(`(?i)\bS(\d{1,4})E(\d{1,4})((?:-?E\d{1,4}|-\d{1,4}\b)*)`)
	// nxnn is the 2x05 marker, or several as 2x05-2x06.
	nxnn = regexp.MustCompile(`(?i)\b(\d{1,2})x(\d{2,3})((?:-\d{1,2}x\d{2,3})*)\b`)
)

// Place reads a file's SxxEyy or NxNN marker in the series' episode order,
// with Aired numbers in Absolute order as releases name them. A live
// episode wins over a withdrawn one with the same numbers.
func (*Series) Place(ctx context.Context, r storage.Reader[file], work works.WorkID) (scanning.Placer, error) {
	q := db.New(r)
	row, err := q.GetSeries(ctx, work)
	if err != nil {
		return nil, err
	}
	episodes, err := q.ListEpisodes(ctx, work)
	if err != nil {
		return nil, err
	}
	items, err := works.ItemsOf(ctx, r, works.Only([]works.WorkID{work}))
	if err != nil {
		return nil, err
	}
	numbered := make(map[works.ItemID]episodeAt, len(episodes))
	for _, e := range episodes {
		numbered[e.ItemID] = searchedAs(tvdb.Order(row.EpisodeOrder), e)
	}
	at := map[episodeAt]works.ItemID{}
	for _, it := range items[work] {
		if n, ok := numbered[it.ID]; ok {
			if _, taken := at[n]; !taken {
				at[n] = it.ID
			}
		}
	}
	return func(rel string) scanning.Placement {
		n, several, ok := markerOf(path.Base(rel))
		switch {
		case !ok:
			return scanning.Unplaced(scanning.Reason{Code: scanning.NoNumber})
		case several:
			return scanning.Unplaced(scanning.Reason{Code: scanning.SeveralItems})
		}
		item, ok := at[n]
		if !ok {
			return scanning.Unplaced(scanning.Reason{Code: scanning.NoSuchItem, Names: fmt.Sprintf("S%02dE%02d", n.season, n.episode)})
		}
		return scanning.Placed(item)
	}, nil
}

// markerOf is the episode a file name's first marker names, and whether it
// names several.
func markerOf(name string) (at episodeAt, several, ok bool) {
	m := sxxeyy.FindStringSubmatch(name)
	if m == nil {
		m = nxnn.FindStringSubmatch(name)
	}
	if m == nil {
		return episodeAt{}, false, false
	}
	season, _ := strconv.Atoi(m[1])
	episode, _ := strconv.Atoi(m[2])
	return episodeAt{season, episode}, m[3] != "", true
}
