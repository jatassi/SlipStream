package plan

import (
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
)

// The stages Run reads. conversions owns the names.
const (
	queued    = "queued"
	copying   = "copying"
	ready     = "ready"
	running   = "running"
	paused    = "paused"
	checking  = "checking"
	returning = "returning"
	landing   = "landing"
)

// onNode is the stages of a row placed on a Node.
var onNode = []string{copying, ready, running, paused, checking, returning, landing}

// never is a time the planner can't tell: a Node that is away, lapsed or
// held by a transcode starts nothing at a known time.
var never = time.Date(9999, 1, 1, 0, 0, 0, 0, time.UTC)

func known(t time.Time) bool { return t.Before(never) }

// Run steps virtual time from s.Now. until == s.Now returns the Step; a
// later until also returns each job's Plan.
func Run(s Snapshot, until time.Time) (Step, Forecast) {
	m := newSim(s)
	m.seat()
	m.schedule()
	m.pass(s.Now, true)
	m.schedule()
	m.pass(s.Now, false)
	m.step.Tests = m.tests()
	m.step.Causes = m.causes()
	if !until.After(s.Now) {
		return m.step, nil
	}
	for t := s.Now; len(m.waiting) > 0; {
		var ok bool
		if t, ok = m.next(t); !ok || t.After(until) {
			break
		}
		m.release(t)
		m.pass(t, false)
	}
	return m.step, m.forecast()
}

type nodeSim struct {
	Node
	// base is when each of AtOnce slots frees once the running rows end,
	// slots the same after every job given to the Node.
	base, slots []time.Time
	units       map[storage.UUID]*unit
	scratch     int64
	// pending is the placed rows not started at s.Now, scheduled in queue
	// order once the Step is decided.
	pending []int
	// stalled Nodes start nothing at a known time; open Nodes take work.
	stalled, open bool
}

// unit is one source on a Node, however many jobs read it.
type unit struct {
	arrives time.Time
	jobs    int
	reserve int64
}

// free is the slot that frees first.
func (n *nodeSim) free() int {
	k := 0
	for i, t := range n.slots {
		if t.Before(n.slots[k]) {
			k = i
		}
	}
	return k
}

func (n *nodeSim) full() bool { return len(n.units) > n.AtOnce }

type sim struct {
	s     Snapshot
	nodes []*nodeSim
	byID  map[role.NodeID]*nodeSim
	roots map[storage.UUID]int64
	// held is how many Nodes hold each source.
	held map[storage.UUID]int
	ends ends
	// fixed is the ends of rows past their start at s.Now.
	fixed ends
	// waiting is the queued jobs not yet given a Node, in queue order.
	waiting []int
	fits    [][]*nodeSim
	plans   []Plan
	// byHours is set when a job's start waits for its Node's hours, copies
	// when its source is still on its way at s.Now.
	byHours, copies []bool
	placed          []bool
	skip            []Wait
	step            Step
}

func newSim(s Snapshot) *sim {
	m := &sim{
		s: s, byID: map[role.NodeID]*nodeSim{}, roots: map[storage.UUID]int64{}, held: map[storage.UUID]int{},
		plans: make([]Plan, len(s.Jobs)), byHours: make([]bool, len(s.Jobs)), copies: make([]bool, len(s.Jobs)),
		placed: make([]bool, len(s.Jobs)), skip: make([]Wait, len(s.Jobs)), fits: make([][]*nodeSim, len(s.Jobs)),
	}
	for _, n := range s.Nodes {
		ns := &nodeSim{
			Node: n, units: map[storage.UUID]*unit{}, scratch: n.ScratchFree,
			stalled: n.Away || n.Lapsed || n.Hold, open: n.Holds && !n.Away && !n.Lapsed && n.May,
		}
		m.nodes = append(m.nodes, ns)
		m.byID[n.ID] = ns
	}
	slices.SortFunc(m.nodes, func(a, b *nodeSim) int { return a.ID.Compare(b.ID.UUID) })
	for _, j := range s.Jobs {
		if _, ok := m.roots[j.Root]; !ok {
			m.roots[j.Root] = j.RootRoom
		}
	}
	return m
}

