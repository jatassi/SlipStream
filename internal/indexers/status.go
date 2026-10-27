package indexers

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jatassi/slipstream/internal/indexers/internal/db"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/searching"
	"github.com/jatassi/slipstream/internal/storage"
)

// Cause is why an Indexer stopped.
type Cause = searching.StopCause

// NodeBound reports whether c belongs to the Node it was met on: met on a
// Node the Indexer reached by failover it makes the Indexer wait for home,
// and met on home it stops it.
func NodeBound(c Cause) bool {
	switch c {
	case searching.ChallengeUnsolved, searching.CaptchaNeeded, searching.CookieExpired:
		return true
	case searching.CredentialsRejected, searching.DefinitionUnloadable, searching.AccountRefused, searching.RequestRejected:
	}
	return false
}

// Retried reports whether the health check may Test an Indexer stopped
// for c again: false where a repeated sign-in would repeat what the site
// rejected.
func Retried(c Cause) bool {
	switch c {
	case searching.CredentialsRejected, searching.CaptchaNeeded, searching.CookieExpired, searching.AccountRefused:
		return false
	case searching.ChallengeUnsolved, searching.DefinitionUnloadable, searching.RequestRejected:
	}
	return true
}

type State string

const (
	OK       State = "ok"
	Limited  State = "limited"
	Failing  State = "failing"
	Stopped  State = "stopped"
	HomeOnly State = "home_only"
)

// Status is how an Indexer stands after the answers so far. Only record
// writes it.
type Status struct {
	State State
	// Rung is the backoff ladder's position. It steps down one on each
	// success.
	Rung int
	// Until is when a limited or failing Indexer may be asked again.
	Until time.Time
	// Since is when a failing Indexer first failed, or when it stopped or
	// began waiting for home.
	Since time.Time
	// Cause is set while it is stopped or home_only.
	Cause Cause
	// Text is the site's or feed's own words.
	Text string
	// Node is where the answer that stopped it came from.
	Node        role.NodeID
	SolverTried bool
	Last        *Failure
	Success     time.Time
	// Revision is the Indexer revision of the answer that wrote it.
	Revision int64
}

type Failure struct {
	At   time.Time
	Text string
}

// answer is one job's answer as record reads it.
type answer struct {
	use      Use
	node     role.NodeID
	revision int64
	sha      string
	started  time.Time
	// outcome is nil when the site responded, else a *searching.Limited,
	// *searching.Failing or *searching.Stopped.
	outcome  error
	session  *searching.Session
	releases int
	// caps are a passing Test's capabilities.
	caps *Caps
}

// next is s after a: pure. A success steps the rung down and clears
// limited and failing, and a passing Test also clears a stop and a wait
// for home. A failure climbs the rung. Limited waits for Retry-After, else
// Timings.Limited. A node-bound stop met on a Node other than home waits
// for home. Any answer from home ends a wait for home first.
func next(s Status, a answer, home role.NodeID, now time.Time, t Timings) Status {
	s.Revision = a.revision
	if s.State == HomeOnly && a.node == home {
		s = cleared(s, OK)
	}
	sticky := s.State == Stopped || s.State == HomeOnly
	var limited *searching.Limited
	var failing *searching.Failing
	var stopped *searching.Stopped
	switch {
	case a.outcome == nil:
		s.Success = now
		s.Rung = max(s.Rung-1, 0)
		if !sticky || a.use == UseTest {
			s = cleared(s, OK)
		}
	case errors.As(a.outcome, &limited):
		if !sticky {
			s = cleared(s, Limited)
			s.Until = now.Add(cmp.Or(max(limited.RetryAfter, 0), t.Limited))
		}
	case errors.As(a.outcome, &failing):
		s.Last = &Failure{At: now, Text: failing.Error()}
		if !sticky {
			s = climbed(s, now, t)
		}
	case errors.As(a.outcome, &stopped):
		s = stop(s, stopped, a.node, home, now)
	}
	return s
}

// climbed is s failing once more: one rung up, waiting that rung's time.
func climbed(s Status, now time.Time, t Timings) Status {
	since := s.Since
	if s.State != Failing {
		since = now
	}
	s.Rung = min(s.Rung+1, len(t.Ladder))
	s = cleared(s, Failing)
	s.Since, s.Until = since, now.Add(t.Ladder[s.Rung-1])
	return s
}

