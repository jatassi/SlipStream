package link_test

import (
	"bytes"
	"context"
	"log/slog"
	"net"
	"net/netip"
	"slices"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/internal/link"
	"github.com/jatassi/slipstream/internal/link/linktest"
	linkv1 "github.com/jatassi/slipstream/internal/pb/slipstream/link/v1"
)

type dial struct {
	at   time.Duration
	addr string
}

// recorder notes when each dial starts, counted from its creation.
type recorder struct {
	link.Network
	start time.Time
	mu    sync.Mutex
	dials []dial
}

func (r *recorder) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	r.mu.Lock()
	r.dials = append(r.dials, dial{time.Since(r.start), address})
	r.mu.Unlock()
	return r.Network.DialContext(ctx, network, address)
}

func (r *recorder) seen() []dial {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.dials)
}

var (
	controller6 = netip.MustParseAddr("fd00::2")
	worker6     = netip.MustParseAddr("fd00::3")
)

// dualStack is a Controller and a Worker with both families, the Worker
// dialing through a recorder.
func dualStack(t *testing.T) (n *fault.Network, controller, worker *linktest.Node, dials *recorder) {
	n = fault.NewNetwork()
	dials = &recorder{start: time.Now()}
	controller = linktest.Start(t, n, "controller", []netip.Addr{controller6, linktest.ControllerAddr}, linktest.Side{})
	worker = linktest.Start(t, n, "worker", []netip.Addr{worker6, linktest.WorkerAddr}, linktest.Side{
		Net: func(h *fault.Host) link.Network { dials.Network = h; return dials },
	})
	controller.Trust.Set(worker.Key, link.Pinned{})
	worker.Trust.Set(controller.Key, link.Pinned{})
	return n, controller, worker, dials
}

func TestStalledIPv6FallsBackToIPv4AfterFallbackDelay(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		n, controller, worker, dials := dualStack(t)
		if _, err := n.Path(worker6, controller6).Set(fault.Rule{Fault: fault.Stall{}}); err != nil {
			t.Fatal(err)
		}
		changes := worker.Subscribe(t.Context())
		worker.Keep(controller.Target())
		live := linktest.AwaitLive(t, changes, controller.Key)
		want := []dial{{0, "[fd00::2]:8485"}, {300 * time.Millisecond, "127.0.0.2:8485"}}
		if got := dials.seen(); !slices.Equal(got, want) {
			t.Fatalf("dials %v, want %v", got, want)
		}
		if live.Remote != netip.MustParseAddrPort("127.0.0.2:8485") {
			t.Fatalf("the session's remote is %v", live.Remote)
		}
	})
}

func TestAnsweringIPv6MakesNoIPv4Dial(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		_, controller, worker, dials := dualStack(t)
		changes := worker.Subscribe(t.Context())
		worker.Keep(controller.Target())
		live := linktest.AwaitLive(t, changes, controller.Key)
		time.Sleep(time.Second)
		if got, want := dials.seen(), []dial{{0, "[fd00::2]:8485"}}; !slices.Equal(got, want) {
			t.Fatalf("dials %v, want %v", got, want)
		}
		if live.Remote != netip.MustParseAddrPort("[fd00::2]:8485") {
			t.Fatalf("the session's remote is %v", live.Remote)
		}
	})
}

// After a drop the Worker re-dials once the first draw from its link
// stream has passed, and a Welcome resets the attempt count.
func TestRedialAfterADrop(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		dials := &recorder{start: time.Now()}
		n := fault.NewNetwork()
		controller := linktest.Start(t, n, "controller", []netip.Addr{linktest.ControllerAddr}, linktest.Side{})
		worker := linktest.Start(t, n, "worker", []netip.Addr{linktest.WorkerAddr}, linktest.Side{
			Net: func(h *fault.Host) link.Network { dials.Network = h; return dials },
		})
		controller.Trust.Set(worker.Key, link.Pinned{})
		worker.Trust.Set(controller.Key, link.Pinned{})
		changes := worker.Subscribe(t.Context())
		worker.Keep(controller.Target())
		linktest.AwaitLive(t, changes, controller.Key)

		path := n.Path(linktest.WorkerAddr, linktest.ControllerAddr)
		if _, err := path.Set(fault.Rule{Fault: fault.Stall{}}); err != nil {
			t.Fatal(err)
		}
		if c := <-changes; c.State != (link.Down{}) {
			t.Fatalf("after the stall the Worker saw %+v", c)
		}
		down := time.Since(dials.start)
		path.Clear("")
		linktest.AwaitLive(t, changes, controller.Key)
		got := dials.seen()
		if want := []dial{{0, "127.0.0.2:8485"}, {35*time.Second + 580305161, "127.0.0.2:8485"}}; down != 35*time.Second || !slices.Equal(got, want) {
			t.Fatalf("down at %v, dials %v", down, got)
		}
	})
}

