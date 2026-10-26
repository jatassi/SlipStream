package transfers_test

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/link"
	"github.com/jatassi/slipstream/internal/link/linktest"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/nodes"
	tv1 "github.com/jatassi/slipstream/internal/pb/slipstream/transfer/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/role/roletest"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/transfers"
	"github.com/jatassi/slipstream/internal/transfers/source"
	"github.com/jatassi/slipstream/internal/works"
)

var discard = slog.New(slog.DiscardHandler)

type control = role.Control[*tv1.Intent, *tv1.Facts, *tv1.Result]

// cluster is a Controller, home, that receives into its own root, and a
// Worker, seedbox, that sends, each holding the transfers channel.
type cluster struct {
	pair          *linktest.Pair
	lib           *storage.DB[storage.Library]
	controls      *role.Controls
	tr            *transfers.Transfers
	home, seedbox role.NodeID
	root          roots.ID
	names         map[role.NodeID]string
	purpose       *purpose
	homeEnd       *nodeEnd
	seedboxEnd    *nodeEnd
	published     *published
	// stop stops the Controller's loop.
	stop func()

	mu      sync.Mutex
	touched []role.NodeID
	// paused, while set, holds the loop at its next wake, before it judges.
	paused chan struct{}
	held   bool
}

func (r *cluster) Self() role.NodeID { return r.home }

func (r *cluster) NodeOf(key link.Key) (role.NodeID, bool) {
	switch key {
	case r.pair.Controller.Key:
		return r.home, true
	case r.pair.Worker.Key:
		return r.seedbox, true
	}
	return role.NodeID{}, false
}

type board map[role.NodeID]transfers.End

func (b board) End(node role.NodeID) (transfers.End, bool) {
	e, ok := b[node]
	return e, ok
}

func (board) Changed() <-chan struct{} { return nil }

type published struct {
	mu  sync.Mutex
	evs []transfers.Changed
}

func (p *published) Publish(_ context.Context, ev transfers.Changed) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.evs = append(p.evs, ev)
}

func (p *published) all() []transfers.Changed {
	p.mu.Lock()
	defer p.mu.Unlock()
	return slices.Clone(p.evs)
}

// purpose records each hook it sees.
type purpose struct {
	mu       sync.Mutex
	verified []string
	failed   []string
	after    []string
}

func (p *purpose) register(tr *transfers.Transfers) {
	tr.Register(transfers.Purpose{
		Kind: transfers.Grab,
		Verified: func(_ context.Context, _ *storage.Tx[storage.Library], t transfers.Transfer, group string) error {
			p.mu.Lock()
			defer p.mu.Unlock()
			p.verified = append(p.verified, t.Job+" "+group)
			return nil
		},
		Failed: func(_ context.Context, _ *storage.Tx[storage.Library], t transfers.Transfer, f transfers.Failure) error {
			p.mu.Lock()
			defer p.mu.Unlock()
			p.failed = append(p.failed, t.Job+": "+f.Sentence)
			return nil
		},
		After: func(jobs []string) {
			p.mu.Lock()
			defer p.mu.Unlock()
			p.after = append(p.after, jobs...)
		},
		Names: func(_ context.Context, _ storage.Reader[storage.Library], jobs []string) (map[string]string, error) {
			out := map[string]string{}
			for _, j := range jobs {
				out[j] = "the " + j + " job"
			}
			return out, nil
		},
	})
}

func (p *purpose) seen() (verified, failed, after []string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return slices.Clone(p.verified), slices.Clone(p.failed), slices.Clone(p.after)
}

// nodeEnd is one Node's transfers holder: it keeps its last intent, and the
// test sets its facts and reports its Results.
type nodeEnd struct {
	local *storage.DB[storage.Local]

	mu     sync.Mutex
	holder role.Holder[*tv1.Intent, *tv1.Facts, *tv1.Result]
	intent *tv1.Intent
}

