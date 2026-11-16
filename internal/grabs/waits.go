package grabs

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs/internal/db"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// GrabWaiting is Grab now: it ends work's waits at cells, or every wait
// of the Work when cells is empty, at once, and answers how many Releases
// it grabs. It fails with ErrReconciling and ErrNoWork.
func (gr *Grabs) GrabWaiting(ctx context.Context, _ identity.Admin, work works.WorkID, cells []files.CellKey) (int, error) {
	ended, err := gr.endWaits(ctx, work, cells, true)
	if err != nil {
		return 0, err
	}
	gr.ended(ctx, work, ended)
	return len(ended.grabs), nil
}

// waitEnds is what the reconciler ends on a pass: each Work with a wait
// due, and when the next wait after those ends, zero when none does.
type waitEnds struct {
	works []works.WorkID
	next  time.Time
}

func waitEndsIn(ctx context.Context, q *db.Queries, now storage.Time) (waitEnds, error) {
	ws, err := q.ListWorksWithWaitsEnding(ctx, now)
	if err != nil {
		return waitEnds{}, err
	}
	next, err := q.NextWaitEnd(ctx, now)
	if errors.Is(err, storage.ErrNoRows) {
		return waitEnds{works: ws}, nil
	}
	return waitEnds{works: ws, next: next.Time()}, err
}

// endDue ends work's waits that are due, the reconciler's job for it.
func (gr *Grabs) endDue(ctx context.Context, work works.WorkID) result {
	ended, err := gr.endWaits(ctx, work, nil, false)
	switch {
	case errors.Is(err, ErrReconciling):
		return waited
	case errors.Is(err, ErrNoWork):
		return committed
	case err != nil:
		if ctx.Err() != nil {
			return halted
		}
		gr.cfg.Log.Error("end the waits of a Work", "work", work.String(), "err", err.Error())
		return waited
	}
	gr.ended(ctx, work, ended)
	return committed
}

// waitsEnded is what ending waits changed: how many waits ended, the
// Grabs it made or joined and the runs whose candidates the waits held.
type waitsEnded struct {
	waits int
	grabs []Grab
	runs  []SearchID
}

// ended tells the Indexers and the Admin what ended changed.
func (gr *Grabs) ended(ctx context.Context, work works.WorkID, ended waitsEnded) {
	if ended.waits == 0 {
		return
	}
	for _, g := range ended.grabs {
		gr.cfg.Indexers.Taken(ctx, g.Ticket)
		gr.cfg.Changed.Publish(ctx, Changed{GrabID: g.ID.String(), WorkID: work.String()})
	}
	for _, run := range ended.runs {
		gr.cfg.SearchChanged.Publish(ctx, SearchChanged{SearchID: run.String(), WorkID: work.String()})
	}
	gr.cfg.QueueChanged.Publish(ctx, QueueChanged{WorkIDs: []string{work.String()}})
	gr.wake()
}

// candidate is a Release a wait kept: its run and position there.
type candidate struct {
	run      SearchID
	position int64
}

// endWaits ends, in one fenced Write, work's waits at cells, every one
// when cells is empty, that are due, or all of them when now. Taking them
// in that Write is what ends a wait once. It decides again over every
// candidate they kept, with a fresh Blocklist read and no Hold, at the
// waiting cells, and grabs what the walk picks. A cell that filled
// meanwhile isn't open, so its wait ends with nothing taken. Grab now is
// the Admin's, so it opens gone and stopped cells too.
func (gr *Grabs) endWaits(ctx context.Context, work works.WorkID, cells []files.CellKey, now bool) (waitsEnded, error) {
	var out waitsEnded
	err := gr.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		if err := gr.fence(ctx, tx); err != nil {
			return err
		}
		tree, err := gr.cfg.Works.TreeIn(ctx, tx, work)
		if err != nil {
			return err
		}
		q := db.New(tx)
		at := storage.Now()
		taken, err := takeWaits(ctx, q, work, cells, now, at)
		if err != nil || len(taken) == 0 {
			return err
		}
		kept, err := q.ListWaitCandidatesOfWork(ctx, work)
		if err != nil {
			return err
		}
		if err := deleteWaits(ctx, q, taken); err != nil {
			return err
		}
		in, of, err := gr.waitDecision(ctx, tx, tree, taken, kept, now)
		if err != nil {
			return err
		}
		out, err = gr.grabCandidates(ctx, tx, in, of, at)
		out.waits = len(taken)
		return err
	})
	return out, err
}

