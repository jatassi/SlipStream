package plan

import (
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/encode"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

func uuid(n int) storage.UUID {
	u, err := storage.ParseUUID(fmt.Sprintf("%08x-0000-4000-8000-000000000000", n))
	if err != nil {
		panic(err)
	}
	return u
}

func nodeID(n int) role.NodeID { return role.NodeID{UUID: uuid(n)} }

var (
	attic, cellar = nodeID(1), nodeID(2)
	now           = time.Date(2026, 11, 28, 12, 0, 0, 0, time.UTC)
	hdrKey        = encode.Key{
		Shape: encode.Shape{Codec: encode.HEVC, Depth: encode.Bit10, Range: encode.SDR, Curve: encode.BT2390},
		Class: encode.ClassHDR10,
	}
	onProcessor = Proof{Pipeline: encode.Pipeline{Path: encode.Processor}, Speed: 1}
)

func holder(id role.NodeID) Node {
	return Node{
		ID: id, Holds: true, May: true, AtOnce: 1, Processor: true, ScratchFree: 1 << 40,
		Proofs: map[encode.Key]Proof{base: onProcessor, hdrKey: onProcessor},
	}
}

func job(id int, on role.NodeID) Job {
	return Job{
		ID: uuid(100 + id), Stage: queued, Key: hdrKey, NotBefore: now,
		Source: uuid(200 + id), SourceNode: on, SourceBytes: 1 << 30, OutBytes: 1 << 29,
		Root: uuid(50), RootNode: on, RootRoom: 1 << 40, Length: time.Hour, OutPixels: 1920 * 1080,
	}
}

func ptr[T any](v T) *T { return &v }

// TestQualifies drives each clause of the qualifying rule, and what Run
// makes of it for a lone queued job: Fits places it, Untested and
// Retesting ask a test run, and every other answer is a cause.
func TestQualifies(t *testing.T) {
	lost := Proof{Pipeline: encode.Pipeline{Path: encode.Processor}, Speed: 1, Lost: true}
	cases := []struct {
		name   string
		node   func(*Node)
		job    func(*Job)
		others []Node
		want   Why
		step   Why
	}{
		{name: "fits", want: Fits, step: Fits},
		{name: "no role", node: func(n *Node) { n.Holds = false }, want: NoRole, step: NoHolder},
		{name: "lapsed", node: func(n *Node) { n.Lapsed = true }, want: Lapsed, step: NoneOnline},
		{name: "run on another Node", job: func(j *Job) { j.RunOn = &cellar }, want: RunsElsewhere, step: RunsElsewhere},
		{name: "pinned to another Node", job: func(j *Job) { j.Pin = &cellar }, want: PinnedElsewhere, step: PinnedElsewhere},
		{name: "failed here", job: func(j *Job) { j.Exclude = map[role.NodeID]bool{attic: true} }, want: FailedHere, step: FailedHere},
		{name: "untested", node: func(n *Node) { delete(n.Proofs, hdrKey) }, want: Untested, step: Untested},
		{name: "test failed", node: func(n *Node) { n.Proofs[hdrKey] = Proof{Failed: true} }, want: TestFailed, step: TestFailed},
		{
			name: "pinned retry of a lost proof", node: func(n *Node) { n.Proofs[hdrKey] = lost },
			job: func(j *Job) { j.Pin = &attic }, want: Retesting, step: Retesting,
		},
		{name: "proof lost", node: func(n *Node) { n.Proofs[hdrKey] = lost }, want: ProofLost, step: ProofLost},
		{name: "processor off", node: func(n *Node) { n.Processor = false }, want: ProcessorOff, step: ProcessorOff},
		{
			name: "processor off is closer than a failed test", node: func(n *Node) { n.Proofs[hdrKey] = Proof{Failed: true} },
			others: []Node{func() Node { n := holder(cellar); n.Processor = false; return n }()},
			want:   TestFailed, step: ProcessorOff,
		},
		{
			name: "a lapsed Run on Node is the cause beside an online holder", node: func(n *Node) { n.Lapsed = true },
			job:    func(j *Job) { j.RunOn = &attic },
			others: []Node{holder(cellar)},
			want:   Lapsed, step: RunsElsewhere,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			n, j := holder(attic), job(1, attic)
			if c.node != nil {
				c.node(&n)
			}
			if c.job != nil {
				c.job(&j)
			}
			if got := Qualifies(n, j); got != c.want {
				t.Fatalf("Qualifies = %s, want %s", got, c.want)
			}
			step, _ := Run(Snapshot{Now: now, Nodes: append([]Node{n}, c.others...), Jobs: []Job{j}, TestsAtOnce: 1}, now)
			if got := stepWhy(step, j); got != c.step {
				t.Fatalf("Run reads %s, want %s (step %+v)", got, c.step, step)
			}
		})
	}
}

