package link_test

import (
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
		n.Path(worker6, controller6).Set(fault.Stall{})
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
		path.Set(fault.Stall{})
		if c := <-changes; c.State != (link.Down{}) {
			t.Fatalf("after the stall the Worker saw %+v", c)
		}
		down := time.Since(dials.start)
		path.Set(fault.None{})
		linktest.AwaitLive(t, changes, controller.Key)
		got := dials.seen()
		if want := []dial{{0, "127.0.0.2:8485"}, {35*time.Second + 760460373, "127.0.0.2:8485"}}; down != 35*time.Second || !slices.Equal(got, want) {
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

		want := []string{"link refused: the peer answered that key " + replacement.Fingerprint() + " superseded this Node"}
		if got := log.warnings(); !slices.Equal(got, want) {
			t.Fatalf("warnings %q, want %q", got, want)
		}
		if got := dials.seen(); !slices.Equal(got, []dial{
			{0, "127.0.0.3:8485"},
			{44919150123, "127.0.0.3:8485"},
			{103481912061, "127.0.0.3:8485"},
			{158508176558, "127.0.0.3:8485"},
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
		p.Net.Path(linktest.WorkerAddr, linktest.ControllerAddr).Set(fault.Stall{})

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