func (e *nodeEnd) run(ctx context.Context, h role.Holder[*tv1.Intent, *tv1.Facts, *tv1.Result]) error {
	e.mu.Lock()
	e.holder = h
	e.mu.Unlock()
	for s := range h.Intent(ctx) {
		e.mu.Lock()
		e.intent = s.Intent
		e.mu.Unlock()
	}
	return nil
}

func (e *nodeEnd) held() role.Held {
	return transfers.Def.Hold(e.run, func(context.Context) error { return nil })
}

func (e *nodeEnd) now() (*tv1.Intent, role.Holder[*tv1.Intent, *tv1.Facts, *tv1.Result]) {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.intent, e.holder
}

// actives is the Transfers it runs, "send 0 1 2" naming the files.
func (e *nodeEnd) actives() []string {
	in, _ := e.now()
	var out []string
	for _, a := range in.GetActive() {
		words := []string{map[tv1.Side]string{tv1.Side_SIDE_SEND: "send", tv1.Side_SIDE_RECEIVE: "receive"}[a.GetSide()]}
		for _, f := range a.GetFiles() {
			words = append(words, strconv.FormatUint(uint64(f.GetPosition()), 10))
		}
		out = append(out, strings.Join(words, " "))
	}
	return out
}

func (e *nodeEnd) report(t *testing.T, r *tv1.Result) {
	t.Helper()
	_, h := e.now()
	if err := e.local.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Local]) error {
		return h.Report(ctx, tx, r)
	}); err != nil {
		t.Fatal(err)
	}
	synctest.Wait()
}

func (e *nodeEnd) facts(f *tv1.Facts) {
	_, h := e.now()
	h.SetFacts(f)
	synctest.Wait()
}