// waitDecision is the decision of tree's waits ending at taken, over
// the candidates of kept those waits held, assessed and gated by the
// Blocklist, and each candidate's run and position by where it stands.
func (gr *Grabs) waitDecision(ctx context.Context, tx *storage.Tx[file], tree works.Tree, taken []files.CellKey, kept []db.ListWaitCandidatesOfWorkRow, now bool) (decisionIn, []candidate, error) {
	var ls []listed
	var of []candidate
	for _, k := range kept {
		c := candidate{k.GrabsRelease.SearchID, k.GrabsRelease.Position}
		if !slices.Contains(taken, files.CellKey{Item: k.ItemID, Edition: k.Edition, Target: k.TargetID}) || slices.Contains(of, c) {
			continue
		}
		r, err := gr.kept(k.GrabsRelease)
		if err != nil {
			return decisionIn{}, nil, err
		}
		ls, of = append(ls, listed{Release: r, at: len(of)}), append(of, c)
	}
	in := waitInput(tree.ID, taken, now)
	expect, err := gr.cfg.Terms[tree.ContentType].Expect(ctx, tx, tree.ID)
	if err != nil {
		return decisionIn{}, nil, err
	}
	j, err := gr.judgingOf(ctx, tx, tree, scopeOf(in.run.Items), expect)
	if err != nil {
		return decisionIn{}, nil, err
	}
	as := assessAll(j, dedupe(ls))
	bl, err := blocksOf(ctx, tx, as)
	if err != nil {
		return decisionIn{}, nil, err
	}
	in.as = bl.gate(as)
	return in, of, nil
}

// takeWaits is work's waits at cells, every one when cells is empty,
// that end: those due at at, or all of them when now.
func takeWaits(ctx context.Context, q *db.Queries, work works.WorkID, cells []files.CellKey, now bool, at storage.Time) ([]files.CellKey, error) {
	rows, err := q.ListWaitsOfWork(ctx, work)
	if err != nil {
		return nil, err
	}
	var out []files.CellKey
	for _, w := range rows {
		k := files.CellKey{Item: w.ItemID, Edition: w.Edition, Target: w.TargetID}
		if (now || w.UntilAt.Compare(at) <= 0) && (len(cells) == 0 || slices.Contains(cells, k)) {
			out = append(out, k)
		}
	}
	return out, nil
}

func deleteWaits(ctx context.Context, q *db.Queries, cells []files.CellKey) error {
	for _, k := range cells {
		if err := q.DeleteWait(ctx, db.DeleteWaitParams{ItemID: k.Item, Edition: k.Edition, TargetID: k.Target}); err != nil {
			return err
		}
	}
	return nil
}

// waitInput is the decision of work's waits ending at cells. Its
// candidates keep the runs that found them, so its run stands for the
// Work and the cells' Items alone. It opens only those cells and applies
// no Hold, since the Hold is what ended. Gone and stopped cells open only
// for Grab now, the Admin's.
func waitInput(work works.WorkID, cells []files.CellKey, now bool) decisionIn {
	in := decisionIn{run: Search{WorkID: work}, open: openRule{gone: now, stopped: now}, only: map[files.CellKey]bool{}}
	for _, k := range cells {
		in.only[k] = true
		if !slices.Contains(in.run.Items, k.Item) {
			in.run.Items = append(in.run.Items, k.Item)
		}
	}
	return in
}

// grabCandidates walks in, a wait's decision over the candidates of, by
// position, and commits each claim to the run whose Release it took: a
// Grab, or a join, and the run's link to it. The taken Release reads
// grabbed and its run took one more Grab. Every other candidate no wait
// still holds reads kept.
func (gr *Grabs) grabCandidates(ctx context.Context, tx *storage.Tx[file], in decisionIn, of []candidate, at storage.Time) (waitsEnded, error) {
	w, err := gr.walkIn(ctx, tx, in, at.Time())
	if err != nil {
		return waitsEnded{}, err
	}
	q := db.New(tx)
	var out waitsEnded
	for _, c := range of {
		if !slices.Contains(out.runs, c.run) {
			out.runs = append(out.runs, c.run)
		}
	}
	for _, cl := range w.claims {
		c := of[cl.pick.at]
		gs, err := gr.commitClaims(ctx, tx, Search{ID: c.run, WorkID: in.run.WorkID}, []claim{cl}, w.held, at)
		if err != nil {
			return waitsEnded{}, err
		}
		out.grabs = append(out.grabs, gs...)
		if err := q.GrabWaitingRelease(ctx, db.GrabWaitingReleaseParams{SearchID: c.run, Position: c.position}); err != nil {
			return waitsEnded{}, err
		}
		notes, err := encodeReasons(cl.pick.Notes)
		if err != nil {
			return waitsEnded{}, err
		}
		if err := q.AddWaitGrab(ctx, db.AddWaitGrabParams{ID: c.run, Notes: &notes}); err != nil {
			return waitsEnded{}, err
		}
	}
	return out, q.KeepEndedCandidates(ctx, out.runs)
}
