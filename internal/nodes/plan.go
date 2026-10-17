package nodes

import (
	"crypto/sha256"
	"encoding/hex"
	"maps"
	"slices"
	"strings"

	"github.com/jatassi/slipstream/internal/role"
)

// Roles is an assignment: the Roles each Node holds. A Node with no Roles
// has no entry.
type Roles map[ID]role.Set

// Basis is a digest of the assignment and forfeit rows a plan was made
// on, so Apply can refuse a change planned against rows that moved since.
type Basis string

// Noun names a kind of held work, "torrent" and "torrents".
type Noun struct{ One, Many string }

// Held is the work a Node holds for a Role, as its owner counts it.
type Held struct {
	Units int64
	Noun  Noun
	// Bytes is 0 when sizes don't apply.
	Bytes int64
	// Endless: the holder can't finish this work on its own, so it stays
	// until it is moved or written off.
	Endless bool
}

type NodeRole struct {
	Node ID
	Role role.Name
}

// ReachResult is a reach test run from a Node against a Role's target.
type ReachResult struct {
	Target string
	OK     bool
	Millis int64
	Err    string
}

// Seat is one live Node as a plan sees it.
type Seat struct {
	Named
	Assigned role.Set
	Forfeit  role.Set
	// Away: out of step, so its probes and reach tests can't run.
	Away   bool
	Probes map[role.Probe]role.Fit
	// Held is the tied work each Role's owner counts on the Node: what drains
	// and what a write-off gives up.
	Held map[role.Name]Held
	// Moves is the portable work each Role's owner moves to the heir once
	// the Node loses the Role.
	Moves map[role.Name]Held
}

// Situation is everything Plan reads: the live Nodes in id order, the
// Reach Roles whose owner has a target configured, and the reach tests
// run for this call.
type Situation struct {
	Controller ID
	Seats      []Seat
	Targets    role.Set
	Reach      map[NodeRole]ReachResult
}

// Change is a write to Roles: SetRoles, WriteOff or Remove.
type Change interface{ change() }

type (
	// SetRoles changes the Roles of the Nodes Want names, planned on Basis.
	// Nodes it doesn't name keep theirs.
	SetRoles struct {
		Basis Basis
		Want  Roles
	}
	// WriteOff gives up what Node holds for Roles. A Role still ticked on
	// Node is unticked in the same Write.
	WriteOff struct {
		Node  ID
		Roles role.Set
	}
	// Remove unticks every Role on Node and leaves a tombstone. It is
	// blocked while Node holds work, unless WriteOff gives the work up.
	Remove struct {
		Node     ID
		WriteOff bool
	}
)

func (SetRoles) change() {}
func (WriteOff) change() {}
func (Remove) change()   {}

// Outcome is what a Change does. Next and Forfeit are the whole
// assignment and forfeit rows Apply stores.
type Outcome struct {
	Basis   Basis
	Next    Roles
	Forfeit Roles
	Effects []Effect
	Checks  []Check
}

func (o Outcome) Blocked() bool {
	return slices.ContainsFunc(o.Checks, func(c Check) bool { return c.Verdict == Block })
}

// Same reports whether applying o would store nothing new over s.
func (o Outcome) Same(s Situation) bool {
	leaves := slices.ContainsFunc(o.Effects, func(e Effect) bool { _, ok := e.(Leaves); return ok })
	return !leaves && maps.Equal(o.Next, s.assigned()) && maps.Equal(o.Forfeit, s.forfeited())
}

type Verdict uint8

const (
	Allow Verdict = iota
	Warn
	Block
)

type Reason string

const (
	RoleTaken        Reason = "role_taken"
	ProbeFails       Reason = "probe_fails"
	ProbeWeak        Reason = "probe_weak"
	OfflineUnchecked Reason = "offline_unchecked"
	ReachFailed      Reason = "reach_failed"
	Reached          Reason = "reached"
	NothingToTest    Reason = "nothing_to_test"
	IsController     Reason = "node_is_controller"
	HoldsWork        Reason = "node_holds_work"
	NothingHeld      Reason = "nothing_held"
	KeepsRole        Reason = "keeps_role"
)

