package engine

import (
	"slices"
	"sync"
	"time"

	tv1 "github.com/jatassi/slipstream/internal/pb/slipstream/transfer/v1"
)

// laneEnd is one side's part in a run's lanes.
type laneEnd struct {
	// inbound is the lanes the peer opens that this end carries, nil when
	// they are served where they arrive.
	inbound <-chan *lane
	// ready gates a first lane: ok, or the wait before asking again, 0 for
	// not until the route changes.
	ready func() (retry time.Duration, ok bool)
	// serve carries l until it ends, and reports whether the peer answered
	// on it.
	serve func(l *lane) (heard bool)
	// grows: this end judges growth from the bytes its lanes carry and
	// retires the lanes it no longer wants. The other end keeps as many
	// lanes as its peer last wanted.
	grows bool
}

// laneEvent is a dial this end finished, or a lane it served letting go.
type laneEvent struct {
	l     *lane
	route tv1.Route
	// opened is a dial's end: l is nil when err says why it failed.
	opened bool
	err    error
	own    bool
	heard  bool
}

// pool is a run's lanes on one end: the lanes up, the dial in flight,
// the gate on a first lane and the growth of the route.
type pool struct {
	r      *run
	end    laneEnd
	events chan laneEvent
	wg     sync.WaitGroup

	route tv1.Route
	g     growth
	up    []*lane
	// own counts the lanes up that this end dialed.
	own     int
	opening bool
	// gated: a first lane passed end.ready. never: it can't until the
	// route changes.
	gated, never bool
	gateAt       time.Time
	nextDial     time.Time
	// extra counts failed dials while other lanes are up: the route works,
	// so they back off without counting toward giving it up.
	extra int
}

// runLanes keeps the run's lanes until it ends. On a route this end
// dials, it opens lanes one at a time, as many as are wanted. It carries
// the peer's lanes as they arrive on inbound. The receiver judges growth
// each LanesGrow from the bytes its lanes carried, since it sees them
// arrive, and tells the sender in each Want.
func (r *run) runLanes(end laneEnd) {
	p := &pool{r: r, end: end, events: make(chan laneEvent)}
	defer p.drain()
	p.reroute()
	for {
		p.pass()
		gate, redial, judge := p.timers()
		var inbound <-chan *lane
		if p.gated {
			inbound = end.inbound
		}
		select {
		case ev := <-p.events:
			p.ended(ev)
		case l := <-inbound:
			p.up = append(p.up, l)
			r.restartWindow()
			p.wg.Go(func() { p.serve(l, false) })
		case <-gate:
		case <-redial:
		case <-r.wantChanged:
		case <-judge:
			p.judge()
		case <-r.routeChanged():
			p.reroute()
		case <-r.ctx.Done():
			return
		}
	}
}

// drain waits for every lane to let go.
func (p *pool) drain() {
	go func() {
		p.wg.Wait()
		close(p.events)
	}()
	for range p.events {
	}
}

func (p *pool) serve(l *lane, own bool) {
	heard := p.end.serve(l)
	p.r.drop(l)
	p.events <- laneEvent{l: l, route: l.route, own: own, heard: heard}
}

// reroute starts the route the run has now, growth and backoffs anew.
func (p *pool) reroute() {
	p.route = p.r.route()
	p.g = startGrowth(p.route, p.r.e.cfg.Timings)
	if p.end.grows {
		p.r.wantLanes(p.g.lanes)
	}
	p.nextDial, p.gateAt, p.never = time.Time{}, time.Time{}, false
	p.r.restartWindow()
}

// pass gates a first lane once its wait is over, and dials a lane when
// one is wanted and its backoff is over.
func (p *pool) pass() {
	now := time.Now()
	if len(p.up) == 0 && !p.opening && !p.gated && !p.never && !now.Before(p.gateAt) {
		if p.end.ready == nil {
			p.gated = true
		} else {
			retry, ok := p.end.ready()
			p.gated, p.never, p.gateAt = ok, !ok && retry == 0, now.Add(retry)
		}
	}
	if !p.dialing() || now.Before(p.nextDial) {
		return
	}
	p.opening = true
	route := dials(p.r.spec.side, p.route)
	p.wg.Go(func() {
		l, err := p.r.open(route)
		p.events <- laneEvent{l: l, route: route, opened: true, err: err}
		if err == nil {
			p.serve(l, true)
		}
	})
}