// recordLog keeps every record's message and error.
type recordLog struct {
	mu   sync.Mutex
	logs []string
}

func (r *recordLog) Enabled(context.Context, slog.Level) bool { return true }
func (r *recordLog) WithAttrs([]slog.Attr) slog.Handler       { return r }
func (r *recordLog) WithGroup(string) slog.Handler            { return r }
func (r *recordLog) Handle(_ context.Context, rec slog.Record) error {
	line := rec.Message
	rec.Attrs(func(a slog.Attr) bool {
		if a.Key == "err" {
			line += ": " + a.Value.String()
		}
		return true
	})
	r.mu.Lock()
	r.logs = append(r.logs, line)
	r.mu.Unlock()
	return nil
}

func (r *recordLog) lines() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.logs)
}

func (r *recordLog) warnings() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.DeleteFunc(slices.Clone(r.logs), func(s string) bool { return s[:len("link refused")] != "link refused" })
}

// A superseded Controller that keeps dialing its old Worker gets Hello
// answered with the new key, backs off at redial_max and logs it once.
func TestSupersededHelloBacksOffAtRedialMaxAndLogsOnce(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		log := &recordLog{}
		dials := &recorder{start: time.Now()}
		n := fault.NewNetwork()
		replacement := link.NewPrivateKey([32]byte{7}).Public()
		worker := linktest.Start(t, n, "worker", []netip.Addr{linktest.WorkerAddr}, linktest.Side{})
		old := linktest.Start(t, n, "controller", []netip.Addr{linktest.ControllerAddr}, linktest.Side{
			Log: slog.New(log),
			Net: func(h *fault.Host) link.Network { dials.Network = h; return dials },
		})
		worker.Trust.Set(old.Key, link.Superseded{By: replacement, Addresses: addresses(t, "127.0.0.9:8485")})
		old.Trust.Set(worker.Key, link.Pinned{})
		old.Keep(worker.Target())
		time.Sleep(3 * time.Minute)

		want := []string{"link refused: the peer reported that key " + replacement.Fingerprint() + " superseded this Node"}
		if got := log.warnings(); !slices.Equal(got, want) {
			t.Fatalf("warnings %q, want %q", got, want)
		}
		if got := dials.seen(); !slices.Equal(got, []dial{
			{0, "127.0.0.3:8485"},
			{52304516711, "127.0.0.3:8485"},
			{101256348671, "127.0.0.3:8485"},
			{153521574840, "127.0.0.3:8485"},
		}) {
			t.Fatalf("dials %v", got)
		}
		if got := old.State(worker.Key); got != (link.Down{}) {
			t.Fatalf("the superseded Controller sees %+v", got)
		}
	})
}