// seat puts every placed row on its Node: running rows take slots, and the
// rest wait in queue order for one.
func (m *sim) seat() {
	for _, n := range m.nodes {
		m.seatRunning(n)
	}
	for i, j := range m.s.Jobs {
		switch {
		case j.Stage == queued:
			m.waiting = append(m.waiting, i)
			m.fits[i] = m.qualifying(j)
		case j.Node != nil && slices.Contains(onNode, j.Stage):
			m.seatPlaced(i, m.byID[*j.Node])
		}
	}
}

// seatRunning gives n's running rows its slots, which free as they end.
func (m *sim) seatRunning(n *nodeSim) {
	var ends []time.Time
	for i, j := range m.s.Jobs {
		if j.Node == nil || *j.Node != n.ID || (j.Stage != running && j.Stage != paused) {
			continue
		}
		end := never
		if !n.stalled {
			end = m.s.Now.Add(left(n, j))
		}
		ends = append(ends, end)
		m.plans[i] = Plan{Node: &n.ID, Finish: at(end), Wait: m.stalledWait(n)}
	}
	slices.SortFunc(ends, func(a, b time.Time) int { return a.Compare(b) })
	n.base = make([]time.Time, n.AtOnce)
	for k := range n.base {
		n.base[k] = m.s.Now
		if n.stalled {
			n.base[k] = never
		}
	}
	copy(n.base[max(0, n.AtOnce-len(ends)):], ends[max(0, len(ends)-n.AtOnce):])
	n.slots = slices.Clone(n.base)
}

// seatPlaced puts a row the snapshot placed on n into n's source units.
func (m *sim) seatPlaced(i int, n *nodeSim) {
	j := m.s.Jobs[i]
	m.placed[i] = true
	switch j.Stage {
	case landing:
		m.plans[i] = Plan{Node: &n.ID}
		return
	case checking, returning:
		m.plans[i] = Plan{Node: &n.ID}
		m.fixed = append(m.fixed, end{t: m.s.Now, job: i})
	case running, paused:
		if e := m.plans[i].Finish; e != nil {
			m.fixed = append(m.fixed, end{t: *e, job: i})
		}
	case copying, ready:
		n.pending = append(n.pending, i)
	}
	u := m.unitOn(n, j)
	if j.Stage == copying {
		u.arrives = latest(u.arrives, m.s.Now.Add(m.copyTime(j, n)))
	}
	u.jobs++
}

// left is a running row's time to its end: from its facts, or its whole
// run when it reports none.
func left(n *nodeSim, j Job) time.Duration {
	for _, r := range n.Running {
		if r.Job == j.ID {
			return r.Left
		}
	}
	return runtime(j, n.Proofs[j.Key])
}

func (m *sim) qualifying(j Job) []*nodeSim {
	var out []*nodeSim
	for _, n := range m.nodes {
		if Qualifies(n.Node, j) == Fits {
			out = append(out, n)
		}
	}
	return out
}

func (m *sim) unitOn(n *nodeSim, j Job) *unit {
	if u, ok := n.units[j.Source]; ok {
		return u
	}
	u := &unit{arrives: m.s.Now}
	if j.SourceNode != n.ID {
		u.reserve = j.SourceBytes
	}
	n.units[j.Source] = u
	m.held[j.Source]++
	return u
}

// someFree is whether an open Node can take one more source.
func (m *sim) someFree() bool {
	return slices.ContainsFunc(m.nodes, func(n *nodeSim) bool { return n.open && !n.full() })
}

func due(j Job, t time.Time) bool { return !j.Source.IsZero() && !j.NotBefore.After(t) }

// pass gives the due waiting jobs a Node in queue order, at virtual time t.
// It passes again when a new source on a Node lets a job it skipped share
// it, so a rerun on its own Step places nothing more.
func (m *sim) pass(t time.Time, record bool) {
	for again := true; again; {
		again = false
		skipped := map[storage.UUID]bool{}
		left := m.waiting[:0]
		free := m.someFree()
		for _, i := range m.waiting {
			j := m.s.Jobs[i]
			if !due(j, t) {
				left = append(left, i)
				continue
			}
			if !free && m.held[j.Source] == 0 {
				m.skip[i] = Wait{Code: InLine}
				left = append(left, i)
				continue
			}
			n := m.pick(i, t)
			if n == nil {
				skipped[j.Source] = true
				left = append(left, i)
				continue
			}
			if _, ok := n.units[j.Source]; !ok && skipped[j.Source] {
				again = true
			}
			m.give(i, n, t, record)
			free = m.someFree()
		}
		m.waiting = left
	}
}

