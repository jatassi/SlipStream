package grabs

import (
	"context"
	"errors"
	"sync"

	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
)

// routes is closed and made anew whenever a Grab's route may have changed
// with no change to the roster: a send-to of a Grab that waits for its
// client, or a change to a Tracker's routes.
type routes struct {
	mu      sync.Mutex
	changed chan struct{}
}

func (r *routes) watch() <-chan struct{} {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.changed
}

func (r *routes) change() {
	r.mu.Lock()
	defer r.mu.Unlock()
	close(r.changed)
	r.changed = make(chan struct{})
}

// Rerouted tells the Grabs that wait for their client's Node that a
// Tracker's routes changed, so each routes again.
func (gr *Grabs) Rerouted() { gr.routes.change() }

// errClientAway is a Grab whose client's Node is out of step. It waits
// for that Node, or for its route to change.
var errClientAway = errors.New("the Node of the Grab's client is out of step")

// routeOf is the client router sends g to now, read in r. It fails with
// *nodes.Unrouted while no Node may take a Grab that goes by its Work's
// root.
func routeOf(ctx context.Context, r storage.Reader[file], router Router, g Grab) (downloadclients.Target, error) {
	root, err := roots.OfWork(ctx, r, g.WorkID)
	if err != nil {
		return downloadclients.Target{}, err
	}
	return router.Route(ctx, r, downloadclients.Ask{Root: root.Node, Indexer: g.Ticket.Indexer, Tracker: g.Ticket.Tracker, Sent: g.Sent, Pinned: g.Source == FromManual})
}

// route is the client g goes to now, or the wait until it may go there:
// while no Node may take it, while it is pinned to a client that can't
// take it, or while its client's Node is out of step.
func (gr *Grabs) route(ctx context.Context, g Grab) (downloadclients.Target, outcome) {
	changed := gr.routes.watch()
	var to downloadclients.Target
	err := gr.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		to, err = routeOf(ctx, r, gr.cfg.Router, g)
		return err
	})
	if _, unrouted := errors.AsType[*nodes.Unrouted](err); unrouted {
		return downloadclients.Target{}, wait{cause: err, roster: true}
	}
	if _, off := errors.AsType[*downloadclients.PinnedError](err); off {
		return downloadclients.Target{}, wait{cause: err, roster: true, rerouted: changed}
	}
	if err != nil {
		return downloadclients.Target{}, local(ctx, err)
	}
	if !gr.cfg.Roster.InStep(to.Node) {
		return downloadclients.Target{}, wait{cause: errClientAway, roster: true, rerouted: changed}
	}
	return to, nil
}

// ClientWait is the client a fetching Grab goes to while it can't go
// there: Node's built-in client when Client is nil.
type ClientWait struct {
	Node   role.NodeID
	Client *downloadclients.ClientRef
	// Off is a client the Grab is pinned to that can't take it: it is off
	// or deleted, or its Node doesn't hold Downloading. Otherwise the
	// client's Node is out of step.
	Off bool
}

// clientWait is the client fetching Grab g waits for, read in r: the
// client it is pinned to while that client can't take it, or the client
// its route names while that client's Node is out of step. It is nil while
// g's fetch may go ahead or no Node may take it.
func clientWait(ctx context.Context, r storage.Reader[file], router Router, inStep func(role.NodeID) bool, g Grab) (*ClientWait, error) {
	to, err := routeOf(ctx, r, router, g)
	if p, off := errors.AsType[*downloadclients.PinnedError](err); off {
		return &ClientWait{Node: p.Node, Client: p.Client, Off: true}, nil
	}
	if _, unrouted := errors.AsType[*nodes.Unrouted](err); unrouted {
		return nil, nil
	}
	if err != nil || inStep(to.Node) {
		return nil, err
	}
	w := &ClientWait{Node: to.Node}
	if to.Client != nil {
		e, err := downloadclients.ExternalOf(ctx, r, *to.Client)
		if err != nil {
			return nil, err
		}
		w.Client = &downloadclients.ClientRef{ID: e.ID, Kind: e.Kind, Name: e.Name}
	}
	return w, nil
}

// clientWaits is the client each fetching Grab of grabs waits for, read
// in r, absent for one that waits for none.
func (gr *Grabs) clientWaits(ctx context.Context, r storage.Reader[file], grabs []Grab) (map[GrabID]ClientWait, error) {
	out := map[GrabID]ClientWait{}
	for _, g := range grabs {
		if g.Stage != Fetching {
			continue
		}
		w, err := clientWait(ctx, r, gr.cfg.Router, gr.cfg.Roster.InStep, g)
		if err != nil {
			return nil, err
		}
		if w != nil {
			out[g.ID] = *w
		}
	}
	return out, nil
}