// Check is a guardrail's answer for one Node and Role. Role is "" for a
// check on the whole Node.
type Check struct {
	Node        Named
	Role        role.Name
	Verdict     Verdict
	Reason      Reason
	Words       string
	gestureWord string
}

func (s Situation) Basis() Basis {
	var rows []string
	for _, seat := range s.Seats {
		for _, r := range seat.Assigned.Names() {
			rows = append(rows, "assigned "+seat.ID.String()+" "+string(r))
		}
		for _, r := range seat.Forfeit.Names() {
			rows = append(rows, "forfeit "+seat.ID.String()+" "+string(r))
		}
	}
	slices.Sort(rows)
	sum := sha256.Sum256([]byte(strings.Join(rows, "\n")))
	return Basis(hex.EncodeToString(sum[:16]))
}

func (s Situation) assigned() Roles {
	out := Roles{}
	for _, seat := range s.Seats {
		if seat.Assigned != 0 {
			out[seat.ID] = seat.Assigned
		}
	}
	return out
}

func (s Situation) forfeited() Roles {
	out := Roles{}
	for _, seat := range s.Seats {
		if seat.Forfeit != 0 {
			out[seat.ID] = seat.Forfeit
		}
	}
	return out
}

func (s Situation) seat(id ID) Seat {
	for _, seat := range s.Seats {
		if seat.ID == id {
			return seat
		}
	}
	return Seat{Named: Named{ID: id}}
}

// Plan evaluates c against s with the Role table's rules, in order: the
// Controller keeps a Controller Role, an Always Role with no holder goes
// back to the Controller, Max blocks an added holder past it, then probes
// and reach tests on each added Role. An unticked Role drains while its
// Node holds work for it, and its portable work moves to the heir.
func Plan(s Situation, c Change) Outcome {
	p := planner{
		s:       s,
		was:     s.assigned(),
		lost:    s.forfeited(),
		written: map[NodeRole]bool{},
		forced:  map[NodeRole]bool{},
	}
	p.next, p.forfeit = maps.Clone(p.was), maps.Clone(p.lost)
	switch c := c.(type) {
	case SetRoles:
		for _, seat := range s.Seats {
			if set, ok := c.Want[seat.ID]; ok {
				p.next[seat.ID] = set
			}
		}
	case WriteOff:
		p.writeOff(s.seat(c.Node), c.Roles)
	case Remove:
		if c.Node == s.Controller {
			p.checks = append(p.checks, isController(s.seat(c.Node).Named))
			return p.outcome()
		}
		p.remove(s.seat(c.Node), c.WriteOff)
	}
	p.rules()
	p.settleWriteOffs()
	prune(p.next)
	prune(p.forfeit)
	p.describe()
	return p.outcome()
}

type planner struct {
	s             Situation
	was, lost     Roles
	next, forfeit Roles
	writing       []NodeRole
	written       map[NodeRole]bool
	forced        map[NodeRole]bool
	leaving       *Seat
	out           []Effect
	handbacks     []Effect
	checks        []Check
}

func (p *planner) writeOff(seat Seat, roles role.Set) {
	held := false
	for _, r := range roles.Names() {
		if seat.Held[r].Units == 0 {
			continue
		}
		held = true
		p.next[seat.ID] = p.next[seat.ID].Without(r)
		p.writing = append(p.writing, NodeRole{seat.ID, r})
	}
	if !held {
		p.checks = append(p.checks, nothingHeld(seat.Named, roles))
	}
}