// pick is the Node with the earliest finish among those j qualifies for
// and that have a source unit and room for it. A job whose source is on
// Nodes it qualifies for already goes to one of them. Ties go to the Node
// holding the source, then the lower id.
func (m *sim) pick(i int, t time.Time) *nodeSim {
	j := m.s.Jobs[i]
	var holding []*nodeSim
	for _, n := range m.fits[i] {
		if _, ok := n.units[j.Source]; ok {
			holding = append(holding, n)
		}
	}
	cands := m.fits[i]
	if len(holding) > 0 {
		cands = holding
	}
	m.skip[i] = Wait{Code: InLine}
	var best *nodeSim
	var bestEnd time.Time
	for _, n := range cands {
		_, has := n.units[j.Source]
		if !n.open || (!has && n.full()) {
			continue
		}
		if free, need := m.room(n, j); need > free {
			m.skip[i] = Wait{Code: Room, Node: &n.ID, Free: free, Need: need}
			continue
		}
		_, _, end := m.finish(n, j, t)
		if best == nil || end.Before(bestEnd) || (end.Equal(bestEnd) && j.SourceNode == n.ID && j.SourceNode != best.ID) {
			best, bestEnd = n, end
		}
	}
	return best
}

// room is the free bytes j needs on n and what it needs there: the scratch
// volume for a copied source and a scratch output, or the root's volume for
// an output written into the root's holding folder.
func (m *sim) room(n *nodeSim, j Job) (free, need int64) {
	if _, ok := n.units[j.Source]; !ok && j.SourceNode != n.ID {
		need = j.SourceBytes
	}
	if j.RootNode == n.ID {
		if need > n.scratch {
			return n.scratch, need
		}
		return m.roots[j.Root], j.OutBytes
	}
	return n.scratch, need + j.OutBytes
}

func (m *sim) give(i int, n *nodeSim, t time.Time, record bool) {
	j := m.s.Jobs[i]
	_, has := n.units[j.Source]
	copied := !has && j.SourceNode != n.ID
	u := m.unitOn(n, j)
	if !has {
		u.arrives = t
		if copied {
			u.arrives = t.Add(m.copyTime(j, n))
		}
		n.scratch -= u.reserve
	}
	u.jobs++
	if j.RootNode == n.ID {
		m.roots[j.Root] -= j.OutBytes
	} else {
		n.scratch -= j.OutBytes
	}
	m.placed[i] = true
	if record {
		m.step.Placed = append(m.step.Placed, Placed{
			Job: j.ID, Stage: j.Stage, Attempt: j.Attempt, Node: n.ID, Proof: n.Proofs[j.Key], Copy: copied,
		})
		n.pending = append(n.pending, i)
	}
	m.seatOne(i, n, t)
}

// schedule times the rows placed but not started at s.Now again, in queue
// order on each Node, whether the snapshot or the Step placed them.
func (m *sim) schedule() {
	m.ends = slices.Clone(m.fixed)
	slices.SortStableFunc(m.ends, func(a, b end) int { return a.t.Compare(b.t) })
	for _, n := range m.nodes {
		slices.Sort(n.pending)
		n.slots = slices.Clone(n.base)
		for _, i := range n.pending {
			m.seatOne(i, n, m.s.Now)
		}
	}
}

func (m *sim) seatOne(i int, n *nodeSim, t time.Time) {
	j := m.s.Jobs[i]
	asked, start, end := m.finish(n, j, t)
	m.copies[i] = !t.After(m.s.Now) && n.units[j.Source].arrives.After(m.s.Now)
	if known(start) {
		n.slots[n.free()] = end
		m.push(end, i)
	}
	m.byHours[i] = start.After(asked)
	m.plans[i] = Plan{Node: &n.ID, Start: at(start), Finish: at(end)}
}