// stop is s after a stop met on node. A wait for home doesn't soften a
// stop.
func stop(s Status, stopped *searching.Stopped, node, home role.NodeID, now time.Time) Status {
	s.Last = &Failure{At: now, Text: stopped.Error()}
	state := Stopped
	if NodeBound(stopped.Cause) && node != home {
		state = HomeOnly
	}
	if s.State == Stopped && state == HomeOnly {
		return s
	}
	s = cleared(s, state)
	s.Since, s.Cause, s.Text, s.Node, s.SolverTried = now, stopped.Cause, stopped.Text, node, stopped.SolverTried
	return s
}

// cleared is s in state with nothing of its last state's.
func cleared(s Status, state State) Status {
	return Status{State: state, Rung: s.Rung, Last: s.Last, Success: s.Success, Revision: s.Revision}
}

// record writes one answer's effects in one Write: the query row, once,
// then the status, the session and a passing Test's caps, unless the
// answer's revision or sha is older than the Indexer's. It returns the
// status it wrote, zero when it wrote none, and publishes StatusChanged.
func (x *Indexers) record(ctx context.Context, q QueryID, idx ID, a answer) (Status, error) {
	var st Status
	err := x.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		var err error
		st, err = x.recordIn(ctx, tx, q, idx, a, storage.Now())
		return err
	})
	if err != nil {
		return Status{}, fmt.Errorf("record an answer of Indexer %s: %w", idx, err)
	}
	if st.State != "" {
		x.cfg.Status.Publish(ctx, StatusChanged{IndexerID: idx.String()})
	}
	return st, nil
}

// recordIn is record inside tx. An answer for a deleted Indexer writes
// nothing.
func (x *Indexers) recordIn(ctx context.Context, tx *storage.Tx[file], q QueryID, id ID, a answer, now storage.Time) (Status, error) {
	if err := complete(ctx, tx, q, a, now); err != nil {
		return Status{}, err
	}
	idx, err := Get(ctx, tx, id)
	if errors.Is(err, ErrNoIndexer) {
		return Status{}, nil
	}
	if err != nil {
		return Status{}, err
	}
	if a.revision < idx.Revision || a.sha != pinned(idx) {
		return Status{}, nil
	}
	st, err := status(ctx, tx, id)
	if err != nil {
		return Status{}, err
	}
	st = next(st, a, idx.Home, now.Time(), x.cfg.Timings)
	if err := putStatus(ctx, tx, id, st); err != nil {
		return Status{}, err
	}
	if a.session != nil {
		if err := x.offer(ctx, tx, id, a.node, *a.session); err != nil {
			return Status{}, err
		}
	}
	if a.caps != nil {
		caps, err := json.Marshal(a.caps)
		if err != nil {
			return Status{}, fmt.Errorf("encode the caps: %w", err)
		}
		if err := db.New(tx).SetCaps(ctx, db.SetCapsParams{ID: id, Caps: string(caps)}); err != nil {
			return Status{}, err
		}
	}
	return st, nil
}

// pinned is the definition sha a catalog Indexer runs, "" for a feed.
func pinned(idx Indexer) string {
	if d, ok := idx.Source.(Definition); ok {
		return d.SHA
	}
	return ""
}

func status(ctx context.Context, r storage.Reader[file], id ID) (Status, error) {
	row, err := db.New(r).GetStatus(ctx, id)
	if errors.Is(err, storage.ErrNoRows) {
		return Status{State: OK}, nil
	}
	if err != nil {
		return Status{}, err
	}
	return statusOf(row), nil
}

func statusOf(row db.IndexersStatus) Status {
	s := Status{
		State: State(row.State), Rung: int(row.Rung), Until: row.UntilAt.Time(), Since: row.SinceAt.Time(), Cause: Cause(deref(row.Cause)),
		Text: deref(row.Text), Node: row.NodeID, SolverTried: row.SolverTried, Success: row.SuccessAt.Time(), Revision: row.Revision,
	}
	if row.LastFailure != nil {
		s.Last = &Failure{At: row.LastFailureAt.Time(), Text: *row.LastFailure}
	}
	return s
}

func putStatus(ctx context.Context, tx *storage.Tx[file], id ID, s Status) error {
	p := db.PutStatusParams{
		IndexerID: id, State: string(s.State), Rung: int64(s.Rung), UntilAt: storage.FromTime(s.Until), SinceAt: storage.FromTime(s.Since),
		Cause: orNil(string(s.Cause)), Text: orNil(s.Text), NodeID: s.Node, SolverTried: s.SolverTried,
		SuccessAt: storage.FromTime(s.Success), Revision: s.Revision,
	}
	if s.Last != nil {
		p.LastFailure, p.LastFailureAt = &s.Last.Text, storage.FromTime(s.Last.At)
	}
	return db.New(tx).PutStatus(ctx, p)
}