func newCluster(t *testing.T) *cluster {
	t.Helper()
	lockstep := http.NewServeMux()
	r := &cluster{
		pair:       linktest.New(t, linktest.Side{Lockstep: lockstep}, linktest.Side{}),
		lib:        storagetest.Library(t, nodes.Schema(), role.LibrarySchema(), works.Schema(), roots.Schema(), health.Schema(), transfers.LibrarySchema()),
		home:       role.NodeID{UUID: storage.NewUUID()},
		seedbox:    role.NodeID{UUID: storage.NewUUID()},
		root:       roots.ID{UUID: storage.NewUUID()},
		purpose:    &purpose{},
		homeEnd:    &nodeEnd{local: storagetest.Local(t, role.LocalSchema())},
		seedboxEnd: &nodeEnd{local: storagetest.Local(t, role.LocalSchema())},
		published:  &published{},
	}
	r.names = map[role.NodeID]string{r.home: "home", r.seedbox: "seedbox"}
	r.exec(t, "INSERT INTO nodes_node (id, public_key, name, paired_at) VALUES (?, ?, 'home', 0)", r.home, r.pair.Controller.Key[:])
	r.exec(t, "INSERT INTO nodes_node (id, public_key, name, paired_at) VALUES (?, ?, 'seedbox', 0)", r.seedbox, r.pair.Worker.Key[:])
	r.exec(t, "INSERT INTO roots_root (id, content_type, node_id, path, is_default) VALUES (?, 'alpha', ?, '/media', 1)", r.root, r.home)
	ends := board{r.home: {Controller: true}, r.seedbox: {}}
	volume := storing.Volume{ID: "v1", Name: "Media", Mount: "/media", Total: 4 << 40, Free: 2 << 40, SampledAt: time.Now()}
	var c *control
	r.tr = transfers.New(transfers.ControllerConfig{
		Library: r.lib,
		Touch: func(ids ...role.NodeID) {
			r.mu.Lock()
			r.touched = append(r.touched, ids...)
			r.mu.Unlock()
			c.Touch(ids...)
		},
		Facts: func(id role.NodeID) (*tv1.Facts, bool) { return c.Facts(id) },
		Board: func() transfers.Board { return ends },
		Destinations: func(ctx context.Context, rd storage.Reader[storage.Library], _ []roots.ID) (map[roots.ID]roots.Destination, error) {
			all, err := roots.All(ctx, rd)
			out := map[roots.ID]roots.Destination{}
			for _, root := range all {
				out[root.ID] = roots.Destination{Root: root, State: roots.Ready, Volume: &volume}
			}
			return out, err
		},
		RootsMoved: func() <-chan struct{} {
			r.mu.Lock()
			paused := r.paused
			r.held = paused != nil
			r.mu.Unlock()
			if paused != nil {
				<-paused
			}
			return nil
		},
		Readiness: func(_ context.Context, _ storage.Reader[storage.Library], _ role.NodeID, refs []source.Ref) (map[source.Ref]source.Readiness, error) {
			out := map[source.Ref]source.Readiness{}
			for _, ref := range refs {
				out[ref] = source.Readiness{State: source.Ready}
			}
			return out, nil
		},
		Moved:   func() <-chan struct{} { return nil },
		Sample:  func(context.Context, role.NodeID) error { return nil },
		Hub:     live.NewHub([]live.Event{roots.ChangedEvent, roots.VolumesChangedEvent, health.ChangedEvent, nodes.ChangedEvent, transfers.ChangedEvent}),
		Changed: r.published,
		Timings: transfers.DefaultTimings(),
		Log:     discard,
	})
	r.purpose.register(r.tr)
	c = transfers.Def.Serve(r.tr.Derive, r.tr.Apply)
	r.controls = role.NewControls(role.ControlsConfig{
		Library: r.lib,
		Nodes:   r,
		Scope: func(context.Context, storage.Reader[storage.Library], role.NodeID) (role.NodeScope, error) {
			return role.NodeScope{}, nil
		},
		Facts:   r.tr.Noted,
		Applied: r.tr.Applied,
		Stall:   45 * time.Second,
		Log:     discard,
	}, c)
	lockstep.Handle(r.controls.Handler())
	r.hold(t, r.homeEnd, role.InProcess(r.controls))
	r.hold(t, r.seedboxEnd, role.OverLink(r.pair.Worker.Link, r.pair.Controller.Key, linktest.Timings))
	r.pair.Connect(t)
	r.stop = r.run(t)
	synctest.Wait()
	return r
}

// run starts the Controller's loop and returns what stops it.
func (r *cluster) run(t *testing.T) (stop func()) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- r.tr.Run(ctx) }()
	stop = sync.OnceFunc(func() {
		cancel()
		if err := <-done; err != nil {
			t.Error(err)
		}
	})
	t.Cleanup(stop)
	return stop
}

// pause holds the loop at its next wake, before it judges, until resume.
func (r *cluster) pause() (resume func(t *testing.T)) {
	paused := make(chan struct{})
	r.mu.Lock()
	r.paused = paused
	r.mu.Unlock()
	return func(t *testing.T) {
		t.Helper()
		r.mu.Lock()
		held := r.held
		r.paused, r.held = nil, false
		r.mu.Unlock()
		if !held {
			t.Fatal("the loop never woke while paused")
		}
		close(paused)
	}
}

func (r *cluster) exec(t *testing.T, query string, args ...any) {
	t.Helper()
	if err := r.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := tx.ExecContext(ctx, query, args...)
		return err
	}); err != nil {
		t.Fatal(err)
	}
}

