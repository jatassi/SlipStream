package grabs

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs/internal/db"
	"github.com/jatassi/slipstream/internal/grabs/internal/pace"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/searches"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// nudge asks, in tx, the caller's Write, for a search of work's items, nil
// for the whole Work, by trigger t in lane, zero for a gap reset only. It
// writes one grabs_nudge row, which the reconciler's drain lands on the
// queue, and wakes the reconciler, whose drain then waits for the Write to
// end. It reads nothing.
func (gr *Grabs) nudge(ctx context.Context, tx *storage.Tx[file], work works.WorkID, t searches.Trigger, lane pace.Lane, items []works.ItemID) error {
	p := db.InsertNudgeParams{WorkID: work, Trigger: string(t), CreatedAt: storage.Now()}
	if lane != 0 {
		p.Lane = new(int64(lane))
	}
	if items != nil {
		raw, err := json.Marshal(items)
		if err != nil {
			return fmt.Errorf("encode the nudge's Items: %w", err)
		}
		p.Items = new(string(raw))
	}
	if err := db.New(tx).InsertNudge(ctx, p); err != nil {
		return err
	}
	gr.wake()
	return nil
}

// drain lands the nudges of the Works nudged first, at most DrainBatch of
// them, on the queue in one Write, oldest first: Files.Wanted read once
// for them all, Facet.Groups once per Work, then each nudge's entry on the
// groups of its Items' cells its trigger opens. A Work that wants nothing
// queues nothing, and a deleted Work's nudges went with it. It isn't
// fenced: while Reconciling the rows wait for dispatch. It publishes
// grabs.queue-changed for the Works it landed, and wakes the reconciler
// again when more Works wait. It always opens its Write: a nudge wakes
// the reconciler before its Write commits, and a Read would miss it.
func (gr *Grabs) drain(ctx context.Context) error {
	interval, err := gr.interval(ctx)
	if err != nil {
		return err
	}
	var landed []works.WorkID
	more := false
	err = gr.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		var err error
		landed, more, err = gr.drainIn(ctx, tx, interval)
		return err
	})
	if err != nil {
		return err
	}
	if len(landed) > 0 {
		out := make([]string, len(landed))
		for i, w := range landed {
			out[i] = w.String()
		}
		gr.cfg.QueueChanged.Publish(ctx, QueueChanged{WorkIDs: out})
	}
	if more {
		gr.wake()
	}
	return nil
}

// drainIn lands the nudges of the Works nudged first, at most DrainBatch,
// in tx, answering the Works that want something and whether more Works
// may wait.
func (gr *Grabs) drainIn(ctx context.Context, tx *storage.Tx[file], interval time.Duration) (landed []works.WorkID, more bool, err error) {
	q := db.New(tx)
	ids, err := q.ListNudgedWorks(ctx, int64(gr.cfg.Timings.DrainBatch))
	if err != nil || len(ids) == 0 {
		return nil, false, err
	}
	nudges, err := q.ListNudgesOf(ctx, ids)
	if err != nil {
		return nil, false, err
	}
	wanted, err := gr.cfg.Files.Wanted(ctx, tx, works.Only(ids))
	if err != nil {
		return nil, false, err
	}
	for _, w := range wanted {
		groups, err := gr.cfg.Terms[w.Type].Groups(ctx, tx, w.Work, itemsOf(w.Cells))
		if err != nil {
			return nil, false, err
		}
		for _, n := range nudges {
			if n.WorkID != w.Work {
				continue
			}
			if err := landNudge(ctx, tx, n, w.Cells, groups, interval); err != nil {
				return nil, false, err
			}
		}
		landed = append(landed, w.Work)
	}
	return landed, len(ids) == gr.cfg.Timings.DrainBatch, q.DeleteNudgesOf(ctx, ids)
}

// landNudge lands n on groups, the search groups of the Work's wanted
// cells, each narrowed to the Items of n's cells its trigger opens.
func landNudge(ctx context.Context, tx *storage.Tx[file], n db.GrabsNudge, wanted []files.Cell, groups []indexers.Group, interval time.Duration) error {
	t := searches.Trigger(n.Trigger)
	r := laneRules[t]
	e := entry{trigger: t, lane: laneOf(n.Lane), gone: r.gone, resets: r.resets, holds: r.holds}
	var asked []works.ItemID
	if n.Items != nil {
		if err := json.Unmarshal([]byte(*n.Items), &asked); err != nil {
			return fmt.Errorf("nudge %d: decode its Items: %w", n.ID, err)
		}
	}
	var cells []files.Cell
	for _, c := range wanted {
		if opens(c, t, e.gone) && (n.Items == nil || slices.Contains(asked, c.Item.ID)) {
			cells = append(cells, c)
		}
	}
	_, err := land(ctx, tx, n.WorkID, narrowed(groups, itemsOf(cells)), cells, e, interval)
	return err
}

// narrowed is groups with only items in each, leaving out the groups that
// hold none of them.
func narrowed(groups []indexers.Group, items []works.ItemID) []indexers.Group {
	var out []indexers.Group
	for _, g := range groups {
		g.Items = slices.DeleteFunc(slices.Clone(g.Items), func(it works.ItemID) bool { return !slices.Contains(items, it) })
		if len(g.Items) > 0 {
			out = append(out, g)
		}
	}
	return out
}
