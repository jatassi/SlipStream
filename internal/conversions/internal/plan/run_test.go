package plan

import (
	"math/rand/v2"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/encode"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
)

// TestFinish checks one job's forecast start and finish against the
// spec's formula: the queue ahead, the copy at the pair's rate, the length
// over the proof's speed at the output's pixels, the hours and the
// download wait.
func TestFinish(t *testing.T) {
	at := func(h, m, s int) time.Time {
		return now.Add(time.Duration(h)*time.Hour + time.Duration(m)*time.Minute + time.Duration(s)*time.Second)
	}
	behind := func(left time.Duration) func(*Snapshot) {
		return func(s *Snapshot) {
			r := job(9, attic)
			r.Stage, r.Node = running, &attic
			s.Jobs = append([]Job{r}, s.Jobs...)
			s.Nodes[0].Running = []Running{{Job: r.ID, Left: left}}
		}
	}
	cases := []struct {
		name          string
		edit          func(*Snapshot)
		start, finish time.Time
		wait          WaitCode
	}{
		{name: "the root on the Node at twice real time", edit: func(s *Snapshot) { s.Nodes[0].Proofs[hdrKey] = Proof{Speed: 2}; s.Jobs[0].Length = 2 * time.Hour },
			start: now, finish: at(1, 0, 0), wait: InLine},
		{name: "a 720p output", edit: func(s *Snapshot) { s.Jobs[0].OutPixels = 1280 * 720 },
			start: now, finish: at(0, 26, 40), wait: InLine},
		{name: "a copy at the measured rate", edit: func(s *Snapshot) {
			s.Jobs[0].SourceNode, s.Jobs[0].SourceBytes = cellar, 10_000_000_000
			s.Rates = map[[2]role.NodeID]float64{{cellar, attic}: 100_000_000}
		}, start: at(0, 1, 40), finish: at(1, 1, 40), wait: Copying},
		{name: "a copy at the assumed rate", edit: func(s *Snapshot) {
			s.Jobs[0].SourceNode, s.Jobs[0].SourceBytes = cellar, 10_000_000_000
			s.AssumedRate = 50_000_000
		}, start: at(0, 3, 20), finish: at(1, 3, 20), wait: Copying},
		{name: "behind a running Conversion", edit: behind(30 * time.Minute), start: at(0, 30, 0), finish: at(1, 30, 0), wait: InLine},
		{name: "beside a running Conversion with two at once", edit: func(s *Snapshot) {
			behind(30 * time.Minute)(s)
			s.Nodes[0].AtOnce = 2
		}, start: now, finish: at(1, 0, 0), wait: InLine},
		{name: "outside its hours", edit: func(s *Snapshot) { s.Nodes[0].Hours = &setting.Span{Start: 60, End: 420} },
			start: at(13, 0, 0), finish: at(14, 0, 0), wait: Hours},
		{name: "inside hours that cross midnight", edit: func(s *Snapshot) { s.Nodes[0].Hours = &setting.Span{Start: 11 * 60, End: 4 * 60} },
			start: now, finish: at(1, 0, 0), wait: InLine},
		{name: "a run ending past its hours", edit: func(s *Snapshot) {
			behind(2 * time.Hour)(s)
			s.Nodes[0].Hours = &setting.Span{Start: 11 * 60, End: 13 * 60}
		}, start: at(23, 0, 0), finish: at(24, 0, 0), wait: Hours},
		{name: "a download wait", edit: func(s *Snapshot) { s.Jobs[0].NotBefore = at(2, 0, 0) },
			start: at(2, 0, 0), finish: at(3, 0, 0), wait: Download},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			n := holder(attic)
			n.Zone = time.UTC
			s := Snapshot{Now: now, Nodes: []Node{n}, Jobs: []Job{job(1, attic)}}
			c.edit(&s)
			_, f := Run(s, now.Add(7*24*time.Hour))
			p := f[job(1, attic).ID]
			if p.Start == nil || p.Finish == nil || !p.Start.Equal(c.start) || !p.Finish.Equal(c.finish) || p.Wait.Code != c.wait {
				t.Fatalf("plan %v to %v waiting %s, want %v to %v waiting %s", p.Start, p.Finish, p.Wait.Code, c.start, c.finish, c.wait)
			}
		})
	}
}