func (r *cluster) hold(t *testing.T, e *nodeEnd, upstream role.Upstream) {
	t.Helper()
	h, err := role.NewHolding(t.Context(), role.HoldingConfig{
		Local:    e.local,
		Upstream: upstream,
		Held:     []role.Held{e.held()},
		Timings:  role.Timings{RestartMin: time.Second, RestartMax: time.Minute, Beat: 15 * time.Second},
		Rand:     rand.New(rand.NewPCG(1, 2)), //nolint:gosec // seeded on purpose so tests replay
		Log:      discard,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = h.Run(ctx)
	}()
	t.Cleanup(func() {
		cancel()
		<-done
	})
}

// start asks for job's Transfer of one file per group from seedbox into
// home's root, as a purpose does: Start in its Write, Touch after.
func (r *cluster) start(t *testing.T, job string, groups ...string) transfers.ID {
	t.Helper()
	req := transfers.Request{Purpose: transfers.Grab, Job: job, Source: r.seedbox, Destination: transfers.Root{ID: r.root}}
	for i, g := range groups {
		var h bencode.InfoHash
		h[0] = byte(i + 1)
		req.Files = append(req.Files, transfers.FileRequest{Group: g, Name: fmt.Sprintf("%s %d.mkv", job, i), Source: source.Torrent{Hash: h}, Size: 1 << 20})
	}
	var id transfers.ID
	r.write(t, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		var err error
		id, err = r.tr.Start(ctx, tx, req)
		return err
	})
	return id
}

// write runs fn in one Write, then Touches the loop and waits for it.
func (r *cluster) write(t *testing.T, fn func(ctx context.Context, tx *storage.Tx[storage.Library]) error) {
	t.Helper()
	if err := r.lib.Write(t.Context(), fn); err != nil {
		t.Fatal(err)
	}
	r.tr.Touch()
	synctest.Wait()
	r.fresh(t)
}

func (r *cluster) fresh(t *testing.T) {
	t.Helper()
	roletest.CheckFresh(t, r.controls, r.names)
}

func (r *cluster) takeTouched() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []string
	for _, n := range r.touched {
		out = append(out, r.names[n])
	}
	r.touched = nil
	slices.Sort(out)
	return out
}

func (r *cluster) view(t *testing.T, id transfers.ID) transfers.View {
	t.Helper()
	v, err := r.tr.Get(t.Context(), id)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func verifiedResult(id transfers.ID, position uint32, place int64) *tv1.Result {
	return &tv1.Result{Result: &tv1.Result_Verified{Verified: &tv1.FileVerified{TransferId: id.Bytes(), Position: position, Place: place}}}
}

func failed(id transfers.ID, code tv1.Failed_Code, detail string) *tv1.Result {
	return &tv1.Result{Result: &tv1.Result_Failed{Failed: &tv1.Failed{TransferId: id.Bytes(), Code: code, Detail: detail}}}
}

func check(t *testing.T, what string, got, want []string) {
	t.Helper()
	if !slices.Equal(got, want) {
		t.Errorf("%s = %q, want %q", what, got, want)
	}
}

func TestStartIsIdempotentOnPurposeAndJob(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newCluster(t)
		first := r.start(t, "heat", "")
		if again := r.start(t, "heat", ""); again != first {
			t.Errorf("starting the job again recorded %s, want its Transfer %s", again, first)
		}
		if other := r.start(t, "ronin", ""); other == first {
			t.Error("another job shares the first job's Transfer")
		}
		var found transfers.Transfer
		var ok bool
		err := r.lib.Read(t.Context(), func(ctx context.Context, rd storage.Reader[storage.Library]) error {
			var err error
			found, ok, err = r.tr.Find(ctx, rd, transfers.Grab, "heat")
			return err
		})
		if err != nil || !ok || found.ID != first || found.Source != r.seedbox || found.Destination != (transfers.Root{ID: r.root}) {
			t.Errorf("Find = %+v, %v, %v; want the first Transfer", found, ok, err)
		}
		check(t, "seedbox runs", r.seedboxEnd.actives(), []string{"send 0"})
		check(t, "home runs", r.homeEnd.actives(), []string{"receive 0"})
		if v := r.view(t, first); v.State != transfers.Waiting || v.Wait.Kind != transfers.Connecting {
			t.Errorf("the first Transfer reads %v, want connecting", v.State)
		}
	})
}