// settleWriteOffs forfeits each Role WriteOff gave up, unless the Always
// rule handed it straight back to the same Node.
func (p *planner) settleWriteOffs() {
	for _, at := range p.writing {
		seat := p.s.seat(at.Node)
		switch {
		case p.next[at.Node].Has(at.Role):
			p.checks = append(p.checks, keepsRole(seat.Named, at.Role, seat.Held[at.Role]))
		case !p.lost[at.Node].Has(at.Role):
			p.written[at] = true
			p.forfeit[at.Node] = p.forfeit[at.Node].With(at.Role)
		}
	}
}

func (p *planner) remove(seat Seat, writeOff bool) {
	p.leaving = &seat
	delete(p.next, seat.ID)
	delete(p.forfeit, seat.ID)
	for _, r := range role.Names() {
		if seat.Held[r].Units == 0 || p.lost[seat.ID].Has(r) {
			continue
		}
		if writeOff {
			p.written[NodeRole{seat.ID, r}] = true
		} else {
			p.checks = append(p.checks, holdsWork(seat.Named, r, seat.Held[r]))
		}
	}
}

func (p *planner) rules() {
	ctrl := p.s.seat(p.s.Controller)
	for _, r := range role.Names() {
		rule := r.Rule()
		if rule.Controller && !p.next[ctrl.ID].Has(r) {
			p.force(ctrl.ID, r)
			p.handbacks = append(p.handbacks, Pinned{Node: ctrl.Named, Role: r})
		}
		if rule.Always && !p.held(r) {
			p.force(ctrl.ID, r)
			p.handbacks = append(p.handbacks, HandedBack{Role: r, From: p.holders(p.was, r), To: ctrl.Named})
		}
		if rule.Max > 0 {
			p.max(r, rule.Max)
		}
	}
	for id, set := range p.forfeit {
		p.forfeit[id] = set &^ p.next[id]
	}
}

func (p *planner) force(id ID, r role.Name) {
	p.next[id] = p.next[id].With(r)
	p.forced[NodeRole{id, r}] = true
}

func (p *planner) held(r role.Name) bool {
	for _, set := range p.next {
		if set.Has(r) {
			return true
		}
	}
	return false
}

func (p *planner) holders(in Roles, r role.Name) []Named {
	var out []Named
	for _, seat := range p.s.Seats {
		if in[seat.ID].Has(r) {
			out = append(out, seat.Named)
		}
	}
	return out
}

// max blocks each holder r gains past most, naming a holder that stays,
// or else another added one.
func (p *planner) max(r role.Name, most int) {
	holders := p.holders(p.next, r)
	if len(holders) <= most {
		return
	}
	for _, added := range holders {
		if p.was[added.ID].Has(r) {
			continue
		}
		named := holders[0]
		for _, h := range holders {
			if h.ID != added.ID && (named.ID == added.ID || p.was[h.ID].Has(r)) {
				named = h
			}
		}
		p.checks = append(p.checks, roleTaken(added, r, named))
	}
}

func (p *planner) describe() {
	for _, seat := range p.s.Seats {
		for _, r := range role.Names() {
			p.effectOf(seat, r)
		}
	}
	for _, seat := range p.s.Seats {
		for _, r := range role.Names() {
			p.move(seat, r)
		}
	}
	p.out = append(p.out, p.handbacks...)
	if p.leaving != nil {
		p.out = append(p.out, Leaves{Node: p.leaving.Named})
	}
}

func (p *planner) effectOf(seat Seat, r role.Name) {
	at := NodeRole{seat.ID, r}
	was, now := p.was[seat.ID].Has(r), p.next[seat.ID].Has(r)
	switch {
	case p.written[at]:
		p.out = append(p.out, WritesOff{Node: seat.Named, Role: r, Held: seat.Held[r]})
	case p.leaving != nil && p.leaving.ID == seat.ID:
	case now && !was && !p.forced[at]:
		p.out = append(p.out, Gains{Node: seat.Named, Role: r})
		p.test(seat, r)
	case was && !now && seat.Held[r].Units > 0:
		p.out = append(p.out, Drains{Node: seat.Named, Role: r, Held: seat.Held[r]})
	case was && !now:
		p.out = append(p.out, LetsGo{Node: seat.Named, Role: r})
	}
}