// FuzzRun checks Run's properties over generated snapshots.
func FuzzRun(f *testing.F) {
	for seed := range uint64(300) {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, seed uint64) {
		g := gen{r: rand.New(rand.NewPCG(seed, 1))} //nolint:gosec // seeded on purpose so each seed replays
		s := g.snapshot()
		until := s.Now.Add(3 * 24 * time.Hour)
		step, forecast := Run(s, until)
		checkPlaced(t, s, step)
		again, forecast2 := Run(apply(s, step), until)
		if len(again.Placed) > 0 {
			t.Fatalf("a rerun on its own Step placed %d more", len(again.Placed))
		}
		for id, p := range forecast {
			if !reflect.DeepEqual(p, forecast2[id]) {
				t.Fatalf("job %v forecast %+v, after its Step %+v", id, p, forecast2[id])
			}
		}
		if len(forecast) != len(forecast2) {
			t.Fatalf("forecast %d jobs, after its Step %d", len(forecast), len(forecast2))
		}
	})
}

// checkPlaced holds that each placed job qualifies on its Node, goes to a
// Node already holding its source when one takes it, and opens no source
// past AtOnce+1 on a Node.
func checkPlaced(t *testing.T, s Snapshot, step Step) {
	t.Helper()
	nodes := map[role.NodeID]Node{}
	for _, n := range s.Nodes {
		nodes[n.ID] = n
	}
	jobs := map[storage.UUID]Job{}
	for _, j := range s.Jobs {
		jobs[j.ID] = j
	}
	held := sources(s)
	for _, p := range step.Placed {
		n, j := nodes[p.Node], jobs[p.Job]
		if w := Qualifies(n, j); w != Fits {
			t.Fatalf("placed %v on a Node that answers %s", j.ID, w)
		}
		for id, srcs := range held {
			if !held[p.Node][j.Source] && srcs[j.Source] && takes(nodes[id], j) {
				t.Fatalf("placed %v away from its source's Node, which takes it", j.ID)
			}
		}
		if !held[p.Node][j.Source] && len(held[p.Node]) > n.AtOnce {
			t.Fatalf("placed %v as a new source on a Node holding %d at %d at once", j.ID, len(held[p.Node]), n.AtOnce)
		}
		if held[p.Node] == nil {
			held[p.Node] = map[storage.UUID]bool{}
		}
		held[p.Node][j.Source] = true
	}
}

func takes(n Node, j Job) bool {
	return Qualifies(n, j) == Fits && !n.Away && n.May
}

// sources is the sources each Node holds for placed rows.
func sources(s Snapshot) map[role.NodeID]map[storage.UUID]bool {
	out := map[role.NodeID]map[storage.UUID]bool{}
	for _, j := range s.Jobs {
		if j.Node == nil || j.Stage == landing || j.Stage == "failed" {
			continue
		}
		if out[*j.Node] == nil {
			out[*j.Node] = map[storage.UUID]bool{}
		}
		out[*j.Node][j.Source] = true
	}
	return out
}

// apply is the reconciler's Write of a Step: each placed row copies or is
// ready on its Node, and reserves its room there.
func apply(s Snapshot, step Step) Snapshot {
	s.Jobs = slices.Clone(s.Jobs)
	s.Nodes = slices.Clone(s.Nodes)
	for _, p := range step.Placed {
		n := &s.Nodes[slices.IndexFunc(s.Nodes, func(n Node) bool { return n.ID == p.Node })]
		j := &s.Jobs[slices.IndexFunc(s.Jobs, func(j Job) bool { return j.ID == p.Job })]
		j.Stage, j.Node = ready, &p.Node
		if p.Copy {
			j.Stage = copying
			n.ScratchFree -= j.SourceBytes
		}
		if j.RootNode != p.Node {
			n.ScratchFree -= j.OutBytes
			continue
		}
		for r := range s.Jobs {
			if s.Jobs[r].Root == j.Root {
				s.Jobs[r].RootRoom -= j.OutBytes
			}
		}
	}
	return s
}

var keys = []encode.Key{
	base, hdrKey,
	{Shape: encode.Shape{Codec: encode.AV1, Depth: encode.Bit10, Range: encode.HDR10, Curve: encode.BT2390}, Class: encode.ClassDV81},
}

type gen struct {
	r *rand.Rand
	s Snapshot
	// on is the Node each source is placed on, units the sources placed
	// on each Node.
	on    map[int]int
	units map[int]map[int]bool
}

func (g *gen) chance(p float64) bool { return g.r.Float64() < p }
func (g *gen) gb(n int) int64        { return int64(g.r.IntN(n)+1) << 30 }
func (g *gen) nodeAt() int           { return g.r.IntN(len(g.s.Nodes)) }
func (g *gen) anyNode() role.NodeID  { return g.s.Nodes[g.nodeAt()].ID }