func TestVerifiedFiresOncePerGroupInTheOrderGroupsComplete(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newCluster(t)
		id := r.start(t, "heat", "a", "b", "a")
		r.homeEnd.report(t, verifiedResult(id, 1, 1))
		r.homeEnd.report(t, verifiedResult(id, 0, 1))
		r.homeEnd.report(t, verifiedResult(id, 1, 1))
		r.fresh(t)
		check(t, "home runs", r.homeEnd.actives(), []string{"receive 2"})
		r.homeEnd.report(t, verifiedResult(id, 2, 1))
		r.fresh(t)
		v, _, after := r.purpose.seen()
		check(t, "Verified", v, []string{"heat b", "heat a"})
		check(t, "After", after, []string{"heat", "heat"})
		check(t, "home runs", r.homeEnd.actives(), nil)
		if got := r.view(t, id); got.State != transfers.Copying || got.Verified != got.Total {
			t.Errorf("a Transfer verified but not placed reads %v at %d of %d, want copying at its full size", got.State, got.Verified, got.Total)
		}
	})
}

func TestPlacedRemovesTheRecordWithItsLastFile(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newCluster(t)
		id := r.start(t, "heat", "", "")
		r.homeEnd.report(t, verifiedResult(id, 0, 1))
		r.homeEnd.report(t, verifiedResult(id, 1, 1))
		placed := func(position int) {
			r.write(t, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
				return r.tr.Placed(ctx, tx, id, position)
			})
		}
		placed(0)
		if _, err := r.tr.Get(t.Context(), id); err != nil {
			t.Fatalf("after placing one of two files the Transfer reads %v, want it kept", err)
		}
		in, _ := r.homeEnd.now()
		if len(in.GetKeep()) != 1 {
			t.Errorf("home keeps %x, want the Transfer's staging", in.GetKeep())
		}
		placed(1)
		if _, err := r.tr.Get(t.Context(), id); !errors.Is(err, transfers.ErrNotFound) {
			t.Errorf("after placing its last file the Transfer reads %v, want it gone", err)
		}
		if in, _ := r.homeEnd.now(); len(in.GetKeep()) != 0 {
			t.Errorf("home keeps %x, want nothing", in.GetKeep())
		}
	})
}

func TestCancelDeletesTheRecordAndBothEndsLetGo(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newCluster(t)
		id := r.start(t, "heat", "")
		r.write(t, func(ctx context.Context, tx *storage.Tx[storage.Library]) error { return r.tr.Cancel(ctx, tx, id) })
		check(t, "seedbox runs", r.seedboxEnd.actives(), nil)
		check(t, "home runs", r.homeEnd.actives(), nil)
		if in, _ := r.homeEnd.now(); len(in.GetKeep()) != 0 {
			t.Errorf("home keeps %x, want nothing", in.GetKeep())
		}
	})
}

// TestALoopStartedAfterACancelLetsGo cancels a Transfer while the loop is
// down, as when the Controller restarts after its Nodes' sessions derived
// their intents: the loop's first pass lets both ends go.
func TestALoopStartedAfterACancelLetsGo(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newCluster(t)
		id := r.start(t, "heat", "")
		r.stop()
		if err := r.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error { return r.tr.Cancel(ctx, tx, id) }); err != nil {
			t.Fatal(err)
		}
		r.tr.Touch()
		r.run(t)
		synctest.Wait()
		check(t, "seedbox runs", r.seedboxEnd.actives(), nil)
		if in, _ := r.homeEnd.now(); len(in.GetKeep()) != 0 {
			t.Errorf("home keeps %x, want nothing", in.GetKeep())
		}
		r.fresh(t)
	})
}

