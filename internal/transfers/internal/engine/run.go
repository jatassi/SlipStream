package engine

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/jatassi/slipstream/internal/link"
	tv1 "github.com/jatassi/slipstream/internal/pb/slipstream/transfer/v1"
)

// run is one active Transfer on one side of this Node, from the intent
// that made it active until one that drops it or changes its shape.
type run struct {
	e      *Engine
	h      Holder
	spec   spec
	ctx    context.Context //nolint:containedctx // a run's lifetime, which every lane it holds shares
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// rx is the receiver's state, nil on the sender.
	rx *receiving

	mu    sync.Mutex
	r     tv1.Route
	moved chan struct{} // closes when the route changes
	lanes map[*lane]bool
	// failing is when the current run of failed attempts on each route
	// this end dials began; tries counts them, for the backoff.
	failing map[tv1.Route]time.Time
	tries   map[tv1.Route]int
	gaveUp  map[tv1.Route]bool
	// binding: the send limit held a chunk back since the last facts.
	binding bool
}

func newRun(e *Engine, h Holder, sp spec) *run {
	return &run{
		e: e, h: h, spec: sp, r: sp.route, moved: make(chan struct{}), lanes: map[*lane]bool{},
		failing: map[tv1.Route]time.Time{}, tries: map[tv1.Route]int{}, gaveUp: map[tv1.Route]bool{},
	}
}

func (r *run) route() tv1.Route {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.r
}

func (r *run) routeChanged() <-chan struct{} {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.moved
}

// setRoute hands the run a new route and resets each lane the route no
// longer holds.
func (r *run) setRoute(route tv1.Route) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.r = route
	close(r.moved)
	r.moved = make(chan struct{})
	for l := range r.lanes {
		if !holds(route, l.route) {
			_ = l.c.Close()
		}
	}
	if r.rx != nil && !holds(route, r.rx.moving) {
		r.rx.moving = tv1.Route_ROUTE_UNSPECIFIED
	}
}

// hold registers l, or closes it and reports false when the route no
// longer holds it.
func (r *run) hold(l *lane) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !holds(r.r, l.route) || r.ctx.Err() != nil {
		_ = l.c.Close()
		return false
	}
	r.lanes[l] = true
	l.unwatch = context.AfterFunc(r.ctx, func() { _ = l.c.Close() })
	return true
}

func (r *run) drop(l *lane) {
	r.mu.Lock()
	delete(r.lanes, l)
	r.mu.Unlock()
	l.unwatch()
	_ = l.c.Close()
	close(l.done)
}

// open opens a lane on route, the end's own direction, and sends its Lane
// frame.
func (r *run) open(route tv1.Route) (*lane, error) {
	var l *lane
	if route == tv1.Route_ROUTE_NODE_LINK {
		s, err := r.e.cfg.Link.OpenStream(r.ctx, r.spec.peer, link.KindTransfer)
		if err != nil {
			return nil, fmt.Errorf("open a stream: %w", err)
		}
		l = newLane(s, route, r.e.cfg.Timings.Chunk)
	} else {
		c, err := r.e.cfg.Link.DialLane(r.ctx, r.spec.peer, r.spec.addrs)
		if err != nil {
			return nil, fmt.Errorf("dial a lane: %w", err)
		}
		l = newLane(c, route, r.e.cfg.Timings.Chunk)
	}
	l.dialed = true
	if !r.hold(l) {
		return nil, fmt.Errorf("the lane's route %s isn't the Transfer's route now", route)
	}
	if err := l.send(laneFrame(r.spec.id, route)); err != nil {
		r.drop(l)
		return nil, err
	}
	return l, nil
}

// take serves a lane the peer opened, until the run lets it go.
func (r *run) take(ctx context.Context, l *lane) {
	if !r.hold(l) {
		return
	}
	if r.rx == nil {
		r.serveSends(l)
		r.drop(l)
		return
	}
	t := time.NewTimer(r.e.cfg.Timings.Chunk)
	defer t.Stop()
	select {
	case r.rx.inbound <- l:
		<-l.done
	case <-t.C:
		r.drop(l)
	case <-ctx.Done():
		r.drop(l)
	case <-r.ctx.Done():
		r.drop(l)
	}
}

// heard notes that a lane this end opened on route reached the peer.
func (r *run) heard(route tv1.Route) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.failing, route)
	delete(r.tries, route)
}

// failed notes a failed attempt on route and returns the backoff before
// the next. Failing for Dial gives the route up for this run, and the
// facts say so at once.
func (r *run) failed(route tv1.Route) time.Duration {
	now := time.Now()
	r.mu.Lock()
	first, ok := r.failing[route]
	if !ok {
		first = now
		r.failing[route] = now
	}
	r.tries[route]++
	tries := r.tries[route]
	gaveUp := !r.gaveUp[route] && now.Sub(first) >= r.e.cfg.Timings.Dial
	if gaveUp {
		r.gaveUp[route] = true
	}
	r.mu.Unlock()
	if gaveUp {
		r.e.facts.urgent()
	}
	return retryDelay(tries, r.e.cfg.Timings)
}

// pause waits d, and reports false when the run ended first. A route
// change ends it early.
func (r *run) pause(d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
	case <-r.routeChanged():
	case <-r.ctx.Done():
		return false
	}
	return true
}

func (r *run) gaveUpRoutes() []tv1.Route {
	var out []tv1.Route
	for route := range r.gaveUp {
		out = append(out, route)
	}
	slices.Sort(out)
	return out
}

// retryDelay is the wait before attempt tries+1: RetryMin, doubling to
// RetryMax.
func retryDelay(tries int, t Timings) time.Duration {
	d := t.RetryMin
	for i := 1; i < tries && d < t.RetryMax; i++ {
		d *= 2
	}
	return min(d, t.RetryMax)
}
