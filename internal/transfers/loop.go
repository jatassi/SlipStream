package transfers

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"sync"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/nodes"
	tv1 "github.com/jatassi/slipstream/internal/pb/slipstream/transfer/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/transfers/internal/ledger"
)

// loop is what Run keeps between passes: what it last judged, wrote and
// published.
type loop struct {
	// intents is nil until the first pass, which Touches every Node: a
	// Node's session may have derived its intent before the loop judged.
	intents intents
	views   map[ID]View
	written map[pairKey]written

	transfers map[string]bool
	nodes     map[string]bool
	published time.Time
}

type written struct {
	at    time.Time
	route tv1.Route
}

// Run judges again after every wake. It stores the failures judge
// decided, Touches the Nodes whose intent moved, asks for volume samples,
// records each pair's speed and route, and publishes transfers.changed.
// It is the only code that Touches the transfers scope.
func (t *Transfers) Run(ctx context.Context) error {
	var sampling sync.WaitGroup
	defer sampling.Wait()
	l := &loop{views: map[ID]View{}, written: map[pairKey]written{}, transfers: map[string]bool{}, nodes: map[string]bool{}}
	watched := t.cfg.Hub.Watch(ctx, roots.ChangedEvent, roots.VolumesChangedEvent, health.ChangedEvent, nodes.ChangedEvent)
	for {
		board := t.cfg.Board()
		rootsMoved, moved := t.cfg.RootsMoved(), t.cfg.Moved()
		due, err := t.pass(ctx, l, board, &sampling)
		if err != nil && ctx.Err() == nil {
			t.cfg.Log.Error("judge the Transfers", "err", err.Error())
		}
		var timer <-chan time.Time
		if !due.IsZero() {
			timer = time.After(time.Until(due))
		}
		select {
		case <-ctx.Done():
			return nil
		case <-t.poke:
		case <-watched:
		case <-board.Changed():
		case <-rootsMoved:
		case <-moved:
		case <-timer:
		}
	}
}

// pass judges once and acts on the judgment. It returns when the loop
// must judge again with no wake, zero for never.
func (t *Transfers) pass(ctx context.Context, l *loop, board Board, sampling *sync.WaitGroup) (time.Time, error) {
	j, err := t.judgeNow(ctx, board)
	if err != nil {
		return time.Time{}, err
	}
	if len(j.fail) > 0 {
		if err := t.store(ctx, j.fail); err != nil {
			return time.Time{}, err
		}
		t.Touch()
		return time.Time{}, nil
	}
	now := time.Now()
	t.touchMoved(l, j)
	views := make(map[ID]View, len(j.views))
	for _, v := range j.views {
		views[v.ID] = v
	}
	for _, node := range sampled(l.views, views) {
		sampling.Go(func() {
			if err := t.cfg.Sample(ctx, node); err != nil {
				t.cfg.Log.Debug("sample the destination's volumes", "node", node.String(), "err", err.Error())
			}
		})
	}
	if err := t.writePairs(ctx, l, j.views, now); err != nil {
		return time.Time{}, err
	}
	t.moved(l, views)
	due := t.publish(ctx, l, now)
	if !j.next.IsZero() && (due.IsZero() || j.next.Before(due)) {
		due = j.next
	}
	return due, nil
}

// moved notes for transfers.changed each Transfer whose view moved since
// the last pass and each Node whose send limit changed, and forgets the
// full disks of Transfers that went.
func (t *Transfers) moved(l *loop, views map[ID]View) {
	for id, v := range views {
		if old, ok := l.views[id]; !ok || !reflect.DeepEqual(old, v) {
			l.transfers[id.String()] = true
		}
	}
	for id := range l.views {
		if _, ok := views[id]; !ok {
			l.transfers[id.String()] = true
		}
	}
	l.views = views
	t.mu.Lock()
	defer t.mu.Unlock()
	for id := range t.fullAt {
		if _, ok := views[id]; !ok {
			delete(t.fullAt, id)
		}
	}
	for _, node := range t.limited {
		l.nodes[node.String()] = true
	}
	t.limited = nil
}

