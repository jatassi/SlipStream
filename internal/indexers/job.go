package indexers

import (
	"context"
	"fmt"
	"time"

	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/searching"
)

// ask runs job on node. It polls Run until the answer or the job's
// deadline, Timings.Job from when the Node started it, and never waits
// past the deadline once the start is known. A job past it is ErrLate, and
// settle polls it on in the background.
func (x *Indexers) ask(ctx context.Context, node role.NodeID, job searching.Job) (searching.Answer, error) {
	remote := x.cfg.Searching(node)
	var deadline time.Time
	for {
		asked := time.Now()
		poll, err := runUntil(ctx, remote, job, deadline)
		switch {
		case ctx.Err() != nil:
			return searching.Answer{}, fmt.Errorf("ask the Searching Node: %w", ctx.Err())
		case err == nil && poll.Answer != nil:
			return *poll.Answer, nil
		case !deadline.IsZero() && !time.Now().Before(deadline):
			go x.settle(context.WithoutCancel(ctx), remote, job)
			return searching.Answer{}, ErrLate
		case err != nil:
			return searching.Answer{}, err
		}
		if !poll.Started.IsZero() {
			deadline = poll.Started.Add(x.cfg.Timings.Job)
		}
		if err := pause(ctx, asked.Add(x.cfg.Timings.Poll), deadline); err != nil {
			return searching.Answer{}, err
		}
	}
}

// runUntil is one Run call, cut short at deadline unless it is zero.
func runUntil(ctx context.Context, remote searching.Remote, job searching.Job, deadline time.Time) (searching.Poll, error) {
	if !deadline.IsZero() {
		var cancel context.CancelFunc
		ctx, cancel = context.WithDeadline(ctx, deadline)
		defer cancel()
	}
	return remote.Run(ctx, job)
}

// pause waits until next, or until deadline when that comes first and
// isn't zero.
func pause(ctx context.Context, next, deadline time.Time) error {
	if !deadline.IsZero() && deadline.Before(next) {
		next = deadline
	}
	t := time.NewTimer(time.Until(next))
	defer t.Stop()
	select {
	case <-t.C:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("wait to ask again: %w", ctx.Err())
	}
}

// settle polls a late job until it answers, and records the answer. The
// Node ends every job within its cap, so the polling ends too.
func (x *Indexers) settle(ctx context.Context, remote searching.Remote, job searching.Job) {
	for {
		asked := time.Now()
		poll, err := remote.Run(ctx, job)
		if err != nil {
			x.cfg.Log.Warn("Couldn't read the response of an Indexer's late job", "indexer", job.Indexer, "job", job.ID, "err", err.Error())
			return
		}
		if poll.Answer != nil {
			x.record(job, *poll.Answer)
			return
		}
		_ = pause(ctx, asked.Add(x.cfg.Timings.Poll), time.Time{})
	}
}

// record keeps what a late answer says about its Indexer, whose Releases
// the search already went without.
func (x *Indexers) record(job searching.Job, a searching.Answer) {
	outcome := "ok"
	if a.Outcome != nil {
		outcome = a.Outcome.Error()
	}
	x.cfg.Log.Info("An Indexer's job finished after its deadline", "indexer", job.Indexer, "job", job.ID, "outcome", outcome, "releases", len(a.Releases))
}
