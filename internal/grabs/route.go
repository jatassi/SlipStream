package grabs

import (
	"context"
	"errors"
	"sync"

	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/nodes"
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

// routeIn is the client g goes to now, read in r. It fails with
// *nodes.Unrouted while no Node may take a Grab that goes by its Work's
// root.
func (gr *Grabs) routeIn(ctx context.Context, r storage.Reader[file], g Grab) (downloadclients.Target, error) {
	root, err := roots.OfWork(ctx, r, g.WorkID)
	if err != nil {
		return downloadclients.Target{}, err
	}
	return gr.cfg.Router.Route(ctx, r, downloadclients.Ask{Root: root.Node, Indexer: g.Ticket.Indexer, Tracker: g.Ticket.Tracker, Sent: g.Sent})
}

// route is the client g goes to now, or the wait until it may go there:
// while no Node may take it, or while its client's Node is out of step.
func (gr *Grabs) route(ctx context.Context, g Grab) (downloadclients.Target, outcome) {
	changed := gr.routes.watch()
	var to downloadclients.Target
	err := gr.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		to, err = gr.routeIn(ctx, r, g)
		return err
	})
	if _, unrouted := errors.AsType[*nodes.Unrouted](err); unrouted {
		return downloadclients.Target{}, wait{cause: err, roster: true}
	}
	if err != nil {
		return downloadclients.Target{}, local(ctx, err)
	}
	if !gr.cfg.Roster.InStep(to.Node) {
		return downloadclients.Target{}, wait{cause: errClientAway, roster: true, rerouted: changed}
	}
	return to, nil
}

// waitsForClient reports whether fetching Grab g waits for its client's
// Node, read in r: its route names a client whose Node is out of step.
func (gr *Grabs) waitsForClient(ctx context.Context, r storage.Reader[file], g Grab) (bool, error) {
	to, err := gr.routeIn(ctx, r, g)
	if _, unrouted := errors.AsType[*nodes.Unrouted](err); unrouted {
		return false, nil
	}
	return err == nil && !gr.cfg.Roster.InStep(to.Node), err
}
