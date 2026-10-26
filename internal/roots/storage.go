package roots

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jatassi/slipstream/internal/nodes"
	storagev1 "github.com/jatassi/slipstream/internal/pb/slipstream/storage/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots/internal/db"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
)

// StorageIntent derives a Node's Storage intent from the library: its roots
// by id, each to adopt while roots_seen doesn't hold its place, with up to
// three of its recorded files as the proof adoption needs, and how often
// to sample.
func StorageIntent(c Contents) func(context.Context, storage.Reader[file], role.NodeID) (*storagev1.Intent, error) {
	return func(ctx context.Context, rd storage.Reader[file], node role.NodeID) (*storagev1.Intent, error) {
		q := db.New(rd)
		rows, err := q.ListIntentRoots(ctx, node)
		if err != nil {
			return nil, err
		}
		every, err := SampleEveryIn(ctx, rd)
		if err != nil {
			return nil, err
		}
		in := &storagev1.Intent{SampleEverySeconds: int64(every / time.Second)}
		for _, row := range rows {
			root := &storagev1.Root{Id: row.ID.Bytes(), Path: row.Path, Place: row.Place}
			if root.Adopt = !row.SeenPlace.Valid || row.SeenPlace.Int64 != row.Place; root.Adopt {
				if root.Proof, err = c.Proof(ctx, rd, row.ID); err != nil {
					return nil, err
				}
			}
			in.Roots = append(in.Roots, root)
		}
		return in, nil
	}
}

// ApplyStorage applies nothing: the Storage Role reports by facts alone.
func ApplyStorage(context.Context, *storage.Tx[file], role.NodeID, *storagev1.Result) error {
	return nil
}

// StorageOwner counts a Node's roots and the bytes of their files. Only a
// re-home or a delete takes a root off its Node, so the Node never
// finishes holding them on its own.
func StorageOwner(c Contents) nodes.Owner {
	return nodes.Owner{Role: role.Storage, Held: func(ctx context.Context, rd storage.Reader[file], node role.NodeID) (nodes.Held, error) {
		rows, err := db.New(rd).ListRootsOnNode(ctx, node)
		if err != nil {
			return nodes.Held{}, err
		}
		ids := make([]ID, len(rows))
		for i, row := range rows {
			ids[i] = row.ID
		}
		use, err := c.Use(ctx, rd, ids)
		if err != nil {
			return nodes.Held{}, err
		}
		var bytes int64
		for _, u := range use {
			bytes += u.Bytes
		}
		return nodes.Held{Units: int64(len(rows)), Noun: nodes.Noun{One: "root", Many: "roots"}, Bytes: bytes, Endless: true}, nil
	}}
}

// stateOf is a root's State: written off and offline come from the roster
// and win over what its Node reported; a root its Node hasn't reported at
// its place, reported zero, is Checking.
func stateOf(writtenOff, inStep bool, reported storing.RootState) State {
	switch {
	case writtenOff:
		return WrittenOff
	case !inStep:
		return Offline
	case reported == 0:
		return Checking
	}
	return State(reported.String())
}

type report struct {
	state  storing.RootState
	volume string
	// detail is the OS error text behind state.
	detail string
}

type placed struct {
	report
	place int64
}

// judged is a root's State and the OS error text its Node gave for it.
type judged struct {
	state  State
	detail string
}

// judge is each of roots' State, reading each Node's facts once.
func (r *Roots) judge(roots []Root, lost map[ID]bool) ([]judged, error) {
	return judgeRoots(roots, lost, r.cfg.Storage, r.cfg.Roster)
}

func judgeRoots(roots []Root, lost map[ID]bool, s Storage, roster Roster) ([]judged, error) {
	facts := map[role.NodeID]map[ID]placed{}
	out := make([]judged, len(roots))
	for i, root := range roots {
		of, ok := facts[root.Node]
		if !ok {
			var err error
			if of, err = reportedOn(s, root.Node); err != nil {
				return nil, err
			}
			facts[root.Node] = of
		}
		var rep report
		if p, ok := of[root.ID]; ok && p.place == root.Place {
			rep = p.report
		}
		out[i] = judged{stateOf(lost[root.ID], roster.InStep(root.Node), rep.state), rep.detail}
	}
	return out, nil
}

func reportedOn(s Storage, node role.NodeID) (map[ID]placed, error) {
	f, ok := s.Facts(node)
	if !ok {
		return nil, nil
	}
	return reportsOf(node, f)
}

func reportsOf(node role.NodeID, f *storagev1.Facts) (map[ID]placed, error) {
	out := make(map[ID]placed, len(f.GetRoots()))
	for _, rf := range f.GetRoots() {
		id, err := storage.UUIDFromBytes(rf.GetId())
		if err != nil {
			return nil, fmt.Errorf("node %s's Storage facts: root id: %w", node, err)
		}
		state, err := storing.ParseRootState(rf.GetState())
		if err != nil {
			return nil, fmt.Errorf("node %s's Storage facts: root %s: %w", node, id, err)
		}
		out[ID{UUID: id}] = placed{report{state, rf.GetVolumeId(), rf.GetDetail()}, rf.GetPlace()}
	}
	return out, nil
}

