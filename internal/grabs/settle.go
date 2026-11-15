package grabs

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/grabs/internal/db"
	"github.com/jatassi/slipstream/internal/grabs/internal/pace"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// heardRun is what a run's lines heard: their Releases in the order they
// landed, each one's Indexer name by that order, whether any asked
// Indexer answered, the Indexers that answered its group's query, and the
// position its next line takes.
type heardRun struct {
	releases []indexers.Release
	names    map[int]string
	answered bool
	asked    []indexers.Line
	next     int64
}

// settle runs once every line of run answered, was skipped or went late:
// it assesses what the lines heard, and in one fenced Write either plans
// the run's second phase or decides, keeps every Release with its outcome
// and finishes the run.
func (gr *Grabs) settle(ctx context.Context, run Search) result {
	var h heardRun
	var j judging
	err := gr.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		if h, err = gr.linesHeard(ctx, r, run.ID); err != nil {
			return err
		}
		tree, err := gr.cfg.Works.TreeIn(ctx, r, run.WorkID)
		if err != nil {
			return err
		}
		expect, err := gr.cfg.Terms[tree.ContentType].Expect(ctx, r, run.WorkID)
		if err != nil {
			return err
		}
		j, err = gr.judgingOf(ctx, r, tree.ContentType, run.WorkID, searchedOf(run), expect)
		return err
	})
	if err != nil {
		return gr.searchWaits(ctx, run, err)
	}
	interval, err := gr.interval(ctx)
	if err != nil {
		return gr.searchWaits(ctx, run, err)
	}
	grabs, planned, err := gr.decideRun(ctx, run, h, assessAll(j, dedupe(listedOf(h.releases))), interval)
	switch {
	case errors.Is(err, errRunGone):
		return committed
	case errors.Is(err, ErrReconciling):
		return waited
	case err != nil:
		return gr.searchWaits(ctx, run, err)
	}
	gr.cfg.QueueChanged.Publish(ctx, QueueChanged{WorkIDs: []string{run.WorkID.String()}})
	gr.wake()
	if planned {
		return committed
	}
	gr.cfg.SearchChanged.Publish(ctx, SearchChanged{SearchID: run.ID.String(), WorkID: run.WorkID.String()})
	for _, g := range grabs {
		gr.cfg.Indexers.Taken(ctx, g.Ticket)
		gr.cfg.Changed.Publish(ctx, Changed{GrabID: g.ID.String(), WorkID: g.WorkID.String()})
	}
	return committed
}

// searchedOf is the Item a Release that run heard must cover: its one
// Item, zero for a run of several, whose cells judge what each Release
// covers.
func searchedOf(run Search) works.ItemID {
	if len(run.Items) == 1 {
		return run.Items[0]
	}
	return works.ItemID{}
}

// decideRun is run's one fenced Write over as, what h heard assessed. A
// first phase that leaves Items it may search alone plans their lines and
// reports planned. Otherwise it decides, keeps every Release with its
// outcome and finishes the run, answering the Grabs the run made or
// joined. It answers errRunGone once run is gone or finished.
func (gr *Grabs) decideRun(ctx context.Context, run Search, h heardRun, as []assessed, interval time.Duration) ([]Grab, bool, error) {
	var grabs []Grab
	planned := false
	err := gr.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		if err := gr.fence(ctx, tx); err != nil {
			return err
		}
		q := db.New(tx)
		row, err := q.GetSearch(ctx, run.ID)
		if errors.Is(err, storage.ErrNoRows) || err == nil && !row.FinishedAt.IsZero() {
			return errRunGone
		}
		if err != nil {
			return err
		}
		if row.Phase == 1 {
			singles, err := gr.singlesOf(ctx, tx, run, laneOf(row.Lane), h, as, interval)
			if err != nil {
				return err
			}
			if len(singles) > 0 {
				planned = true
				return planSingles(ctx, q, run.ID, singles, h)
			}
		}
		if err := q.DeleteHeard(ctx, run.ID); err != nil {
			return err
		}
		grabs, err = gr.decided(ctx, tx, run, h, as, interval)
		return err
	})
	return grabs, planned, err
}

// decided decides run in tx over as, stores every Release with its
// outcome and finishes the run under Backlog search's interval, answering
// the Grabs it made or joined.
func (gr *Grabs) decided(ctx context.Context, tx *storage.Tx[file], run Search, h heardRun, as []assessed, interval time.Duration) ([]Grab, error) {
	d, err := gr.decideIn(ctx, tx, runInput(run, as))
	if err != nil {
		return nil, err
	}
	if err := gr.storeReleases(ctx, db.New(tx), run.ID, d.rows, h.names, len(d.grabs) > 0, nil); err != nil {
		return nil, err
	}
	return d.grabs, gr.finishRun(ctx, tx, run, d, h, interval)
}