// TestANodeThatDerivesBetweenPassesLetsGo starts and cancels a Transfer
// while the loop is busy, and seedbox's session derives in between: the
// loop's next judgment matches its last, yet seedbox lets go.
func TestANodeThatDerivesBetweenPassesLetsGo(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newCluster(t)
		resume := r.pause()
		var id transfers.ID
		req := transfers.Request{Purpose: transfers.Grab, Job: "heat", Source: r.seedbox, Destination: transfers.Root{ID: r.root}, Files: []transfers.FileRequest{{Name: "heat.mkv", Source: source.Torrent{Hash: bencode.InfoHash{1}}, Size: 1 << 20}}}
		if err := r.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
			var err error
			id, err = r.tr.Start(ctx, tx, req)
			return err
		}); err != nil {
			t.Fatal(err)
		}
		r.tr.Touch()
		r.controls.Touch(r.seedbox)
		synctest.Wait()
		check(t, "seedbox runs after its session derived", r.seedboxEnd.actives(), []string{"send 0"})
		if err := r.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error { return r.tr.Cancel(ctx, tx, id) }); err != nil {
			t.Fatal(err)
		}
		r.tr.Touch()
		resume(t)
		synctest.Wait()
		check(t, "seedbox runs", r.seedboxEnd.actives(), nil)
		r.fresh(t)
	})
}

func TestFailedFiresOnceWithItsSentence(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newCluster(t)
		id := r.start(t, "heat", "")
		next := r.start(t, "ronin", "")
		r.homeEnd.report(t, failed(id, tv1.Failed_CODE_SOURCE_FAILED, "file not found"))
		r.homeEnd.report(t, failed(id, tv1.Failed_CODE_DAMAGED, ""))
		r.fresh(t)
		_, f, after := r.purpose.seen()
		check(t, "Failed", f, []string{"heat: seedbox's torrent client failed heat 0.mkv: file not found."})
		check(t, "After", after, []string{"heat"})
		if v := r.view(t, id); v.State != transfers.Failed || v.Failure.Code != transfers.SourceFailed {
			t.Errorf("the failed Transfer reads %v, want failed source_failed", v.State)
		}
		if v := r.view(t, next); v.State != transfers.Waiting || v.Wait.Kind != transfers.Connecting {
			t.Errorf("the next on the pair reads %v, want it active", v.State)
		}
	})
}

func TestAStaleOrForeignResultAppliesAsNothing(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newCluster(t)
		id := r.start(t, "heat", "")
		gone := r.start(t, "ronin", "")
		r.write(t, func(ctx context.Context, tx *storage.Tx[storage.Library]) error { return r.tr.Cancel(ctx, tx, gone) })
		r.seedboxEnd.report(t, verifiedResult(id, 0, 1))
		r.seedboxEnd.report(t, failed(id, tv1.Failed_CODE_SOURCE_GONE, ""))
		r.homeEnd.report(t, verifiedResult(id, 0, 2))
		r.homeEnd.report(t, verifiedResult(id, 5, 1))
		r.homeEnd.report(t, verifiedResult(gone, 0, 1))
		r.homeEnd.report(t, failed(gone, tv1.Failed_CODE_DAMAGED, ""))
		r.homeEnd.report(t, &tv1.Result{Result: &tv1.Result_Verified{Verified: &tv1.FileVerified{TransferId: []byte("not an id"), Place: 1}}})
		v, f, after := r.purpose.seen()
		if len(v)+len(f)+len(after) != 0 {
			t.Fatalf("stale Results fired Verified %q, Failed %q, After %q; want nothing", v, f, after)
		}
		r.homeEnd.report(t, verifiedResult(id, 0, 1))
		v, _, _ = r.purpose.seen()
		check(t, "Verified after the stale Results", v, []string{"heat "})
		r.fresh(t)
	})
}