func (p *planner) move(seat Seat, r role.Name) {
	moved := seat.Moves[r]
	if moved.Units == 0 || !p.was[seat.ID].Has(r) || p.next[seat.ID].Has(r) {
		return
	}
	if to, ok := p.heir(r); ok {
		p.out = append(p.out, Moves{Role: r, From: seat.Named, To: to, Held: moved})
	}
}

// heir is where an unticked Role's portable work moves: the Controller
// when it holds the Role next, else the first holder by id.
func (p *planner) heir(r role.Name) (Named, bool) {
	if p.next[p.s.Controller].Has(r) {
		return p.s.seat(p.s.Controller).Named, true
	}
	if holders := p.holders(p.next, r); len(holders) > 0 {
		return holders[0], true
	}
	return Named{}, false
}

func (p *planner) test(seat Seat, r role.Name) {
	rule := r.Rule()
	if rule.Probe == "" && !rule.Reach {
		return
	}
	if seat.Away {
		p.checks = append(p.checks, offline(seat.Named, r))
		return
	}
	switch seat.Probes[rule.Probe] {
	case role.Fails:
		p.checks = append(p.checks, probed(seat.Named, r, Block))
	case role.Weak:
		p.checks = append(p.checks, probed(seat.Named, r, Warn))
	case role.Fits:
	}
	if !rule.Reach {
		return
	}
	if !p.s.Targets.Has(r) {
		p.checks = append(p.checks, nothingToTest(seat.Named, r))
		return
	}
	if res, ok := p.s.Reach[NodeRole{seat.ID, r}]; ok {
		p.checks = append(p.checks, reached(seat.Named, r, res))
	}
}

func (p *planner) outcome() Outcome {
	return Outcome{
		Basis:   p.s.Basis(),
		Next:    p.next,
		Forfeit: p.forfeit,
		Effects: p.out,
		Checks:  p.checks,
	}
}

func prune(rs Roles) {
	maps.DeleteFunc(rs, func(_ ID, set role.Set) bool { return set == 0 })
}

// Gesture is the plan's verdict on a drag: a move of Role from From to
// To, or an add onto To when From is zero.
type Gesture struct {
	Role    role.Name
	From    ID
	To      ID
	Verdict Verdict
	Word    string
	// Note is the words of a Role the gesture hands back or pins.
	Note string
}

// Gestures plans every move and add on s, without reach tests, so a drag
// needs no round trip.
func Gestures(s Situation) []Gesture {
	s.Reach = nil
	was := s.assigned()
	var out []Gesture
	for _, r := range role.Names() {
		for _, to := range s.Seats {
			if was[to.ID].Has(r) {
				continue
			}
			for _, from := range s.Seats {
				if was[from.ID].Has(r) {
					out = append(out, gesture(s, was, r, from.ID, to.ID))
				}
			}
			out = append(out, gesture(s, was, r, ID{}, to.ID))
		}
	}
	return out
}

func gesture(s Situation, was Roles, r role.Name, from, to ID) Gesture {
	want := Roles{to: was[to].With(r)}
	g := Gesture{Role: r, From: from, To: to, Verdict: Allow, Word: "Adds " + r.Label()}
	if !from.IsZero() {
		want[from] = was[from].Without(r)
		g.Word = "Moves " + r.Label()
	}
	if r.Rule().Reach && s.Targets.Has(r) {
		g.Word = "Tested when you drop"
	}
	o := Plan(s, SetRoles{Want: want})
	for _, c := range o.Checks {
		if c.Verdict > g.Verdict {
			g.Verdict, g.Word = c.Verdict, c.gestureWord
		}
	}
	for _, e := range o.Effects {
		switch e.(type) {
		case HandedBack, Pinned:
			g.Note = e.Words()
		}
	}
	return g
}
