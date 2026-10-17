package nodes

import (
	"maps"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/role"
)

// inputs is everything the board derives from: the stored rows, the grace
// clock's pauses and what the roster remembers of each Node's sessions.
type inputs struct {
	controller ID
	start      time.Time
	grace      time.Duration
	blip       time.Duration
	settle     time.Duration
	pausing    int
	// pauses is every stop of the grace clock, the last one open while
	// pausing is above zero.
	pauses []span
	// nodes is the live rows in id order.
	nodes []nodeIn
}

type span struct{ from, to time.Time }

type nodeIn struct {
	id       ID
	name     string
	assigned role.Set
	forfeit  role.Set
	// work is the Roles whose owner counts tied work on the Node.
	work role.Set
	// attached: the Node's Sync session is attached, so it is in step.
	attached bool
	since    time.Time
	// seen is its last Sync frame, or seen_at as stored.
	seen time.Time
	// out is when its last session ended, zero when none ended since start.
	out        time.Time
	mismatched bool
	// back is when it came back in step from a lapse, zero when it didn't.
	back time.Time
}

// RoleState is what a Node is to a Role at one instant.
type RoleState uint8

const (
	NoState RoleState = iota
	Holding
	Away
	Lapsed
	Draining
	WritingOff
	Covering
)

func (s RoleState) String() string {
	return [...]string{"", "holding", "away", "lapsed", "draining", "writing-off", "covering"}[s]
}

// Standing is how a Node stands with the Controller at one instant.
type Standing uint8

const (
	InStep Standing = iota
	Reconnecting
	Offline
	LapsedNode
	MismatchedNode
)

// NodeView is one Node on the board.
type NodeView struct {
	Named
	Controller bool
	Standing   Standing
	// Since is when it came in step, or, while away, its last Sync frame.
	Since time.Time
	// Settling: back from a lapse for less than the settle time, it may
	// hold its Roles but takes no new work.
	Settling bool
}

// Board is the roster at one instant. It never changes once derived, and
// its Changed channel closes when a newer board replaces it.
type Board struct {
	nodes   []NodeView
	states  []map[role.Name]RoleState
	forfeit []role.Set
	changed chan struct{}
}

func (b *Board) Changed() <-chan struct{} { return b.changed }

func (b *Board) Node(id ID) (NodeView, bool) {
	i, ok := b.index(id)
	if !ok {
		return NodeView{}, false
	}
	return b.nodes[i], true
}

func (b *Board) index(id ID) (int, bool) {
	return slices.BinarySearchFunc(b.nodes, id, func(n NodeView, id ID) int { return n.ID.Compare(id.UUID) })
}

// same reports whether b and o say the same about every Node.
func (b *Board) same(o *Board) bool {
	return slices.Equal(b.nodes, o.nodes) && slices.Equal(b.forfeit, o.forfeit) &&
		slices.EqualFunc(b.states, o.states, maps.Equal)
}

// covering is the Roles the Controller covers.
func (b *Board) covering() role.Set {
	var out role.Set
	for i, n := range b.nodes {
		if !n.Controller {
			continue
		}
		for r, st := range b.states[i] {
			if st == Covering {
				out = out.With(r)
			}
		}
	}
	return out
}

// RoleView is one Role a Node stands in and the tied work it holds for it.
type RoleView struct {
	Role  role.Name
	State RoleState
	// Held is zero when the Node holds no tied work for Role.
	Held Held
}

// Roles is every Role id stands in on b, in table order, each with its
// tied work from held.
func (b *Board) Roles(id ID, held map[role.Name]Held) []RoleView {
	var out []RoleView
	for _, r := range role.Names() {
		if st := b.Who(r).State(id); st != NoState {
			out = append(out, RoleView{Role: r, State: st, Held: held[r]})
		}
	}
	return out
}

func (b *Board) Who(r role.Name) Holders { return Holders{b: b, r: r} }

// Holders answers who may hold one Role on one board.
type Holders struct {
	b *Board
	r role.Name
}

func (h Holders) State(n ID) RoleState {
	i, ok := h.b.index(n)
	if !ok {
		return NoState
	}
	return h.b.states[i][h.r]
}

// Holds reports whether n holds the Role: assigned, in step or not, or
// covering.
func (h Holders) Holds(n ID) bool {
	switch h.State(n) {
	case Holding, Away, Lapsed, Covering:
		return true
	case NoState, Draining, WritingOff:
	}
	return false
}

// May reports whether n takes new work for the Role: holding or covering,
// in step and not settling.
func (h Holders) May(n ID) bool {
	switch h.State(n) {
	case Holding, Covering:
		v, _ := h.b.Node(n)
		return !v.Settling
	case NoState, Away, Lapsed, Draining, WritingOff:
	}
	return false
}

// Serves reports whether n is assigned the Role, draining it, writing it
// off or covering it. The dispatch gate asks it.
func (h Holders) Serves(n ID) bool { return h.State(n) != NoState }

// Pick is prefer when it may, else the Controller when it may, else the
// first Node by id that may.
func (h Holders) Pick(prefer ID) (ID, bool) {
	if h.May(prefer) {
		return prefer, true
	}
	for _, n := range h.b.nodes {
		if n.Controller && h.May(n.ID) {
			return n.ID, true
		}
	}
	for _, n := range h.b.nodes {
		if h.May(n.ID) {
			return n.ID, true
		}
	}
	return ID{}, false
}

func (h Holders) Lapsed(n ID) bool { return h.State(n) == Lapsed }

func (h Holders) Covering() bool {
	for _, n := range h.b.nodes {
		if n.Controller {
			return h.State(n.ID) == Covering
		}
	}
	return false
}

