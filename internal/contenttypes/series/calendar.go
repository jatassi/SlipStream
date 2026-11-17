package series

import (
	"context"

	"github.com/jatassi/slipstream/internal/calendar"
	"github.com/jatassi/slipstream/internal/contenttypes/series/internal/db"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

var airKind = works.DatedKind{Key: "air", Label: "Air date"}

// Dated is each episode on its air date in s. The air delay moves when an
// episode counts as released, not its day.
func (*Series) Dated(ctx context.Context, r storage.Reader[file], s calendar.Span) ([]calendar.Dated, error) {
	rows, err := db.New(r).ListAiredIn(ctx, db.ListAiredInParams{FromDay: new(s.From().String()), ToDay: new(s.To().String())})
	if err != nil {
		return nil, err
	}
	out := make([]calendar.Dated, len(rows))
	for i, row := range rows {
		day, err := calendar.ParseDay(*row.AirDate)
		if err != nil {
			return nil, err
		}
		out[i] = calendar.Dated{Work: row.WorkID, Item: row.ItemID, Kind: airKind.Key, Day: day}
	}
	return out, nil
}