// A Worker that comes back under the same key while the Controller still
// holds its old, half-open session replaces it: the Controller goes from
// the old session to the new one without a Down between.
func TestNewerSessionClosesTheOlder(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		p := linktest.New(t, linktest.Side{}, linktest.Side{})
		changes := p.Controller.Subscribe(t.Context())
		p.Connect(t)
		if _, err := p.Net.Path(linktest.WorkerAddr, linktest.ControllerAddr).Set(fault.Rule{Fault: fault.Stall{}}); err != nil {
			t.Fatal(err)
		}

		again := linktest.Start(t, p.Net, "worker", []netip.Addr{netip.MustParseAddr("127.0.0.4")}, linktest.Side{})
		again.Trust.Set(p.Controller.Key, link.Pinned{})
		again.Keep(p.Controller.Target())
		first, next := <-changes, <-changes
		synctest.Wait()

		old, ok := first.State.(link.Live)
		second, ok2 := next.State.(link.Live)
		if !ok || !ok2 || old.Remote != netip.MustParseAddrPort("127.0.0.3:49152") || second.Remote != netip.MustParseAddrPort("127.0.0.4:49153") {
			t.Fatalf("the Controller saw %+v then %+v", first, next)
		}
		if got := p.Controller.State(p.Worker.Key); got != second {
			t.Fatalf("the Controller settled on %+v", got)
		}
		select {
		case c := <-changes:
			t.Fatalf("after the new session the Controller saw %+v", c)
		default:
		}
	})
}

// Keeping a Target whose key the peer does not hold dials, finds another
// key and stays Down, logging why at Debug.
func TestKeepRefusesAPeerHoldingAnotherKey(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		log := &recordLog{}
		n := fault.NewNetwork()
		controller := linktest.Start(t, n, "controller", []netip.Addr{linktest.ControllerAddr}, linktest.Side{})
		worker := linktest.Start(t, n, "worker", []netip.Addr{linktest.WorkerAddr}, linktest.Side{Log: slog.New(log)})
		controller.Trust.Set(worker.Key, link.Pinned{})
		pinned := link.NewPrivateKey([32]byte{9}).Public()
		worker.Keep(link.Target{Key: pinned, Addresses: controller.Target().Addresses})
		synctest.Wait()
		want := []string{"link dial failed: handshake with 127.0.0.2:8485: peer key " + controller.Key.Fingerprint() + " is not the pinned key " + pinned.Fingerprint()}
		if got := log.lines(); !slices.Equal(got, want) {
			t.Fatalf("the Worker logged %q, want %q", got, want)
		}
		if got := worker.State(pinned); got != (link.Down{}) {
			t.Fatalf("the Worker sees %+v for the pinned key, want Down", got)
		}
	})
}

// drain is every Change queued on changes, once the bubble is idle.
func drain(changes <-chan link.Change) []link.Change {
	var got []link.Change
	for {
		synctest.Wait()
		select {
		case c := <-changes:
			got = append(got, c)
		default:
			return got
		}
	}
}

// When each side keeps the other, both dial at once, and over a 20 ms path
// each side welcomes the two sessions in its own order. Both keep the
// session the lower key dialed and close the other, so neither dials
// again: they settle on that session and stay on it. The lower key's side
// may first welcome the other session and see it end before its own
// dial's Welcome arrives, which is one Down, not a redial. The race runs
// eight times, since the order of goroutines at one instant varies.
func TestBothSidesKeepOneSession(t *testing.T) {
	for range 8 {
		bothSidesKeep(t)
	}
}

func bothSidesKeep(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		atC, atW := &recorder{start: time.Now()}, &recorder{start: time.Now()}
		p := linktest.New(t,
			linktest.Side{Net: func(h *fault.Host) link.Network { atC.Network = h; return atC }},
			linktest.Side{Net: func(h *fault.Host) link.Network { atW.Network = h; return atW }},
		)
		for _, path := range []*fault.Plan{p.Net.Path(linktest.WorkerAddr, linktest.ControllerAddr), p.Net.Path(linktest.ControllerAddr, linktest.WorkerAddr)} {
			if _, err := path.Set(fault.Rule{Fault: fault.Latency(20 * time.Millisecond)}); err != nil {
				t.Fatal(err)
			}
		}
		atController, atWorker := p.Controller.Subscribe(t.Context()), p.Worker.Subscribe(t.Context())
		p.Controller.Keep(p.Worker.Target())
		p.Worker.Keep(p.Controller.Target())
		time.Sleep(time.Minute)
		if c, w := len(atC.seen()), len(atW.seen()); c != 1 || w != 1 {
			t.Fatalf("the Controller dialed %d times and the Worker %d, want once each", c, w)
		}

		c, cok := p.Controller.State(p.Worker.Key).(link.Live)
		w, wok := p.Worker.State(p.Controller.Key).(link.Live)
		if !cok || !wok || c.Local != w.Remote || c.Remote != w.Local {
			t.Fatalf("the Controller sees %+v and the Worker %+v, want one live session", p.Controller.State(p.Worker.Key), p.Worker.State(p.Controller.Key))
		}
		workerLower := bytes.Compare(p.Worker.Key[:], p.Controller.Key[:]) < 0
		if w.Dialed != workerLower || c.Dialed == w.Dialed {
			t.Fatalf("the Worker dialed the live session: %v, the Controller: %v; want the lower key's side, and the Worker's key is lower: %v", w.Dialed, c.Dialed, workerLower)
		}

		drain(atController)
		drain(atWorker)
		time.Sleep(3 * time.Minute)
		if got := [2][]link.Change{drain(atController), drain(atWorker)}; len(got[0])+len(got[1]) > 0 {
			t.Fatalf("after settling the Controller saw %+v and the Worker %+v", got[0], got[1])
		}
		if p.Controller.State(p.Worker.Key) != c || p.Worker.State(p.Controller.Key) != w {
			t.Fatal("the session changed without a Change")
		}
	})
}