func (g *gen) snapshot() Snapshot {
	g.s = Snapshot{
		Now: now, AssumedRate: 50e6, TestsAtOnce: 1 + g.r.IntN(2),
		Rates: map[[2]role.NodeID]float64{}, Tests: map[role.NodeID]int{},
	}
	g.on, g.units = map[int]int{}, map[int]map[int]bool{}
	for b := range 1 + g.r.IntN(4) {
		n := g.node(b + 1)
		g.s.Nodes = append(g.s.Nodes, n)
		g.s.Tests[n.ID] = g.r.IntN(2)
	}
	for _, a := range g.s.Nodes {
		for _, b := range g.s.Nodes {
			if a.ID != b.ID && g.chance(0.5) {
				g.s.Rates[[2]role.NodeID{a.ID, b.ID}] = 1e6 + 1e8*g.r.Float64()
			}
		}
	}
	srcNodes, srcBytes := make([]role.NodeID, 1+g.r.IntN(9)), []int64{}
	for k := range srcNodes {
		srcNodes[k] = g.anyNode()
		srcBytes = append(srcBytes, g.gb(60))
	}
	roots, room := []role.NodeID{g.anyNode(), g.anyNode()}, []int64{g.gb(500), g.gb(500)}
	for k := range g.r.IntN(24) {
		si, ri := g.r.IntN(len(srcNodes)), g.r.IntN(2)
		j := g.job(k)
		j.Source, j.SourceNode, j.SourceBytes = uuid(200+si), srcNodes[si], srcBytes[si]
		j.Root, j.RootNode, j.RootRoom = uuid(50+ri), roots[ri], room[ri]
		if g.chance(0.08) {
			j.Source = storage.UUID{}
		} else if g.chance(0.35) {
			g.place(&j, si)
		}
		g.s.Jobs = append(g.s.Jobs, j)
	}
	g.running()
	return g.s
}

func (g *gen) node(id int) Node {
	n := Node{
		ID: nodeID(id), Holds: g.chance(0.85), Away: g.chance(0.15), Lapsed: g.chance(0.1), May: g.chance(0.9),
		AtOnce: 1 + g.r.IntN(3), Zone: time.UTC, Processor: g.chance(0.8), Hold: g.chance(0.1),
		ScratchFree: g.gb(400), Proofs: map[encode.Key]Proof{},
	}
	if start, end := g.r.IntN(24)*60, (g.r.IntN(23)+1)*60; g.chance(0.3) && start != end {
		n.Hours = &setting.Span{Start: start, End: end}
	}
	for _, k := range keys {
		if !g.chance(0.75) {
			continue
		}
		path := encode.Processor
		if g.chance(0.5) {
			path = encode.Hardware
		}
		n.Proofs[k] = Proof{Pipeline: encode.Pipeline{Path: path}, Speed: 0.5 + 4*g.r.Float64(), Failed: g.chance(0.1), Lost: g.chance(0.1)}
	}
	return n
}

func (g *gen) job(k int) Job {
	j := Job{
		ID: uuid(100 + k), Stage: queued, Attempt: g.r.IntN(3), Target: uuid(60 + g.r.IntN(3)),
		Key: keys[g.r.IntN(len(keys))], NotBefore: now, OutBytes: g.gb(20),
		Length: time.Duration(20+g.r.IntN(160)) * time.Minute, OutPixels: []int{1920 * 1080, 1280 * 720, 3840 * 2160}[g.r.IntN(3)],
	}
	if g.chance(0.15) {
		j.NotBefore = now.Add(time.Duration(g.r.IntN(48)) * time.Hour)
	}
	if g.chance(0.1) {
		j.Pin = ptr(g.anyNode())
	}
	if g.chance(0.1) {
		j.RunOn = ptr(g.anyNode())
	}
	if g.chance(0.1) {
		j.Exclude = map[role.NodeID]bool{g.anyNode(): true}
	}
	return j
}

// place puts j on its source's Node, or a new one with a free source unit.
func (g *gen) place(j *Job, si int) {
	at, ok := g.on[si]
	if !ok {
		at = g.nodeAt()
	}
	n := g.s.Nodes[at]
	if g.units[at] == nil {
		g.units[at] = map[int]bool{}
	}
	if !g.units[at][si] && len(g.units[at]) > n.AtOnce {
		return
	}
	g.units[at][si], g.on[si] = true, at
	j.Node, j.Pin = &n.ID, nil
	j.Stage = []string{copying, ready, running, paused, checking, returning, landing, "failed"}[g.r.IntN(8)]
}

// running gives most running rows their time left.
func (g *gen) running() {
	for k, n := range g.s.Nodes {
		for _, j := range g.s.Jobs {
			if j.Node != nil && *j.Node == n.ID && j.Stage == running && g.chance(0.8) {
				g.s.Nodes[k].Running = append(g.s.Nodes[k].Running, Running{Job: j.ID, Left: time.Duration(g.r.IntN(120)) * time.Minute})
			}
		}
	}
}