// WrittenOff reports whether the Admin wrote off what n holds for the Role, so its owner
// settles that work.
func (h Holders) WrittenOff(n ID) bool {
	i, ok := h.b.index(n)
	return ok && h.b.forfeit[i].Has(h.r)
}

// derive is the board at now. The grace runs from max(seen, start) on the
// grace clock, so a Controller down for hours lapses nobody when it starts.
func derive(in inputs, now time.Time) *Board {
	b := &Board{
		nodes:   make([]NodeView, len(in.nodes)),
		states:  make([]map[role.Name]RoleState, len(in.nodes)),
		forfeit: make([]role.Set, len(in.nodes)),
		changed: make(chan struct{}),
	}
	for i, n := range in.nodes {
		b.nodes[i] = in.view(n, now)
		b.forfeit[i] = n.forfeit
	}
	for i, n := range in.nodes {
		states := map[role.Name]RoleState{}
		for _, r := range role.Names() {
			if st := in.state(n, b.nodes[i].Standing == InStep, r, now); st != NoState {
				states[r] = st
			}
		}
		b.states[i] = states
	}
	return b
}

func (in *inputs) view(n nodeIn, now time.Time) NodeView {
	v := NodeView{Named: Named{ID: n.id, Name: n.name}, Controller: n.id == in.controller}
	switch {
	case v.Controller:
		v.Since = in.start
	case n.attached:
		v.Since = n.since
		v.Settling = in.settling(n, now)
	default:
		v.Since = n.seen
		if v.Since.IsZero() {
			v.Since = in.start
		}
		switch {
		case n.mismatched:
			v.Standing = MismatchedNode
		case in.lapsed(n, now):
			v.Standing = LapsedNode
		case now.Before(later(n.out, in.start).Add(in.blip)):
			v.Standing = Reconnecting
		default:
			v.Standing = Offline
		}
	}
	return v
}

// state is the truth table for one Node and Role.
func (in *inputs) state(n nodeIn, inStep bool, r role.Name, now time.Time) RoleState {
	switch {
	case n.assigned.Has(r) && inStep:
		return Holding
	case n.assigned.Has(r) && in.lapsed(n, now):
		return Lapsed
	case n.assigned.Has(r):
		return Away
	case n.work.Has(r) && n.forfeit.Has(r):
		return WritingOff
	case n.work.Has(r):
		return Draining
	case n.id == in.controller && in.covers(r, now):
		return Covering
	}
	return NoState
}

// covers: the Controller covers r for its assigned holders while every
// one of them is lapsed or settling, and r's work goes to the Controller.
func (in *inputs) covers(r role.Name, now time.Time) bool {
	if r.Rule().Lapse != role.ToController {
		return false
	}
	holders := 0
	for _, n := range in.nodes {
		if !n.assigned.Has(r) {
			continue
		}
		if !in.lapsed(n, now) && !in.settling(n, now) {
			return false
		}
		holders++
	}
	return holders > 0
}

func (in *inputs) lapsed(n nodeIn, now time.Time) bool {
	return n.id != in.controller && !n.attached && in.elapsed(later(n.seen, in.start), now) >= in.grace
}

func (in *inputs) settling(n nodeIn, now time.Time) bool {
	return n.attached && !n.back.IsZero() && now.Before(n.back.Add(in.settle))
}

// elapsed is the grace clock from from to now: wall time less every
// pause's overlap with it.
func (in *inputs) elapsed(from, now time.Time) time.Duration {
	d := now.Sub(from)
	for _, p := range in.pauses {
		to := p.to
		if to.IsZero() {
			to = now
		}
		if lo, hi := later(p.from, from), earlier(to, now); hi.After(lo) {
			d -= hi.Sub(lo)
		}
	}
	return d
}

// nextEdge is the earliest instant after now where derive's answer can
// change with nothing else changing: a blip ending, a grace ending or a
// settle ending. It is zero when there is none.
func nextEdge(in inputs, now time.Time) time.Time {
	var next time.Time
	at := func(t time.Time) {
		if t.After(now) && (next.IsZero() || t.Before(next)) {
			next = t
		}
	}
	for _, n := range in.nodes {
		switch {
		case n.id == in.controller:
		case n.attached:
			if !n.back.IsZero() {
				at(n.back.Add(in.settle))
			}
		default:
			at(later(n.out, in.start).Add(in.blip))
			if in.pausing == 0 && !in.lapsed(n, now) {
				at(now.Add(in.grace - in.elapsed(later(n.seen, in.start), now)))
			}
		}
	}
	return next
}

func (in *inputs) node(id ID) *nodeIn {
	for i := range in.nodes {
		if in.nodes[i].id == id {
			return &in.nodes[i]
		}
	}
	return nil
}

// attach puts id in step. A Node back from a lapse settles first.
func (in *inputs) attach(id ID, now time.Time) {
	n := in.node(id)
	if n == nil || n.attached {
		return
	}
	n.back = time.Time{}
	if in.lapsed(*n, now) {
		n.back = now
	}
	n.attached, n.since = true, now
}

func (in *inputs) detach(id ID, now time.Time) {
	if n := in.node(id); n != nil && n.attached {
		n.attached, n.out = false, now
	}
}

func (in *inputs) pause(now time.Time) {
	if in.pausing == 0 {
		in.pauses = append(in.pauses, span{from: now})
	}
	in.pausing++
}

func (in *inputs) resume(now time.Time) {
	in.pausing--
	if in.pausing == 0 {
		in.pauses[len(in.pauses)-1].to = now
	}
}

func later(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

func earlier(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}
