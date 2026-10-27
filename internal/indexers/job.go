package indexers

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/rpc"
	"github.com/jatassi/slipstream/internal/searching"
	"github.com/jatassi/slipstream/internal/storage"
)

// job is one call for one Indexer on one Node. Its id is its query row's.
type job struct {
	q    QueryID
	idx  Indexer
	use  Use
	node role.NodeID
	wire searching.Job
}

// answer is a as record reads it.
func (j job) answer(a searching.Answer) answer {
	out := answer{
		use: j.use, node: j.node, revision: j.idx.Revision, sha: pinned(j.idx), started: a.Started, outcome: a.Outcome,
		session: a.Session, releases: len(a.Releases),
	}
	if a.Report != nil && a.Outcome == nil {
		out.caps = &Caps{Modes: a.Report.Capabilities.Modes, Categories: CategoriesOf(a.Report.Capabilities.Categories)}
	}
	return out
}

// prepare is c for idx as node runs it, with idx's secrets and its session
// on node opened for the length of one job.
func (x *Indexers) prepare(ctx context.Context, q QueryID, idx Indexer, u Use, node role.NodeID, c searching.Call) (job, error) {
	secrets, err := x.open(idx)
	if err != nil {
		return job{}, err
	}
	session, err := x.session(ctx, x.cfg.Library, idx.ID, node)
	if err != nil {
		return job{}, err
	}
	w := searching.Job{
		ID: q.String(), Indexer: idx.ID.String(), Revision: idx.Revision, Kind: kindOnWire[idx.Kind], Address: secrets.Address,
		APIKey: secrets.APIKey, Session: session, Call: c,
	}
	if d, ok := idx.Source.(Definition); ok {
		w.Address = d.BaseAddress
		w.Settings = map[string]string{}
		for k, v := range d.Plain {
			w.Settings[k] = v
		}
		for k, v := range secrets.Fields {
			w.Settings[k] = v
		}
	}
	return job{q: q, idx: idx, use: u, node: node, wire: w}, nil
}

var kindOnWire = map[Kind]searching.Kind{Catalog: searching.Catalog, Torznab: searching.Torznab, RSS: searching.RSS}

// routed is the job running c for idx as query q on the Node idx routes
// to now, or *WaitError while no Node may take it. The caller reserved q.
func (x *Indexers) routed(ctx context.Context, q QueryID, idx Indexer, u Use, c searching.Call) (job, error) {
	var st Status
	err := x.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		st, err = status(ctx, r, idx.ID)
		return err
	})
	if err != nil {
		return job{}, err
	}
	node, wait := route(x.cfg.Roster, idx, st)
	if wait != nil {
		return job{}, &WaitError{Outcome: *wait}
	}
	return x.prepare(ctx, q, idx, u, node, c)
}

// run asks j's Node and records its answer, returning it with the status
// record wrote, zero when the answer was from an older revision. It fails
// with *UnavailableError when the Node couldn't run it, completing its
// query as lost, or ErrLate, which settle records once the Node answers.
func (x *Indexers) run(ctx context.Context, j job) (searching.Answer, Status, error) {
	a, err := x.ask(ctx, j)
	if err != nil {
		if !errors.Is(err, ErrLate) {
			x.lose(ctx, j.q)
		}
		return searching.Answer{}, Status{}, err
	}
	st, err := x.record(context.WithoutCancel(ctx), j.q, j.idx.ID, j.answer(a))
	return a, st, err
}

// lose completes q as lost: its job never answered.
func (x *Indexers) lose(ctx context.Context, q QueryID) {
	err := x.cfg.Library.Write(context.WithoutCancel(ctx), func(ctx context.Context, tx *storage.Tx[file]) error {
		return completeLost(ctx, tx, q, storage.Now())
	})
	if err != nil {
		x.cfg.Log.Error("record a job that never answered", "job", q.String(), "err", err.Error())
	}
}

// ask runs j on its Node. It polls Run until the answer or the job's
// deadline, Timings.Job from when the Node started it, and never waits
// past the deadline once the start is known. A job past it is ErrLate,
// and settle polls it on in the background and records it.
func (x *Indexers) ask(ctx context.Context, j job) (searching.Answer, error) {
	remote := x.cfg.Searching(j.node)
	var deadline time.Time
	for {
		asked := time.Now()
		poll, err := runUntil(ctx, remote, j.wire, deadline)
		switch {
		case ctx.Err() != nil:
			return searching.Answer{}, fmt.Errorf("ask the Searching Node: %w", ctx.Err())
		case err == nil && poll.Answer != nil:
			return *poll.Answer, nil
		case !deadline.IsZero() && !time.Now().Before(deadline):
			if !x.late.goOn(func(ctx context.Context) { x.settle(ctx, remote, j) }) {
				x.cfg.Log.Warn("An Indexer's late job won't be recorded because the Indexers are stopping", "indexer", j.idx.ID.String(), "job", j.q.String())
			}
			return searching.Answer{}, ErrLate
		case err != nil:
			return searching.Answer{}, unavailable(err)
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
// Node ends every job within its cap, so the polling ends too, and so
// does ctx, the Indexers' own.
func (x *Indexers) settle(ctx context.Context, remote searching.Remote, j job) {
	for {
		asked := time.Now()
		poll, err := remote.Run(ctx, j.wire)
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			x.cfg.Log.Warn("Couldn't read the response of an Indexer's late job", "indexer", j.idx.ID.String(), "job", j.q.String(), "err", err.Error())
			x.lose(ctx, j.q)
			return
		}
		if poll.Answer != nil {
			if _, err := x.record(ctx, j.q, j.idx.ID, j.answer(*poll.Answer)); err != nil && ctx.Err() == nil {
				x.cfg.Log.Error("record an Indexer's late answer", "indexer", j.idx.ID.String(), "job", j.q.String(), "err", err.Error())
			}
			return
		}
		if pause(ctx, asked.Add(x.cfg.Timings.Poll), time.Time{}) != nil {
			return
		}
	}
}

func unavailable(err error) error {
	if why, ok := rpc.NotSent(err); ok {
		return &UnavailableError{Wait: why != rpc.NotHolder, Err: err}
	}
	return &UnavailableError{Wait: rpc.Lost(err), Err: err}
}
