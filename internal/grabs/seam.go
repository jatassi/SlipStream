package grabs

import (
	"context"

	"github.com/jatassi/slipstream/internal/searches"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// Queue is searches.Queue: it nudges each Ask in tx, the caller's Write,
// in the lane laneRules gives t for one Work or several. The drain lands
// the nudges once the Write commits.
func (gr *Grabs) Queue(ctx context.Context, tx *storage.Tx[file], t searches.Trigger, asks ...searches.Ask) error {
	lane := landing(t, len(asks) > 1).lane
	for _, a := range asks {
		if err := gr.nudge(ctx, tx, a.Work, t, lane, a.Items); err != nil {
			return err
		}
	}
	return nil
}

// Released is metadata.Released: it nudges items, whose release date
// arrived, in its own Write. Calling it again for the same Items lands on
// the same queue rows.
func (gr *Grabs) Released(ctx context.Context, items []works.ItemID) error {
	return gr.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		byWork, err := works.WorksOfItems(ctx, tx, items)
		if err != nil {
			return err
		}
		asks := make([]searches.Ask, 0, len(byWork))
		for w, its := range byWork {
			asks = append(asks, searches.Ask{Work: w, Items: its})
		}
		return gr.Queue(ctx, tx, searches.ReleaseDate, asks...)
	})
}