// dialing reports whether this end wants to open a lane.
func (p *pool) dialing() bool {
	return p.gated && !p.opening && dials(p.r.spec.side, p.route) != tv1.Route_ROUTE_UNSPECIFIED && p.own < p.wanted()
}

// wanted is the lanes this end keeps: its own growth's when it judges or
// on the no-route wait, the receiver's otherwise.
func (p *pool) wanted() int {
	if p.end.grows || p.route == tv1.Route_ROUTE_BOTH_DIAL {
		return p.g.lanes
	}
	return p.r.peerWants()
}

// timers is when the pool next has something to do: ask the gate again,
// dial again, or judge growth.
func (p *pool) timers() (gate, redial, judge <-chan time.Time) {
	switch {
	case !p.gated && !p.never && len(p.up) == 0 && !p.opening:
		gate = after(p.gateAt)
	case p.dialing():
		redial = after(p.nextDial)
	case p.end.grows && !p.g.settled && !p.opening && len(p.up) == p.g.lanes:
		judge = after(p.r.windowSince().Add(p.r.e.cfg.Timings.LanesGrow))
	}
	return gate, redial, judge
}

// ended takes a dial's end or a lane letting go.
func (p *pool) ended(ev laneEvent) {
	switch {
	case ev.opened && ev.err == nil:
		p.opening = false
		p.own++
		p.up = append(p.up, ev.l)
	case ev.opened:
		p.opening = false
		p.failed(ev.route)
	default:
		p.up = slices.DeleteFunc(p.up, func(l *lane) bool { return l == ev.l })
		if !ev.own {
			break
		}
		p.own--
		if ev.heard {
			p.r.heard(ev.route)
			p.extra = 0
		} else {
			p.failed(ev.route)
		}
	}
	p.gated = p.gated && (len(p.up) > 0 || p.opening)
	p.r.restartWindow()
}

// failed backs off a dial on route that failed. With no lane up it
// counts toward giving the route up.
func (p *pool) failed(route tv1.Route) {
	if p.own == 0 {
		p.nextDial = time.Now().Add(p.r.failed(route))
		return
	}
	p.extra++
	p.nextDial = time.Now().Add(retryDelay(p.extra, p.r.e.cfg.Timings))
}

// judge judges the window that ended, and retires the lane growth no
// longer wants.
func (p *pool) judge() {
	was := p.g.lanes
	if rate, binding, ok := p.r.judgeWindow(); ok {
		p.g = p.g.next(rate, binding, p.r.e.cfg.Timings)
	}
	p.r.wantLanes(p.g.lanes)
	if p.g.lanes < was {
		p.r.retire(p.up[len(p.up)-1])
	}
}

// wantLanes sets the lanes this end wants, which the receiver sends in each
// Want.
func (r *run) wantLanes(n int) {
	r.lanesWanted.Store(int32(n)) //nolint:gosec // at most LanesMax
}

// peerWants is the lanes the receiver last wanted, LanesStart until it
// said.
func (r *run) peerWants() int {
	if n := r.lanesWanted.Load(); n > 0 {
		return int(n)
	}
	return r.e.cfg.Timings.LanesStart
}

// peerWanted notes the lanes the receiver wants, from a Want.
func (r *run) peerWanted(n uint32) {
	if n == 0 || r.lanesWanted.Swap(int32(n)) == int32(n) { //nolint:gosec // a lane count
		return
	}
	poke(r.wantChanged)
}

// restartWindow starts the growth window over: the lane count or a wait
// changed, so the bytes since say nothing about the lanes now.
func (r *run) restartWindow() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.restartWindowLocked()
}

func (r *run) restartWindowLocked() {
	r.win = window{since: time.Now(), bytes: r.carried.Load()}
}

func (r *run) windowSince() time.Time {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.win.since
}

// judgeWindow is the rate over the window that just ended and whether
// the send limit bound in it, and starts the next. ok is false while the
// receiver waits, when the window says nothing about lanes.
func (r *run) judgeWindow() (rate float64, binding, ok bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	w := r.win
	r.restartWindowLocked()
	if r.rx.wait.GetKind() != tv1.Wait_KIND_UNSPECIFIED {
		return 0, false, false
	}
	return float64(r.win.bytes-w.bytes) / r.win.since.Sub(w.since).Seconds(), w.binding, true
}

// after fires at t, at once when t has passed.
func after(t time.Time) <-chan time.Time {
	return time.After(time.Until(t))
}