// Reset closes the live session and leaves the keeper, so the Worker
// dials again, whichever side resets.
func TestResetRedialsAKeptPeer(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		p := linktest.New(t, linktest.Side{}, linktest.Side{})
		changes := p.Worker.Subscribe(t.Context())
		p.Worker.Keep(p.Controller.Target())
		first := linktest.AwaitLive(t, changes, p.Controller.Key)
		for _, reset := range []func(){
			func() { p.Controller.Reset(p.Worker.Key) },
			func() { p.Worker.Reset(p.Controller.Key) },
		} {
			reset()
			if c := <-changes; c.State != (link.Down{}) {
				t.Fatalf("after Reset the Worker saw %+v, want Down", c)
			}
			again := linktest.AwaitLive(t, changes, p.Controller.Key)
			if again.Local == first.Local {
				t.Fatalf("the Worker came back on %v, the session Reset closed", again.Local)
			}
			first = again
		}
	})
}

// A kept peer that refuses Hello leaves the key Down with the reason, in
// one Change however often the keeper dials again. A Welcome clears it,
// and so does Drop.
func TestRefusedHelloIsDownWithTheReason(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		p := linktest.New(t, linktest.Side{}, linktest.Side{})
		p.Controller.Trust.Set(p.Worker.Key, link.Pairing{})
		changes := p.Worker.Subscribe(t.Context())
		p.Worker.Keep(p.Controller.Target())
		time.Sleep(3 * time.Minute)
		refused := link.Down{Refused: linkv1.RefusedReason_REFUSED_REASON_UNKNOWN_KEY}
		if got := drain(changes); !slices.Equal(got, []link.Change{{Key: p.Controller.Key, State: refused}}) {
			t.Fatalf("the Worker saw %+v, want one Change to %+v", got, refused)
		}
		if got := p.Worker.State(p.Controller.Key); got != refused {
			t.Fatalf("the Worker sees %+v, want %+v", got, refused)
		}

		p.Controller.Trust.Set(p.Worker.Key, link.Pinned{})
		linktest.AwaitLive(t, changes, p.Controller.Key)
		p.Controller.Reset(p.Worker.Key)
		if c := <-changes; c.State != (link.Down{}) {
			t.Fatalf("after a Welcome the session's end is %+v, want Down with no reason", c)
		}

		p.Controller.Trust.Set(p.Worker.Key, link.Pairing{})
		p.Worker.Reset(p.Controller.Key)
		time.Sleep(3 * time.Minute)
		if got := p.Worker.State(p.Controller.Key); got != refused {
			t.Fatalf("refused again, the Worker sees %+v", got)
		}
		drain(changes)
		p.Worker.Drop(p.Controller.Key)
		if got := drain(changes); !slices.Equal(got, []link.Change{{Key: p.Controller.Key, State: link.Down{}}}) {
			t.Fatalf("after Drop the Worker saw %+v, want one Change to Down", got)
		}
	})
}