// finish is when j would start and end on n from virtual time t: once a
// slot frees, its source is there and its download wait ends (asked), at
// the first minute n's hours allow.
func (m *sim) finish(n *nodeSim, j Job, t time.Time) (asked, start, end time.Time) {
	if n.stalled {
		return never, never, never
	}
	copied := t
	if u, ok := n.units[j.Source]; ok {
		copied = u.arrives
	} else if j.SourceNode != n.ID {
		copied = t.Add(m.copyTime(j, n))
	}
	asked = latest(n.slots[n.free()], copied, j.NotBefore)
	start = nextOpen(n.Hours, n.Zone, asked)
	return asked, start, start.Add(runtime(j, n.Proofs[j.Key]))
}

func (m *sim) copyTime(j Job, n *nodeSim) time.Duration {
	rate, ok := m.s.Rates[[2]role.NodeID{j.SourceNode, n.ID}]
	if !ok {
		rate = m.s.AssumedRate
	}
	return seconds(float64(j.SourceBytes) / rate)
}

// runtime scales a proof's real-time factor at 1080p by the output's
// pixel count.
func runtime(j Job, p Proof) time.Duration {
	return time.Duration(float64(j.Length) * float64(j.OutPixels) / (p.Speed * 1920 * 1080))
}

func seconds(s float64) time.Duration { return time.Duration(s * float64(time.Second)) }

func latest(t time.Time, ts ...time.Time) time.Time {
	for _, u := range ts {
		if u.After(t) {
			t = u
		}
	}
	return t
}

func at(t time.Time) *time.Time {
	if !known(t) {
		return nil
	}
	return &t
}

// nextOpen is the first time from t inside span's daily hours in zone; a
// nil span is any hour.
func nextOpen(span *setting.Span, zone *time.Location, t time.Time) time.Time {
	if span == nil || !known(t) {
		return t
	}
	local := t.In(zone)
	minute := local.Hour()*60 + local.Minute()
	if inside(*span, minute) {
		return t
	}
	opens := time.Date(local.Year(), local.Month(), local.Day(), span.Start/60, span.Start%60, 0, 0, zone)
	if !opens.After(t) {
		opens = time.Date(local.Year(), local.Month(), local.Day()+1, span.Start/60, span.Start%60, 0, 0, zone)
	}
	return opens
}

func inside(s setting.Span, minute int) bool {
	if s.Start < s.End {
		return minute >= s.Start && minute < s.End
	}
	return minute >= s.Start || minute < s.End
}

type end struct {
	t   time.Time
	job int
}

// ends is in time order.
type ends []end

func byTime(e end, t time.Time) int { return e.t.Compare(t) }

func (m *sim) push(t time.Time, job int) {
	k, _ := slices.BinarySearchFunc(m.ends, t, byTime)
	m.ends = slices.Insert(m.ends, k, end{t: t, job: job})
}

// next is the first virtual time from t that something can change: a run
// ends or a download wait does.
func (m *sim) next(t time.Time) (time.Time, bool) {
	now := t
	t, ok := never, false
	if len(m.ends) > 0 {
		t, ok = m.ends[0].t, true
	}
	for _, i := range m.waiting {
		if nb := m.s.Jobs[i].NotBefore; nb.After(now) && nb.Before(t) && !m.s.Jobs[i].Source.IsZero() {
			t, ok = nb, true
		}
	}
	return t, ok
}

// release ends every run due by t: its output leaves scratch, and its
// source once no job on the Node reads it.
func (m *sim) release(t time.Time) {
	for len(m.ends) > 0 && !m.ends[0].t.After(t) {
		i := m.ends[0].job
		m.ends = m.ends[1:]
		j := m.s.Jobs[i]
		n := m.byID[*m.plans[i].Node]
		if j.RootNode != n.ID {
			n.scratch += j.OutBytes
		}
		u := n.units[j.Source]
		u.jobs--
		if u.jobs == 0 {
			n.scratch += u.reserve
			delete(n.units, j.Source)
			m.held[j.Source]--
		}
	}
}