// touchMoved Touches every Node whose intent differs from the last pass's.
func (t *Transfers) touchMoved(l *loop, j judgment) {
	var moved []role.NodeID
	for node := range j.intents {
		if l.intents == nil || !proto.Equal(j.intents.of(node), l.intents.of(node)) {
			moved = append(moved, node)
		}
	}
	for node := range l.intents {
		if _, ok := j.intents[node]; !ok && !proto.Equal(j.intents.of(node), l.intents.of(node)) {
			moved = append(moved, node)
		}
	}
	l.intents = j.intents
	if len(moved) > 0 {
		slices.SortFunc(moved, func(a, b role.NodeID) int { return a.Compare(b.UUID) })
		t.cfg.Touch(moved...)
	}
}

// sampled is the destination Nodes whose volumes need a fresh sample: a
// Transfer that first needs space, or one that placed a file or went, may
// change what fits there.
func sampled(last, next map[ID]View) []role.NodeID {
	var out []role.NodeID
	add := func(n role.NodeID) {
		if !slices.Contains(out, n) {
			out = append(out, n)
		}
	}
	for id, v := range next {
		old, ok := last[id]
		if v.State == NeedsSpace && (!ok || old.State != NeedsSpace) {
			add(v.Space.Node)
		}
		if ok && placed(v.Transfer) > placed(old.Transfer) {
			add(v.DestNode)
		}
	}
	for id, old := range last {
		if _, ok := next[id]; !ok {
			add(old.DestNode)
		}
	}
	return out
}

func placed(t Transfer) int {
	n := 0
	for _, f := range t.Files {
		if f.Placed {
			n++
		}
	}
	return n
}

// store records the failures judge decided, each in the Write that fires
// its Failed hook, then runs After.
func (t *Transfers) store(ctx context.Context, fail []judged) error {
	var fired map[PurposeKind][]string
	err := t.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		fired = map[PurposeKind][]string{}
		for _, f := range fail {
			tr, err := load(ctx, tx, f.id)
			if errors.Is(err, storage.ErrNoRows) {
				continue
			}
			if err != nil {
				return err
			}
			if tr.Failure != nil {
				continue
			}
			root, err := roots.Get(ctx, tx, rootOf(tr.Destination))
			if err != nil {
				return err
			}
			if err := t.fail(ctx, tx, tr, root.Node, f.code, f.position, ""); err != nil {
				return err
			}
			fired[tr.Purpose] = append(fired[tr.Purpose], tr.Job)
		}
		return nil
	})
	if err != nil {
		return err
	}
	t.after(fired)
	return nil
}

// writePairs records the speed and route of each pair bytes move on: at
// once when they first move on a route, so a short Transfer seeds the
// next, then at most once each PairWrite.
func (t *Transfers) writePairs(ctx context.Context, l *loop, views []View, now time.Time) error {
	var rows []ledger.PutPairParams
	for _, v := range views {
		if v.State != Copying || v.Route == tv1.Route_ROUTE_UNSPECIFIED {
			continue
		}
		key := pairKey{v.Source, v.DestNode}
		if w, ok := l.written[key]; ok && w.route == v.Route && now.Sub(w.at) < t.cfg.Timings.PairWrite {
			continue
		}
		l.written[key] = written{at: now, route: v.Route}
		at := storage.FromTime(now)
		rows = append(rows, ledger.PutPairParams{SourceNode: v.Source, DestinationNode: v.DestNode, Rate: v.Rate, RateAt: at, Route: new(int64(v.Route)), RouteAt: at})
	}
	if len(rows) == 0 {
		return nil
	}
	err := t.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		q := ledger.New(tx)
		for _, row := range rows {
			if err := q.PutPair(ctx, row); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	t.Touch()
	return nil
}

// publish sends transfers.changed with everything that moved since the
// last one, at most once each Changed. It returns when the next is due,
// zero when nothing waits.
func (t *Transfers) publish(ctx context.Context, l *loop, now time.Time) time.Time {
	if len(l.transfers) == 0 && len(l.nodes) == 0 {
		return time.Time{}
	}
	if due := l.published.Add(t.cfg.Timings.Changed); now.Before(due) {
		return due
	}
	ev := Changed{TransferIDs: sortedKeys(l.transfers), NodeIDs: sortedKeys(l.nodes)}
	clear(l.transfers)
	clear(l.nodes)
	l.published = now
	t.cfg.Changed.Publish(ctx, ev)
	return time.Time{}
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}
