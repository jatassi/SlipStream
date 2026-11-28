package conversions

import (
	"context"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/conversions/internal/db"
	"github.com/jatassi/slipstream/internal/converting"
	"github.com/jatassi/slipstream/internal/encode"
	"github.com/jatassi/slipstream/internal/history/entry/entrytest"
	"github.com/jatassi/slipstream/internal/mediaservers"
	"github.com/jatassi/slipstream/internal/nodes"
	cv1 "github.com/jatassi/slipstream/internal/pb/slipstream/converting/v1"
	"github.com/jatassi/slipstream/internal/pb/slipstream/converting/v1/convertingv1connect"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

// controller is the life world with a reconciler: a roster the test
// sets, each Node's facts, a Converting client that records the test
// runs it is asked, and Media Servers that transcode on cue.
type controller struct {
	*life
	mu      sync.Mutex
	holding []role.NodeID
	lapsed  map[role.NodeID]bool
	facts   map[role.NodeID]*cv1.Facts
	asked   []string
	playing map[role.NodeID]bool
}

var (
	sigA = strings.Repeat("a", 64)
	sigB = strings.Repeat("b", 64)
)

func newController(t *testing.T) *controller {
	t.Helper()
	w := &controller{life: newLife(t), lapsed: map[role.NodeID]bool{}, facts: map[role.NodeID]*cv1.Facts{}, playing: map[role.NodeID]bool{}}
	ctl := converting.Def.Serve(w.c.Derive, w.c.Apply)
	role.NewControls(role.ControlsConfig{Library: w.lib}, ctl)
	cfg := &w.c.cfg
	cfg.Control = func() *role.Control[*cv1.Intent, *cv1.Facts, *cv1.Result] { return ctl }
	cfg.Converting = func(n role.NodeID) convertingv1connect.ConvertingServiceClient { return tested{w, n} }
	cfg.TranscodingOn = func(_ context.Context, _ storage.Reader[storage.Library], n role.NodeID) ([]mediaservers.Transcoding, error) {
		w.mu.Lock()
		defer w.mu.Unlock()
		return []mediaservers.Transcoding{{Name: "Living room Plex", Known: true, Now: w.playing[n]}}, nil
	}
	w.c.roster = w.roster
	return w
}

// hold gives ns the Converting Role, each with a scratch volume and
// signature sigA.
func (w *controller) hold(ns ...role.NodeID) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.holding = ns
	for _, n := range ns {
		w.facts[n] = &cv1.Facts{Signature: sigA, Scratch: &cv1.Volume{Id: "v", Free: 1 << 50, Total: 1 << 50}}
	}
}

func (w *controller) roster() roster {
	return roster{holders: w, facts: func(n role.NodeID) (*cv1.Facts, bool) {
		w.mu.Lock()
		defer w.mu.Unlock()
		f, ok := w.facts[n]
		return f, ok
	}}
}

func (w *controller) Holds(n nodes.ID) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return slices.Contains(w.holding, n)
}
func (w *controller) May(n nodes.ID) bool    { return w.Holds(n) && !w.Lapsed(n) }
func (w *controller) Lapsed(n nodes.ID) bool { w.mu.Lock(); defer w.mu.Unlock(); return w.lapsed[n] }
func (w *controller) Away(n nodes.ID) (nodes.Absence, bool) {
	return nodes.Absence{Lapsed: true}, w.Lapsed(n)
}
func (w *controller) WrittenOff(nodes.ID) bool { return false }
func (w *controller) Serving() []nodes.ID {
	w.mu.Lock()
	defer w.mu.Unlock()
	return slices.Clone(w.holding)
}

func (w *controller) pass(t *testing.T) {
	t.Helper()
	if _, err := w.c.pass(t.Context(), w.roster()); err != nil {
		t.Fatal(err)
	}
	w.c.rec.tests.Wait()
}

// proveAt gives node a proof of the world's key under sig.
func (w *controller) proveAt(t *testing.T, node role.NodeID, sig string, pauses bool) {
	t.Helper()
	k := encode.Key{Shape: w.settings.Shape(), Class: encode.ClassOf(w.media)}
	w.write(t, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		speed := 4.0
		return db.New(tx).UpsertProof(ctx, db.UpsertProofParams{NodeID: node, Signature: sig, Shape: k.Shape.String(), Class: string(k.Class),
			Path: "hardware", Api: "qsv", Speed: &speed, Pauses: pauses, ProvenAt: storage.FromTime(w.now.Add(-time.Hour))})
	})
}

