package reachability

import (
	"context"
	"errors"
	"slices"
	"time"

	"google.golang.org/protobuf/types/known/emptypb"

	netv1 "github.com/jatassi/slipstream/internal/pb/slipstream/network/v1"
	"github.com/jatassi/slipstream/internal/reachability/internal/db"
	"github.com/jatassi/slipstream/internal/reachability/judge"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/tasks"
)

// Intent is one Node's network intent: whether it watches, how often, how
// long a change must hold, the STUN servers it asks (absent: its own), and
// with "Open router ports automatically" on, the forwards it maps. A Node
// watches while it is the origin of a wanted Exposure or may test one, so
// with nothing wanted no Node sends anything.
func (r *Reachability) Intent(ctx context.Context, rd storage.Reader[storage.Library], node role.NodeID) (*netv1.Intent, error) {
	wanted, err := wantedIn(ctx, rd)
	if err != nil || len(wanted) == 0 {
		return &netv1.Intent{}, err
	}
	o, err := originsIn(ctx, rd, r.cfg.Roster.Self())
	if err != nil {
		return nil, err
	}
	if !r.watches(o, wanted, node) {
		return &netv1.Intent{}, nil
	}
	s, _, err := settingsIn(ctx, rd)
	if err != nil {
		return nil, err
	}
	in := &netv1.Intent{Watch: true, IntervalMs: s.watchEvery.Milliseconds(), WaitMs: s.changeWait.Milliseconds()}
	if s.stun != nil {
		in.Stun = &netv1.StunList{Servers: *s.stun}
	}
	in.Map, err = mapsIn(ctx, rd, node)
	return in, err
}

// mapsIn is the forwards node maps: with its "Open router ports
// automatically" on, each wanted Exposure's whose last check found a Direct
// route from node that mapping can serve, once per external port.
func mapsIn(ctx context.Context, rd storage.Reader[storage.Library], node role.NodeID) ([]*netv1.MapWant, error) {
	q := db.New(rd)
	switch set, err := q.GetNodeSetting(ctx, node); {
	case errors.Is(err, storage.ErrNoRows):
		return nil, nil
	case err != nil:
		return nil, err
	case set.MapPorts == nil || *set.MapPorts == 0:
		return nil, nil
	}
	rows, err := q.ListResults(ctx)
	if err != nil {
		return nil, err
	}
	var out []*netv1.MapWant
	for _, row := range rows {
		res, err := resultOf(row)
		if err != nil {
			return nil, err
		}
		fw := res.judged.Forward
		if !res.read || res.judged.Origin != node || fw == nil ||
			slices.ContainsFunc(out, func(m *netv1.MapWant) bool { return m.GetExternal() == uint32(fw.External) }) {
			continue
		}
		out = append(out, &netv1.MapWant{External: uint32(fw.External), Internal: uint32(fw.Internal)})
	}
	return out, nil
}

// watches is whether node is the origin of an Exposure or a candidate to
// test one: every Node but a Worker that serves it.
func (r *Reachability) watches(o origins, wanted []Exposure, node role.NodeID) bool {
	serving := r.cfg.Roster.Serving(role.Serving)
	for _, e := range wanted {
		origin, _ := o.of(e)
		if origin == node || node == o.self {
			return true
		}
		serves := contains(serving, node)
		if e.Kind == judge.MediaServer {
			rec := o.servers[e.Server]
			serves = node == rec.Reaching || node == rec.RunsOn
		}
		if !serves {
			return true
		}
	}
	return false
}

// Apply refuses every result: the network channel has none.
func (r *Reachability) Apply(context.Context, *storage.Tx[storage.Library], role.NodeID, *emptypb.Empty) error {
	return errors.New("the network channel has no results")
}

// Noted reruns the checks a Node's confirmed network change may move: the
// Exposures it is the origin of. At most one rerun per watch interval per
// Node; a change inside it waits for the interval's end and marks the Node
// churning, which the next check shows.
func (r *Reachability) Noted(node role.NodeID, scope string) {
	if scope != string(role.Network) {
		return
	}
	ctx := context.Background()
	var every time.Duration
	err := r.cfg.Library.Read(ctx, func(ctx context.Context, rd storage.Reader[storage.Library]) error {
		s, _, err := settingsIn(ctx, rd)
		every = s.watchEvery
		return err
	})
	if err != nil {
		r.cfg.Log.WarnContext(ctx, "read the watch interval", "err", err)
		return
	}
	now := r.now()
	r.mu.Lock()
	last, ran := r.reruns[node]
	if ran && now.Sub(last) < every {
		r.churn[node] = true
		if !r.armed[node] {
			r.armed[node] = true
			time.AfterFunc(last.Add(every).Sub(now), func() { r.rerun(node, true) })
		}
		r.mu.Unlock()
		return
	}
	r.mu.Unlock()
	r.rerun(node, false)
}

func (r *Reachability) rerun(node role.NodeID, churned bool) {
	ctx := context.Background()
	now := r.now()
	r.mu.Lock()
	r.reruns[node], r.freshAfter[node] = now, now
	r.armed[node] = false
	if !churned {
		delete(r.churn, node)
	}
	r.mu.Unlock()
	units, err := r.unitsOf(ctx, node)
	if err != nil {
		r.cfg.Log.WarnContext(ctx, "list the routes a network change moved", "err", err)
		return
	}
	r.start(ctx, units...)
}

// unitsOf is the units of the Exposures whose origin or last tester is
// node.
func (r *Reachability) unitsOf(ctx context.Context, node role.NodeID) ([]tasks.UnitKey, error) {
	var out []tasks.UnitKey
	err := r.cfg.Library.Read(ctx, func(ctx context.Context, rd storage.Reader[storage.Library]) error {
		wanted, err := wantedIn(ctx, rd)
		if err != nil {
			return err
		}
		o, err := originsIn(ctx, rd, r.cfg.Roster.Self())
		if err != nil {
			return err
		}
		testers, err := testersIn(ctx, rd)
		if err != nil {
			return err
		}
		for _, e := range wanted {
			if origin, _ := o.of(e); origin == node || testers[e.ID] == node {
				out = append(out, e.Unit())
			}
		}
		return nil
	})
	return out, err
}