// singlesOf is the Items of run, a first phase queued in lane, that its
// group's query left without a grabbable Release at an open cell and that
// may each be searched alone: all of them on a press or once the group's
// gap reached the longest gap, else those no run searched before. A run of
// one Item, or one no Indexer answered, has none.
func (gr *Grabs) singlesOf(ctx context.Context, tx *storage.Tx[file], run Search, lane pace.Lane, h heardRun, as []assessed, interval time.Duration) ([]works.ItemID, error) {
	if len(run.Items) < 2 || len(h.asked) == 0 {
		return nil, nil
	}
	uncovered, err := gr.uncovered(ctx, tx, run, as)
	if err != nil || len(uncovered) == 0 || lane == pace.Now {
		return uncovered, err
	}
	q := db.New(tx)
	switch longest, err := atLongest(ctx, q, run, interval); {
	case err != nil:
		return nil, err
	case longest:
		return uncovered, nil
	}
	searched, err := q.ListSearchedItems(ctx, uncovered)
	if err != nil {
		return nil, err
	}
	return slices.DeleteFunc(uncovered, func(it works.ItemID) bool { return slices.Contains(searched, it) }), nil
}

// uncovered is the Items of run with an open cell no Release of as is
// grabbable at, in tree order. A Release with a refusal of its own, the
// Blocklist's included, covers nothing.
func (gr *Grabs) uncovered(ctx context.Context, tx *storage.Tx[file], run Search, as []assessed) ([]works.ItemID, error) {
	all, err := gr.cfg.Files.Cells(ctx, tx, run.WorkID)
	if err != nil {
		return nil, err
	}
	bl, err := blocksOf(ctx, tx, as)
	if err != nil {
		return nil, err
	}
	c := runInput(run, as).board(all, nil)
	covered := map[works.ItemID]bool{}
	for _, j := range decide(c, bl.gate(as)).Ranked {
		if len(j.refusals) > 0 {
			continue
		}
		for _, v := range j.verdicts {
			if v.grabbable() && v.cell < len(c.open) {
				covered[c.open[v.cell].Item.ID] = true
			}
		}
	}
	var out []works.ItemID
	for _, cell := range c.open {
		if id := cell.Item.ID; !covered[id] && !slices.Contains(out, id) {
			out = append(out, id)
		}
	}
	return out, nil
}

// atLongest reports whether run's group's gap reached the longest gap,
// which is never under Backlog search's interval.
func atLongest(ctx context.Context, q *db.Queries, run Search, interval time.Duration) (bool, error) {
	due, err := q.GetDue(ctx, db.GetDueParams{WorkID: run.WorkID, GroupKey: run.Group})
	if err != nil {
		return false, err
	}
	s, err := q.GetSettings(ctx)
	if err != nil {
		return false, err
	}
	return time.Duration(due.GapSeconds)*time.Second >= max(LongestGap.Value(s.LongestGapSeconds), interval), nil
}

// planSingles moves run to its second phase with a line for each of
// items at each Indexer that answered its group's query, which dispatch
// sends as it sent the first.
func planSingles(ctx context.Context, q *db.Queries, run SearchID, items []works.ItemID, h heardRun) error {
	next := h.next
	for _, it := range items {
		for _, l := range h.asked {
			if err := q.InsertSingleLine(ctx, db.InsertSingleLineParams{SearchID: run, Position: next, IndexerID: l.Indexer, IndexerName: l.Name, ItemID: it}); err != nil {
				return err
			}
			next++
		}
	}
	return q.StartPhaseTwo(ctx, run)
}

// linesHeard is what run's lines heard, read in r.
func (gr *Grabs) linesHeard(ctx context.Context, r storage.Reader[file], run SearchID) (heardRun, error) {
	q := db.New(r)
	h := heardRun{names: map[int]string{}}
	lines, err := q.ListSearchIndexersOfSearch(ctx, run)
	if err != nil {
		return heardRun{}, err
	}
	for _, l := range lines {
		if o := l.Outcome; o != nil && (*o == indexers.Found || *o == indexers.NoReleases) {
			h.answered = true
			if l.ItemID.IsZero() {
				h.asked = append(h.asked, indexers.Line{Indexer: l.IndexerID, Name: l.IndexerName})
			}
		}
		h.next = max(h.next, l.Position+1)
	}
	rows, err := q.ListHeard(ctx, run)
	if err != nil {
		return heardRun{}, err
	}
	for i, row := range rows {
		rel, err := gr.kept(row)
		if err != nil {
			return heardRun{}, err
		}
		h.releases = append(h.releases, rel)
		h.names[i] = row.IndexerName
	}
	return h, nil
}