func TestOnlyNodesWhoseIntentMovedAreTouched(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newCluster(t)
		r.takeTouched()
		id := r.start(t, "heat", "")
		check(t, "starting a Transfer touched", r.takeTouched(), []string{"home", "seedbox"})
		r.homeEnd.facts(&tv1.Facts{Receiving: []*tv1.Receiving{{TransferId: id.Bytes(), Route: tv1.Route_ROUTE_RECEIVER_DIALS, Active: true, VerifiedBytes: 1 << 10}}})
		r.fresh(t)
		check(t, "facts that move no intent touched", r.takeTouched(), nil)
		limit := int64(5 << 20)
		if _, err := r.tr.SetNodeSettings(t.Context(), r.seedbox, transfers.NodeChange{SendLimit: parsed(transfers.SendLimit.Parse(&limit))}); err != nil {
			t.Fatal(err)
		}
		synctest.Wait()
		r.fresh(t)
		check(t, "a send limit touched", r.takeTouched(), []string{"seedbox"})
	})
}

func TestSendLimitRoundTrips(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newCluster(t)
		if o, err := r.tr.NodeSettings(t.Context(), r.seedbox); err != nil || o.SendLimit != nil {
			t.Fatalf("an unset send limit reads %v, %v; want no override", o.SendLimit, err)
		}
		limit := int64(5 << 20)
		o, err := r.tr.SetNodeSettings(t.Context(), r.seedbox, transfers.NodeChange{SendLimit: parsed(transfers.SendLimit.Parse(&limit))})
		if err != nil || o.SendLimit == nil || *o.SendLimit != limit {
			t.Fatalf("setting the send limit = %v, %v; want %d", o.SendLimit, err, limit)
		}
		synctest.Wait()
		if read, err := r.tr.NodeSettings(t.Context(), r.seedbox); err != nil || *read.SendLimit != limit {
			t.Errorf("the stored send limit reads %v, %v; want %d", read.SendLimit, err, limit)
		}
		if in, _ := r.seedboxEnd.now(); in.GetSendLimit() != limit {
			t.Errorf("seedbox's intent limits sending to %d, want %d", in.GetSendLimit(), limit)
		}
		evs := r.published.all()
		if len(evs) == 0 || !slices.Contains(evs[len(evs)-1].NodeIDs, r.seedbox.String()) {
			t.Errorf("transfers.changed = %+v, want seedbox named", evs)
		}
		if o, err := r.tr.SetNodeSettings(t.Context(), r.seedbox, transfers.NodeChange{SendLimit: parsed(transfers.SendLimit.Parse(nil))}); err != nil || o.SendLimit != nil {
			t.Errorf("resetting the send limit = %v, %v; want no override", o.SendLimit, err)
		}
		synctest.Wait()
		if in, _ := r.seedboxEnd.now(); in.GetSendLimit() != 0 {
			t.Errorf("seedbox's intent limits sending to %d after a reset, want none", in.GetSendLimit())
		}
		r.fresh(t)
	})
}

func TestBytesMovingOnARouteRecordThePairAtOnce(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newCluster(t)
		id := r.start(t, "heat", "")
		r.homeEnd.facts(&tv1.Facts{Receiving: []*tv1.Receiving{{TransferId: id.Bytes(), Route: tv1.Route_ROUTE_RECEIVER_DIALS, Active: true, Moving: true, Rate: 3 << 20}}})
		r.fresh(t)
		pairs, err := r.tr.Speeds(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		if len(pairs) != 1 || pairs[0].Source != r.seedbox || pairs[0].Destination != r.home || pairs[0].Rate != 3<<20 || pairs[0].Route != tv1.Route_ROUTE_RECEIVER_DIALS {
			t.Errorf("pairs = %+v, want seedbox to home at 3 MB/s on route 1", pairs)
		}
		if v := r.view(t, id); v.State != transfers.Copying || v.Route != tv1.Route_ROUTE_RECEIVER_DIALS {
			t.Errorf("the Transfer reads %v on %v, want copying on route 1", v.State, v.Route)
		}
		evs := r.published.all()
		if len(evs) == 0 || !slices.Contains(evs[len(evs)-1].TransferIDs, id.String()) {
			t.Errorf("transfers.changed = %+v, want the Transfer named", evs)
		}
	})
}

func parsed(s setting.Stored[int64], err error) *setting.Stored[int64] {
	if err != nil {
		panic(err)
	}
	return &s
}
