package conversions

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/conversions/internal/db"
	"github.com/jatassi/slipstream/internal/conversions/internal/plan"
	"github.com/jatassi/slipstream/internal/encode"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

// settle puts back what each lapsed or written-off Node holds, one Write
// per Node, and moves its pinned retries on. It forgets the settings,
// proofs and causes of a written-off Node that holds nothing.
func (c *Conversions) settle(ctx context.Context, ros roster) ([]ID, error) {
	var placed, known []role.NodeID
	err := c.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		q := db.New(r)
		var err error
		if placed, err = q.ListPlacedNodes(ctx); err != nil {
			return err
		}
		known, err = q.ListKnownNodes(ctx)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("read the Nodes holding Conversions: %w", err)
	}
	var moved []ID
	for _, n := range placed {
		if !ros.holders.Lapsed(n) && !ros.holders.WrittenOff(n) {
			continue
		}
		ids, err := c.putBackLapsed(ctx, n)
		moved = append(moved, ids...)
		if err != nil {
			return moved, err
		}
	}
	for _, n := range known {
		if !ros.holders.WrittenOff(n) || slices.Contains(placed, n) {
			continue
		}
		if err := c.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
			return forget(ctx, db.New(tx), n)
		}); err != nil {
			return moved, fmt.Errorf("forget a written-off Node: %w", err)
		}
	}
	return moved, nil
}

// putBackLapsed puts back what n holds and moves its pinned retries on,
// in one Write.
func (c *Conversions) putBackLapsed(ctx context.Context, n role.NodeID) ([]ID, error) {
	var moved []ID
	err := c.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		q := db.New(tx)
		rows, err := q.ListPlacedOn(ctx, n)
		if err != nil {
			return err
		}
		pinned, err := q.ListPinnedTo(ctx, n)
		if err != nil {
			return err
		}
		moved = pinned
		for _, r := range rows {
			moved = append(moved, r.ID)
		}
		if err := c.putBack(ctx, tx, n); err != nil {
			return err
		}
		return c.unpin(ctx, tx, n)
	})
	if err != nil {
		return nil, fmt.Errorf("put back the Conversions on a lapsed Node: %w", err)
	}
	return moved, nil
}

func forget(ctx context.Context, q *db.Queries, n role.NodeID) error {
	if err := q.DeleteNodeSettings(ctx, n); err != nil {
		return err
	}
	if err := q.DeleteProofsOf(ctx, n); err != nil {
		return err
	}
	return q.DeleteCausesOf(ctx, n)
}

// place writes plan.Run's first step in one Write: it puts back what a
// transcode preempts on a Node that can't pause, fails what its container
// can't hold, places at most AtOnce+1 sources per Node and diffs the
// causes. due is the next time a queued row's wait ends.
func (c *Conversions) place(ctx context.Context, ros roster) (step plan.Step, moved []ID, due time.Time, err error) {
	err = c.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		moved, due, step = nil, time.Time{}, plan.Step{}
		s, err := c.snapshot(ctx, tx, ros)
		if err != nil {
			return err
		}
		ids, err := c.preempt(ctx, tx, s)
		if err != nil {
			return err
		}
		moved = append(moved, ids...)
		ids, err = c.unfit(ctx, tx, s)
		if err != nil {
			return err
		}
		moved = append(moved, ids...)
		step, _ = plan.Run(s.Snapshot, s.Now)
		for _, p := range step.Placed {
			proof := p.Proof
			id := ID{UUID: p.Job}
			if err := c.advance(ctx, tx, id, Event{
				On: Placed, Node: p.Node, Proof: &proof, Copy: p.Copy, Basis: Basis{Stage: Stage(p.Stage), Attempt: p.Attempt},
			}); err != nil {
				return err
			}
			moved = append(moved, id)
		}
		for _, j := range s.Jobs {
			if j.Stage == string(StageQueued) && j.NotBefore.After(s.Now) && (due.IsZero() || j.NotBefore.Before(due)) {
				due = j.NotBefore
			}
		}
		return c.causes(ctx, tx, s, step.Causes)
	})
	return step, moved, due, err
}

// preempt puts back each running or paused row on a Node a transcode
// holds whose proof can't pause, so the Node isn't asked to pause it.
func (c *Conversions) preempt(ctx context.Context, tx *storage.Tx[storage.Library], s *snap) ([]ID, error) {
	var moved []ID
	for ni, n := range s.Nodes {
		if !n.Hold {
			continue
		}
		for i, j := range s.Jobs {
			if j.Node == nil || *j.Node != n.ID || (j.Stage != string(StageRunning) && j.Stage != string(StagePaused)) || n.Proofs[j.Key].Pauses {
				continue
			}
			id := ID{UUID: j.ID}
			if err := c.advance(ctx, tx, id, Event{On: Preempted, Node: n.ID}); err != nil {
				return nil, err
			}
			moved = append(moved, id)
			s.Jobs[i].Stage, s.Jobs[i].Node = string(StageQueued), nil
			s.Nodes[ni].Running = slices.DeleteFunc(s.Nodes[ni].Running, func(r plan.Running) bool { return r.Job == j.ID })
		}
	}
	return moved, nil
}

// unfit fails each due queued row whose settings copy audio its container
// can't hold, which no Node can change, and takes it out of the plan.
func (c *Conversions) unfit(ctx context.Context, tx *storage.Tx[storage.Library], s *snap) ([]ID, error) {
	var moved []ID
	for i, j := range s.Jobs {
		if j.Stage != string(StageQueued) || j.NotBefore.After(s.Now) {
			continue
		}
		row := s.rows[ID{UUID: j.ID}]
		src, settings, ok := s.made(row)
		if !ok {
			continue
		}
		u := encode.Fits(settings, src.Media, nil)
		if u == nil {
			continue
		}
		if err := c.advance(ctx, tx, row.ID, Event{On: Fault, Code: Code(u.Code), Basis: Basis{Stage: row.Stage, Attempt: row.Attempt}}); err != nil {
			return nil, err
		}
		moved = append(moved, row.ID)
		s.Jobs[i].Source = storage.UUID{}
	}
	return moved, nil
}