// Arrivals is the Nodes whose Storage facts arrived since the recorder last
// took them. Note never blocks, so it can be the roster's facts hook.
type Arrivals struct {
	mu    sync.Mutex
	nodes map[role.NodeID]bool
	wake  chan struct{}
}

func NewArrivals() *Arrivals {
	return &Arrivals{nodes: map[role.NodeID]bool{}, wake: make(chan struct{}, 1)}
}

func (a *Arrivals) Note(node role.NodeID, scope string) {
	if scope != string(role.Storage) {
		return
	}
	a.mu.Lock()
	a.nodes[node] = true
	a.mu.Unlock()
	select {
	case a.wake <- struct{}{}:
	default:
	}
}

func (a *Arrivals) take() []role.NodeID {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := make([]role.NodeID, 0, len(a.nodes))
	for node := range a.nodes {
		out = append(out, node)
	}
	clear(a.nodes)
	return out
}

// Run is the recorder until ctx ends. For each Node whose Storage facts
// arrived it records where the Node found each root ready, and the Node's
// volumes; Touches the Node once a root it was adopting is seen at its
// place; publishes volumes.changed for the volumes whose figures moved;
// and publishes roots.changed for each root whose report moved, closing
// Watch.
func (r *Roots) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-r.cfg.Arrivals.wake:
		}
		for _, node := range r.cfg.Arrivals.take() {
			if err := r.record(ctx, node); err != nil && ctx.Err() == nil {
				r.cfg.Log.Warn("record a Node's Storage facts", "node", node.String(), "err", err.Error())
			}
		}
	}
}

func (r *Roots) record(ctx context.Context, node role.NodeID) error {
	f, ok := r.cfg.Storage.Facts(node)
	if !ok {
		return nil
	}
	reports, err := reportsOf(node, f)
	if err != nil {
		return err
	}
	var (
		on      []db.RootsRoot
		adopted bool
		volumes []string
	)
	err = r.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		q := db.New(tx)
		var err error
		if on, err = q.ListRootsOnNode(ctx, node); err != nil {
			return err
		}
		if adopted, err = see(ctx, q, on, reports); err != nil {
			return err
		}
		volumes, err = replaceVolumes(ctx, q, node, f.GetVolumes())
		return err
	})
	if err != nil {
		return err
	}
	r.recorded(node, time.UnixMilli(f.GetPassedAtUnixMs()))
	if adopted {
		r.cfg.Storage.Touch(node)
	}
	if len(volumes) > 0 {
		r.cfg.Volumes.Publish(ctx, VolumesChanged{NodeID: node.String(), VolumeIDs: volumes})
	}
	var moved []ID
	for _, row := range on {
		p, ok := reports[row.ID]
		if !ok || p.place != row.Place || r.last[row.ID] == p {
			continue
		}
		r.last[row.ID] = p
		moved = append(moved, row.ID)
	}
	if len(moved) > 0 {
		r.moved()
		r.cfg.Changed.Publish(ctx, changedOf(moved))
	}
	return nil
}

// see records each root of on its Node reports ready at its place,
// adopting one seen there for the first time.
func see(ctx context.Context, q *db.Queries, on []db.RootsRoot, reports map[ID]placed) (adopted bool, err error) {
	rows, err := q.ListSeen(ctx)
	if err != nil {
		return false, err
	}
	seen := make(map[ID]db.RootsSeen, len(rows))
	for _, row := range rows {
		seen[row.RootID] = row
	}
	for _, root := range on {
		p, ok := reports[root.ID]
		if !ok || p.place != root.Place || p.state != storing.Ready {
			continue
		}
		was, ok := seen[root.ID]
		fresh := !ok || was.Place != root.Place
		if !fresh && was.VolumeID == db.VolumeID(p.volume) {
			continue
		}
		if err := q.SeeRoot(ctx, db.SeeRootParams{RootID: root.ID, Place: root.Place, VolumeID: db.VolumeID(p.volume)}); err != nil {
			return false, err
		}
		adopted = adopted || fresh
	}
	return adopted, nil
}

func (r *Roots) recorded(node role.NodeID, passed time.Time) {
	r.passed.Lock()
	defer r.passed.Unlock()
	if !passed.After(r.passed.at[node]) {
		return
	}
	r.passed.at[node] = passed
	close(r.passed.ch)
	r.passed.ch = make(chan struct{})
}

// Sample has node's Storage judge every root and sample its volumes now,
// and returns once the recorder has written that pass's facts or a later
// one's. A Node out of step is ErrNodeAway.
func (r *Roots) Sample(ctx context.Context, node role.NodeID) error {
	if !r.cfg.Roster.InStep(node) {
		return ErrNodeAway
	}
	at, err := r.cfg.Remote.Sample(ctx, node)
	if err := reached(err); err != nil {
		return err
	}
	for {
		r.passed.Lock()
		passed, ch := r.passed.at[node], r.passed.ch
		r.passed.Unlock()
		if !passed.Before(at) {
			return nil
		}
		select {
		case <-ch:
		case <-ctx.Done():
			return fmt.Errorf("await node %s's sample: %w", node, ctx.Err())
		}
	}
}
