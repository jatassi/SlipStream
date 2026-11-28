package conversions

import (
	"context"
	"fmt"

	"github.com/jatassi/slipstream/internal/conversions/internal/db"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// drain converges a batch of nudged Works in one Write: it always opens a
// Write, since a Read could miss a nudge whose Write is still open. more
// is that nudges remain.
func (c *Conversions) drain(ctx context.Context) (changed []ID, more bool, err error) {
	err = c.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		changed, more, err = c.drainIn(ctx, tx)
		return err
	})
	return changed, more, err
}

func (c *Conversions) drainIn(ctx context.Context, tx *storage.Tx[storage.Library]) (changed []ID, more bool, err error) {
	q := db.New(tx)
	nudges, err := q.TakeNudges(ctx, int64(c.cfg.Timings.DrainBatch))
	if err != nil || len(nudges) == 0 {
		return nil, false, err
	}
	asks := asksOf(nudges)
	ws := make([]works.WorkID, len(asks))
	for i, a := range asks {
		ws[i] = a.Work
	}
	sourced, err := c.cfg.Files.Sourced(ctx, tx, works.Only(ws))
	if err != nil {
		return nil, false, fmt.Errorf("read the made-from cells: %w", err)
	}
	have, err := openOf(ctx, q, ws, sourced)
	if err != nil {
		return nil, false, err
	}
	goals, err := goalsOf(ctx, tx)
	if err != nil {
		return nil, false, err
	}
	summaries, err := works.SummariesOf(ctx, tx, ws)
	if err != nil {
		return nil, false, fmt.Errorf("read the Works: %w", err)
	}
	for _, a := range asks {
		ids, err := c.converge(ctx, tx, a, sourced[a.Work], have, goals, summaries[a.Work])
		if err != nil {
			return nil, false, err
		}
		changed = append(changed, ids...)
	}
	ids := make([]db.NudgeID, len(nudges))
	for i, n := range nudges {
		ids[i] = n.ID
	}
	return changed, len(nudges) == c.cfg.Timings.DrainBatch, q.DeleteNudges(ctx, ids)
}

// converge queues what a's cells want and cancels what they don't.
func (c *Conversions) converge(ctx context.Context, tx *storage.Tx[storage.Library], a ask, cells []files.Sourced, have open,
	goals map[targets.TargetID]targets.Target, work works.Summary,
) ([]ID, error) {
	q := db.New(tx)
	inserts, cancels := diff(have.rows[a.Work], desired(cells, goals, have.declined, a))
	var changed []ID
	for _, w := range inserts {
		id, err := c.enqueue(ctx, q, a, w, work, goals[w.Target].Name)
		if err != nil {
			return nil, err
		}
		changed = append(changed, id)
	}
	for _, x := range cancels {
		if err := c.advance(ctx, tx, x.ID, Event{On: Unneeded, Why: x.Why}); err != nil {
			return nil, err
		}
		changed = append(changed, x.ID)
	}
	if a.explicit() {
		return changed, clearDeclines(ctx, q, cells)
	}
	return changed, nil
}

// asksOf merges nudges by Work, oldest first. A Work keeps its first
// nudge's ask unless a later one is explicit and it isn't.
func asksOf(nudges []db.ConversionsNudge) []ask {
	var out []ask
	at := map[works.WorkID]int{}
	for _, n := range nudges {
		a := ask{Work: n.WorkID, Why: files.Arrival(n.Why), Since: n.Since.Time(), Backfill: n.BackfillID}
		i, ok := at[n.WorkID]
		switch {
		case !ok:
			at[n.WorkID] = len(out)
			out = append(out, a)
		case a.explicit() && !out[i].explicit():
			out[i] = a
		}
	}
	return out
}

// open is the unfinished Conversions of a batch's Works, failed included,
// and the declined pairs of their source files.
type open struct {
	rows     map[works.WorkID][]Row
	declined map[Pair]bool
}

func openOf(ctx context.Context, q *db.Queries, ws []works.WorkID, sourced map[works.WorkID][]files.Sourced) (open, error) {
	out := open{rows: map[works.WorkID][]Row{}, declined: map[Pair]bool{}}
	raws, err := q.ListOpenOfWorks(ctx, ws)
	if err != nil {
		return open{}, fmt.Errorf("read the Works' Conversions: %w", err)
	}
	for _, raw := range raws {
		r, err := rowOf(raw)
		if err != nil {
			return open{}, err
		}
		out.rows[r.Work] = append(out.rows[r.Work], r)
	}
	var srcs []files.ID
	for _, cells := range sourced {
		for _, c := range cells {
			if s, ok := c.Source(); ok {
				srcs = append(srcs, s.ID)
			}
		}
	}
	declined, err := q.ListDeclinedOf(ctx, srcs)
	if err != nil {
		return open{}, fmt.Errorf("read the declined pairs: %w", err)
	}
	for _, d := range declined {
		out.declined[Pair{Source: d.SourceFileID, Target: d.TargetID}] = true
	}
	return out, nil
}

func clearDeclines(ctx context.Context, q *db.Queries, cells []files.Sourced) error {
	for _, c := range cells {
		s, ok := c.Source()
		if !ok {
			continue
		}
		if err := q.DeleteDecline(ctx, db.DeleteDeclineParams{SourceFileID: s.ID, TargetID: c.Target}); err != nil {
			return fmt.Errorf("clear a decline: %w", err)
		}
	}
	return nil
}

func goalsOf(ctx context.Context, r storage.Reader[storage.Library]) (map[targets.TargetID]targets.Target, error) {
	all, err := targets.All(ctx, r)
	if err != nil {
		return nil, fmt.Errorf("read the Targets: %w", err)
	}
	out := make(map[targets.TargetID]targets.Target, len(all))
	for _, t := range all {
		out[t.ID] = t
	}
	return out, nil
}