func stepWhy(s Step, j Job) Why {
	switch {
	case len(s.Placed) == 1 && s.Placed[0].Job == j.ID:
		return Fits
	case len(s.Causes) == 1 && s.Causes[0].Job == j.ID:
		return s.Causes[0].Why
	case len(s.Causes) == 0 && slices.ContainsFunc(s.Tests, func(t Test) bool { return t.Key == j.Key }):
		if j.Pin != nil {
			return Retesting
		}
		return Untested
	}
	return ""
}

func TestOwed(t *testing.T) {
	lost := Proof{Pipeline: encode.Pipeline{Path: encode.Processor}, Speed: 1, Lost: true}
	pinned := job(1, attic)
	pinned.Pin = &attic
	cases := []struct {
		name string
		node func(*Node)
		jobs []Job
		want []encode.Key
	}{
		{name: "proven owes nothing", jobs: []Job{job(1, attic)}},
		{name: "the base key while it has no proof row", node: func(n *Node) { delete(n.Proofs, base) }, want: []encode.Key{base}},
		{name: "a failed base test is a proof row", node: func(n *Node) { n.Proofs[base] = Proof{Failed: true} }},
		{
			name: "an untested key a job needs, once", node: func(n *Node) { delete(n.Proofs, hdrKey) },
			jobs: []Job{job(1, attic), job(2, attic)}, want: []encode.Key{hdrKey},
		},
		{name: "a pinned retry's lost key", node: func(n *Node) { n.Proofs[hdrKey] = lost }, jobs: []Job{pinned}, want: []encode.Key{hdrKey}},
		{name: "a lost key with no pinned retry", node: func(n *Node) { n.Proofs[hdrKey] = lost }, jobs: []Job{job(1, attic)}},
		{
			name: "a lost key of a retry pinned elsewhere", node: func(n *Node) { n.Proofs[hdrKey] = lost },
			jobs: []Job{func() Job { j := pinned; j.Pin = &cellar; return j }()},
		},
		{
			name: "a key failed here", node: func(n *Node) { delete(n.Proofs, hdrKey) },
			jobs: []Job{func() Job { j := job(1, attic); j.Exclude = map[role.NodeID]bool{attic: true}; return j }()},
		},
		{name: "a failed test", node: func(n *Node) { n.Proofs[hdrKey] = Proof{Failed: true} }, jobs: []Job{job(1, attic)}},
		{name: "a lapsed Node", node: func(n *Node) { n.Lapsed = true; n.Proofs = nil }, jobs: []Job{job(1, attic)}},
		{name: "a Node without the Role", node: func(n *Node) { n.Holds = false; n.Proofs = nil }, jobs: []Job{job(1, attic)}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			n := holder(attic)
			if c.node != nil {
				c.node(&n)
			}
			if got := owed(n, c.jobs); !slices.Equal(got, c.want) {
				t.Fatalf("owed = %v, want %v", got, c.want)
			}
		})
	}
}
