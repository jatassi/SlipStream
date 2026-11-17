package music

import (
	"context"
	"slices"

	"github.com/jatassi/slipstream/fakes/contenttypes/music/internal/db"
	"github.com/jatassi/slipstream/internal/calendar"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

var releaseKind = works.DatedKind{Key: "release", Label: "Release date"}

// Dated is every live track of each album released on a full day in s:
// an album, not a track, carries the date.
func (*Music) Dated(ctx context.Context, r storage.Reader[storage.Library], s calendar.Span) ([]calendar.Dated, error) {
	albums, err := db.New(r).ListAlbumsReleasedIn(ctx, db.ListAlbumsReleasedInParams{FromDay: s.From().String(), ToDay: s.To().String()})
	if err != nil || len(albums) == 0 {
		return nil, err
	}
	days := map[works.LevelID]calendar.Day{}
	var ids []works.WorkID
	for _, a := range albums {
		day, err := calendar.ParseDay(a.Released)
		if err != nil {
			return nil, err
		}
		days[a.LevelID] = day
		if !slices.Contains(ids, a.WorkID) {
			ids = append(ids, a.WorkID)
		}
	}
	items, err := works.ItemsOf(ctx, r, works.Only(ids))
	if err != nil {
		return nil, err
	}
	var out []calendar.Dated
	for _, w := range ids {
		for _, it := range items[w] {
			if day, ok := days[it.LevelID]; ok && it.Live() {
				out = append(out, calendar.Dated{Work: w, Item: it.ID, Kind: releaseKind.Key, Day: day})
			}
		}
	}
	return out, nil
}