func (w *controller) proofs(t *testing.T, node role.NodeID) []string {
	t.Helper()
	var out []string
	err := w.lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		ps, err := db.New(r).ListProofsOf(ctx, node)
		for _, p := range ps {
			out = append(out, p.Signature)
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// tested is the Converting client of one Node: it records each test run
// it is asked and fails it.
type tested struct {
	w    *controller
	node role.NodeID
}

func (c tested) Test(_ context.Context, r *cv1.TestRequest) (*cv1.TestResponse, error) {
	c.w.mu.Lock()
	defer c.w.mu.Unlock()
	c.w.asked = append(c.w.asked, c.node.String()+" "+r.GetClass())
	return nil, context.Canceled
}

func TestPlacementPlacesAtMostAtOnceAndOneSourcesPerNode(t *testing.T) {
	for _, atOnce := range []int{1, 3} {
		w := newController(t)
		w.hold(w.attic, w.den)
		for _, n := range []role.NodeID{w.attic, w.den} {
			w.proveAt(t, n, sigA, true)
			if _, err := w.c.SetNodeSettings(t.Context(), n, NodeSettings{AtOnce: &atOnce, Processor: true}); err != nil {
				t.Fatal(err)
			}
		}
		var rows []Row
		for range 12 {
			rows = append(rows, w.seed(t, seed{Stage: StageQueued}))
		}
		w.pass(t)
		w.pass(t)
		on := map[role.NodeID]int{}
		for _, r := range rows {
			if got := w.read(t, r.ID); got.Node != nil {
				on[*got.Node]++
			}
		}
		for _, n := range []role.NodeID{w.attic, w.den} {
			if on[n] != atOnce+1 {
				t.Errorf("at once %d: %d placed on a Node, want %d", atOnce, on[n], atOnce+1)
			}
		}
	}
}

func TestACauseOpensOnceForManyRowsAndAgainOnlyAfterItCloses(t *testing.T) {
	w := newController(t)
	opened := func() int { return len(entrytest.Of(w.history, NoNodeCanConvert)) }
	for range 3 {
		w.seed(t, seed{Stage: StageQueued})
	}
	steps := []struct {
		name string
		do   func()
		want int
	}{
		{"no Node has the Role", func() {}, 1},
		{"it stays open", func() { w.seed(t, seed{Stage: StageQueued}) }, 1},
		{"a Node takes the Role and the cause closes", func() { w.hold(w.attic); w.proveAt(t, w.attic, sigA, true) }, 1},
		{"the Role goes and it opens again", func() { w.hold(); w.seed(t, seed{Stage: StageQueued}) }, 2},
	}
	for _, s := range steps {
		s.do()
		w.pass(t)
		if got := opened(); got != s.want {
			t.Fatalf("%s: No Node can convert recorded %d times, want %d", s.name, got, s.want)
		}
	}
}

func TestASignatureChangeDeletesOldProofsAndAsksAgain(t *testing.T) {
	w := newController(t)
	w.hold(w.attic)
	w.proveAt(t, w.attic, sigA, true)
	w.seed(t, seed{Stage: StageQueued})
	for _, sig := range []string{sigA, sigB} {
		if _, err := w.c.observe(t.Context(), w.attic, &cv1.Facts{Signature: sig}); err != nil {
			t.Fatal(err)
		}
	}
	w.facts[w.attic].Signature = sigB
	if got := w.proofs(t, w.attic); len(got) != 0 {
		t.Fatalf("proofs after the signature changed = %v, want none", got)
	}
	w.pass(t)
	if len(w.asked) == 0 || !strings.HasPrefix(w.asked[0], w.attic.String()) {
		t.Errorf("test runs asked = %v, want one of Attic", w.asked)
	}
}

func TestDeriveSetsHoldAndKeep(t *testing.T) {
	cases := []struct {
		name    string
		playing bool
		pauses  bool
		hold    bool
	}{
		{"a transcode holds a Node whose runs pause", true, true, true},
		{"no transcode", false, true, false},
		{"a run that can't pause isn't asked to", true, false, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w := newController(t)
			w.proveAt(t, w.den, sigA, c.pauses)
			w.playing[w.den] = c.playing
			running := w.seed(t, seed{Stage: StageRunning, Node: &w.den, Attempt: 1})
			ready := w.seed(t, seed{Stage: StageReady, Node: &w.den, Attempt: 1})
			var in *cv1.Intent
			err := w.lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
				var err error
				in, err = w.c.Derive(ctx, r, w.den)
				return err
			})
			if err != nil {
				t.Fatal(err)
			}
			if in.GetHold() != c.hold {
				t.Errorf("hold = %v, want %v", in.GetHold(), c.hold)
			}
			var want []string
			for _, r := range []Row{running, ready} {
				want = append(want, "in/s-"+r.SourceFile.String()+".mkv", "out/c-"+r.ID.String()+"-1.mkv")
			}
			if got := in.GetKeep(); !sameSet(got, want) {
				t.Errorf("keep = %v, want %v", got, want)
			}
			if len(in.GetJobs()) != 2 {
				t.Errorf("jobs = %d, want 2", len(in.GetJobs()))
			}
		})
	}
}

func sameSet(a, b []string) bool {
	a, b = slices.Clone(a), slices.Clone(b)
	slices.Sort(a)
	slices.Sort(b)
	return slices.Equal(a, b)
}

func TestATranscodePutsBackARunThatCantPause(t *testing.T) {
	w := newController(t)
	w.hold(w.den)
	w.proveAt(t, w.den, sigA, false)
	w.playing[w.den] = true
	r := w.seed(t, seed{Stage: StageRunning, Node: &w.den, Attempt: 1})
	w.pass(t)
	if got := w.read(t, r.ID); got.Stage == StageRunning || got.Stage == StagePaused {
		t.Errorf("after the transcode the Conversion is %s, want put back", got.Stage)
	}
	if a := w.attempts(t, r.ID); a[0].Outcome != AttemptPutBack {
		t.Errorf("the running attempt's outcome = %q, want %q", a[0].Outcome, AttemptPutBack)
	}
}