// tests is up to TestsAtOnce test runs per reachable holder, beside those
// in flight, of the keys it owes the due jobs.
func (m *sim) tests() []Test {
	due := m.dueJobs()
	var out []Test
	for _, n := range m.nodes {
		if n.Away {
			continue
		}
		keys := owed(n.Node, due)
		for _, k := range keys[:min(len(keys), max(0, m.s.TestsAtOnce-m.s.Tests[n.ID]))] {
			out = append(out, Test{Node: n.ID, Key: k})
		}
	}
	return out
}

func (m *sim) dueJobs() []Job {
	var out []Job
	for i, j := range m.s.Jobs {
		if j.Stage == queued && !m.placed[i] && due(j, m.s.Now) {
			out = append(out, j)
		}
	}
	return out
}

// causes is a Cause for each due queued job no Node qualifies for and none
// is testing. An away Node that qualifies counts, so a restart opens none.
func (m *sim) causes() []Cause {
	var out []Cause
	for i, j := range m.s.Jobs {
		if j.Stage != queued || m.placed[i] || !due(j, m.s.Now) || len(m.fits[i]) > 0 {
			continue
		}
		if w, n := m.closest(j); w != Untested && w != Retesting {
			out = append(out, Cause{Job: j.ID, Why: w, Node: n})
		}
	}
	return out
}

// closest is the Why of the Node nearest to qualifying for j, and that
// Node: NoHolder when none holds the Role, NoneOnline when every holder
// lapsed. A Node that would qualify after a test run answers Untested or
// Retesting.
func (m *sim) closest(j Job) (Why, *role.NodeID) {
	holders, online := 0, 0
	best, bestNode := Why(""), (*role.NodeID)(nil)
	for _, n := range m.nodes {
		if n.Holds {
			holders++
			if !n.Lapsed {
				online++
			}
		}
		w := Qualifies(n.Node, j)
		if provable(w) {
			return w, &n.ID
		}
		if best == "" || rank(w) < rank(best) {
			best, bestNode = w, &n.ID
		}
	}
	switch {
	case holders == 0:
		return NoHolder, nil
	case online == 0:
		return NoneOnline, nil
	case best == RunsElsewhere:
		return best, j.RunOn
	}
	return best, bestNode
}

func (m *sim) stalledWait(n *nodeSim) Wait {
	switch {
	case n.Away || n.Lapsed:
		return Wait{Code: Away, Node: &n.ID}
	case n.Hold:
		return Wait{Code: Transcode, Node: &n.ID}
	}
	return Wait{}
}

// forecast is each unfinished, unfailed job's Plan with its wait.
func (m *sim) forecast() Forecast {
	f := Forecast{}
	place := 0
	for i, j := range m.s.Jobs {
		switch j.Stage {
		case queued, copying, ready:
			place++
		default:
			if m.plans[i].Node != nil {
				f[j.ID] = m.plans[i]
			}
			continue
		}
		p := m.plans[i]
		p.Wait = m.wait(i, place)
		f[j.ID] = p
	}
	return f
}

func (m *sim) wait(i, place int) Wait {
	j := m.s.Jobs[i]
	p := m.plans[i]
	switch {
	case j.Source.IsZero():
		return Wait{Code: NoSource}
	case j.NotBefore.After(m.s.Now):
		return Wait{Code: Download, Until: &j.NotBefore}
	case p.Node != nil:
		return m.placedWait(i, place)
	case len(m.fits[i]) == 0:
		w, n := m.closest(j)
		if provable(w) {
			return Wait{Code: Testing, Node: n}
		}
		return Wait{Code: NoNode, Node: n, Cause: w}
	}
	for _, n := range m.fits[i] {
		if n.open {
			w := m.skip[i]
			if w.Code == InLine {
				w.Place = place
			}
			return w
		}
	}
	return Wait{Code: Away, Node: &m.fits[i][0].ID}
}

func (m *sim) placedWait(i, place int) Wait {
	n := m.byID[*m.plans[i].Node]
	if w := m.stalledWait(n); w.Code != "" {
		return w
	}
	if m.copies[i] {
		return Wait{Code: Copying, Node: &n.ID}
	}
	if m.byHours[i] {
		return Wait{Code: Hours, Node: &n.ID, Until: m.plans[i].Start}
	}
	return Wait{Code: InLine, Node: &n.ID, Place: place}
}
