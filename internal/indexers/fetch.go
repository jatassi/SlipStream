package indexers

import (
	"context"
	"errors"
	"time"

	"github.com/jatassi/slipstream/internal/indexers/internal/db"
	"github.com/jatassi/slipstream/internal/magnet"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/searching"
	"github.com/jatassi/slipstream/internal/storage"
)

// Ticket is a Release made ready to fetch: its Indexer and the grab Take
// reserved, zero for a magnet link, which reaches no site.
type Ticket struct {
	Indexer ID
	Query   QueryID
}

// NotTakenError is a Release whose fetch would wait for the Admin: its
// Indexer stopped, at its grab limit, or deleted.
type NotTakenError struct{ Outcome Outcome }

func (e *NotTakenError) Error() string {
	return "the Indexer can't fetch the Release now: " + string(e.Outcome.Kind)
}

// Take readies r inside the decision's Write: *NotTakenError while r's
// Indexer is stopped, at its grab limit or deleted, else its Ticket with a
// grab reserved, none for a magnet link.
func (x *Indexers) Take(ctx context.Context, tx *storage.Tx[file], r Release) (Ticket, error) {
	idx, err := Get(ctx, tx, r.Indexer)
	if errors.Is(err, ErrNoIndexer) {
		return Ticket{}, &NotTakenError{Outcome: Outcome{Kind: Removed}}
	}
	if err != nil {
		return Ticket{}, err
	}
	st, err := status(ctx, tx, idx.ID)
	if err != nil {
		return Ticket{}, err
	}
	if st.State == Stopped {
		return Ticket{}, &NotTakenError{Outcome: Outcome{Kind: IsStopped, Cause: st.Cause, Text: st.Text, Node: st.Node}}
	}
	t := Ticket{Indexer: idx.ID}
	if r.Offered().IsMagnet() {
		return t, nil
	}
	t.Query, err = reserve(ctx, tx, idx, UseFetch, storage.Now())
	if limit, ok := errors.AsType[*LimitError](err); ok {
		return Ticket{}, &NotTakenError{Outcome: Outcome{Kind: LimitReached, Until: limit.Until}}
	}
	return t, err
}

// Got is what a Release's link leads to: a Torrent or a Magnet.
type Got interface{ got() }

// Torrent is a .torrent's bytes as the Indexer sent them, and the
// Searching Node that fetched them.
type Torrent struct {
	Raw  []byte
	Node role.NodeID
}

// Magnet is a magnet link: the Release's own, or the one its link
// redirected to.
type Magnet struct{ Link magnet.Link }

func (Torrent) got() {}
func (Magnet) got()  {}

// DropError is a fetch that would wait for the Admin, so its Grab drops
// and the decision reruns.
type DropError struct{ Why DropWhy }

type DropWhy string

const (
	DropStopped     DropWhy = "indexer_stopped"
	DropHomeOffline DropWhy = "home_offline"
	DropRemoved     DropWhy = "indexer_removed"
)

func (e *DropError) Error() string { return "the Indexer can't fetch the Release: " + string(e.Why) }

// Fetch is what link leads to. A magnet link needs no Node. Any other link
// is one job on the Node t's Indexer routes to now, which may fetch it
// from any host and follow it to a magnet link. The first attempt's job
// is t's reserved query, so a call that lost its answer joins it. It fails
// with *WaitError while the Indexer is limited, failing or its home is
// away, *DropError once it stopped, a home it signs in on lapsed or it was
// deleted, *UnavailableError, *magnet.InvalidError or bencode.ErrV2Only.
func (x *Indexers) Fetch(ctx context.Context, t Ticket, link searching.Link) (Got, error) {
	if link.IsMagnet() {
		return parseMagnet(link)
	}
	var idx Indexer
	var st Status
	var answered bool
	err := x.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		if idx, err = Get(ctx, r, t.Indexer); err != nil {
			return err
		}
		if st, err = status(ctx, r, idx.ID); err != nil {
			return err
		}
		answered, err = db.New(r).QueryCompleted(ctx, t.Query)
		return err
	})
	if errors.Is(err, ErrNoIndexer) {
		return nil, &DropError{Why: DropRemoved}
	}
	if err != nil {
		return nil, err
	}
	if wait := x.fetchGate(idx, st, time.Now()); wait != nil {
		return nil, wait
	}
	j, err := x.routed(ctx, t.Query, idx, UseFetch, searching.FetchCall{Link: link})
	if err != nil {
		return nil, err
	}
	if answered {
		j.wire.ID = storage.NewUUID().String()
	}
	a, st, err := x.run(ctx, j)
	if errors.Is(err, ErrLate) {
		return nil, &WaitError{Outcome: Outcome{Kind: Late}}
	}
	if err != nil {
		return nil, err
	}
	if errors.As(a.Outcome, new(*searching.Stopped)) {
		return nil, &DropError{Why: DropStopped}
	}
	if a.Outcome != nil {
		return nil, &WaitError{Outcome: outcomeOf(a, st)}
	}
	switch got := a.Got.(type) {
	case searching.Torrent:
		return Torrent{Raw: got.Raw, Node: j.node}, nil
	case searching.Magnet:
		return parseMagnet(got.Link)
	}
	return nil, &UnavailableError{Err: errors.New("the Searching Node's response holds neither a .torrent nor a magnet link")}
}

// fetchGate is why idx can't fetch now: a *DropError once it stopped or a
// home it signs in on lapsed, a *WaitError while it is limited, failing or
// waits for its home, nil when it may. A grab is reserved at Take, so no
// limit holds a fetch.
func (x *Indexers) fetchGate(idx Indexer, st Status, now time.Time) error {
	_, wait := gate(idx, st, Usage{}, x.cfg.Roster, UseFetch, now)
	switch {
	case wait == nil:
		return nil
	case wait.Kind == IsStopped:
		return &DropError{Why: DropStopped}
	case wait.Kind == Waiting && wait.Lapsed && !portable(idx, st):
		return &DropError{Why: DropHomeOffline}
	}
	return &WaitError{Outcome: *wait}
}

func parseMagnet(link searching.Link) (Got, error) {
	l, err := link.Magnet()
	if err != nil {
		return nil, err
	}
	return Magnet{Link: l}, nil
}

// FetchWait is why fetches from an Indexer wait now.
type FetchWait struct {
	Outcome Outcome
	// Sentence is the wait in words, for the Grab's stage.
	Sentence string
}

// FetchWaits is why each of ids' fetches waits now, read in r, absent for
// one whose fetches don't wait or that no longer exists.
func (x *Indexers) FetchWaits(ctx context.Context, r storage.Reader[file], ids []ID) (map[ID]FetchWait, error) {
	statuses, err := statusesIn(ctx, r)
	if err != nil {
		return nil, err
	}
	out := map[ID]FetchWait{}
	now := time.Now()
	for _, id := range ids {
		idx, err := Get(ctx, r, id)
		if errors.Is(err, ErrNoIndexer) {
			continue
		}
		if err != nil {
			return nil, err
		}
		st, ok := statuses[id]
		if !ok {
			st = Status{State: OK}
		}
		if wait, ok := errors.AsType[*WaitError](x.fetchGate(idx, st, now)); ok {
			out[id] = FetchWait{Outcome: wait.Outcome, Sentence: x.fetchWaitWords(idx.Name, wait.Outcome)}
		}
	}
	return out, nil
}
