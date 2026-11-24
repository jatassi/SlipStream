package netfacts_test

import (
	"context"
	"fmt"
	"iter"
	"log/slog"
	"net"
	"net/netip"
	"reflect"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/fake/faketest"
	"github.com/jatassi/slipstream/fakes/scenario"
	fakestun "github.com/jatassi/slipstream/fakes/stun"
	"github.com/jatassi/slipstream/internal/netfacts"
	"github.com/jatassi/slipstream/internal/netmodel"
	"github.com/jatassi/slipstream/internal/node/endpoint"
	netv1 "github.com/jatassi/slipstream/internal/pb/slipstream/network/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

// holder is the network channel's holder half as Watch sees it: the
// intents a test sends, and the facts Watch publishes.
type holder struct {
	intents chan *netv1.Intent
	mu      sync.Mutex
	facts   []*netv1.Facts
	set     chan struct{}
}

func newHolder() *holder {
	return &holder{intents: make(chan *netv1.Intent), set: make(chan struct{}, 16)}
}

func (h *holder) Intent(ctx context.Context) iter.Seq[role.Snapshot[*netv1.Intent]] {
	return func(yield func(role.Snapshot[*netv1.Intent]) bool) {
		for {
			select {
			case i := <-h.intents:
				if !yield(role.Snapshot[*netv1.Intent]{Intent: i, Standing: role.Live}) {
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}
}

func (h *holder) SetFacts(f *netv1.Facts) {
	h.mu.Lock()
	h.facts = append(h.facts, f)
	h.mu.Unlock()
	h.set <- struct{}{}
}

func (h *holder) Report(context.Context, *storage.Tx[storage.Local], *emptypb.Empty) error {
	return nil
}

func (h *holder) published() []*netv1.Facts {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]*netv1.Facts(nil), h.facts...)
}

// watch runs n's watch until the test ends.
func watch(t *testing.T, n *netfacts.Netfacts) *holder {
	t.Helper()
	h := newHolder()
	ctx, cancel := context.WithCancel(context.WithoutCancel(t.Context()))
	done := make(chan error, 1)
	go func() { done <- n.Watch(ctx, h) }()
	t.Cleanup(func() {
		cancel()
		if err := <-done; err != nil {
			t.Errorf("watch: %v", err)
		}
	})
	return h
}

// awaitFacts waits for the watch's next publish.
func awaitFacts(t *testing.T, h *holder) {
	t.Helper()
	select {
	case <-h.set:
	case <-t.Context().Done():
		t.Fatal("the watch published nothing")
	}
}

type loopback struct{}

func (loopback) DialUDP(_ context.Context, raddr netip.AddrPort) (*net.UDPConn, error) {
	c, err := net.DialUDP("udp", nil, net.UDPAddrFromAddrPort(raddr))
	if err != nil {
		return nil, fmt.Errorf("open a UDP socket to %s: %w", raddr, err)
	}
	return c, nil
}

func (loopback) ListenUDP(_ context.Context, local netip.AddrPort) (*net.UDPConn, error) {
	c, err := net.ListenUDP("udp", net.UDPAddrFromAddrPort(local))
	if err != nil {
		return nil, fmt.Errorf("open a UDP socket on %s: %w", local, err)
	}
	return c, nil
}

var home = []netmodel.Iface{{Name: "eth0", Index: 2, Addrs: []netmodel.IfaceAddr{{Prefix: netip.MustParsePrefix("192.168.1.20/24")}}}}

// env is a machine with home's interfaces, no router and no firewall
// reader, whose sockets are loopback's.
func env() netfacts.Env {
	return netfacts.Env{
		Interfaces: func(context.Context) ([]netmodel.Iface, error) { return home, nil },
		Route: func(context.Context, netip.Addr) (netfacts.Route, error) {
			return netfacts.Route{Iface: "eth0"}, nil
		},
		Sockets: func(context.Context, []uint16) ([]netmodel.Listener, error) { return nil, nil },
		Firewall: func(_ context.Context, _ string, qs []netfacts.Query) ([]netmodel.Rule, error) {
			return make([]netmodel.Rule, len(qs)), nil
		},
		Packets: loopback{},
		Self:    func() []netmodel.Listener { return nil },
	}
}

func timings() netfacts.Timings {
	t := netfacts.DefaultTimings()
	t.STUN, t.Gateway, t.ConfirmFloor = 200*time.Millisecond, 200*time.Millisecond, 10*time.Millisecond
	return t
}

func stunSites() scenario.Scenario {
	return scenario.Scenario{
		Name: "stun",
		Nodes: []scenario.Node{
			{Name: "atlas", Setup: scenario.SetupController, Site: "home"},
			{Name: "den", Setup: scenario.SetupJoin, Site: "cell"},
		},
		Network: &scenario.Network{Sites: []scenario.Site{
			{Name: "home", WAN: "public", Address: "203.0.113.9"},
			{Name: "cell", WAN: "cgnat", Address: "203.0.113.50", Gateway: "100.64.12.7", STUN: "timeout"},
		}},
	}
}

// stunServers serves fake-stun's two servers and answers each origin's
// endpoints.network.stun and the capture both record into.
func stunServers(t *testing.T) (func(origin string) netfacts.Endpoints, faketest.Served) {
	t.Helper()
	s, shared := stunSites(), fake.NewShared()
	origins := faketest.WithOrigins("atlas", "den")
	first := faketest.UDP(t, fakestun.Fake, s, origins, faketest.WithShared(shared))
	second := faketest.UDP(t, fakestun.Second, s, origins, faketest.WithShared(shared))
	return func(origin string) netfacts.Endpoints {
		return netfacts.Endpoints{STUN: endpoint.Optional{
			endpoint.MustParseHostPort(first.Sockets[origin].String()),
			endpoint.MustParseHostPort(second.Sockets[origin].String()),
		}}
	}, first
}

func TestAWatchedReadAsksEachSTUNServer(t *testing.T) {
	endpoints, _ := stunServers(t)
	for _, c := range []struct {
		origin string
		want   []netip.AddrPort
		err    netmodel.STUNError
		class  netmodel.Class4
	}{
		{"atlas", []netip.AddrPort{netip.MustParseAddrPort("203.0.113.9:50000"), netip.MustParseAddrPort("203.0.113.9:50000")}, netmodel.STUNNone, netmodel.BehindRouter},
		{"den", []netip.AddrPort{{}, {}}, netmodel.STUNTimeout, netmodel.Class4Unknown},
	} {
		t.Run(c.origin, func(t *testing.T) {
			e := endpoints(c.origin)
			n := netfacts.New(netfacts.Config{Env: env(), Endpoints: e, Timings: timings(), Rules: netmodel.Rules{DocumentationGlobal: true}, Log: slog.New(slog.DiscardHandler)})
			h := watch(t, n)
			h.intents <- &netv1.Intent{Watch: true, IntervalMs: time.Hour.Milliseconds()}
			awaitFacts(t, h)
			f, err := n.Facts(t.Context(), nil, time.Hour, false)
			if err != nil {
				t.Fatal(err)
			}
			var want []netmodel.STUNAnswer
			for i, m := range c.want {
				want = append(want, netmodel.STUNAnswer{Server: e.STUN[i].String(), Mapped: m, Err: c.err})
			}
			if !f.Watched || !reflect.DeepEqual(f.STUN, want) {
				t.Errorf("STUN answers %+v (watched %t), want %+v", f.STUN, f.Watched, want)
			}
			s, err := netmodel.ParseSummary(h.published()[0])
			if err != nil || s.Class != c.class {
				t.Errorf("published class %v, %v; want %v", s.Class, err, c.class)
			}
		})
	}
}

// Reads share the last pass while it is younger than max_age; a local
// Read, or any Read once the intent stops the watch, sends nothing.
func TestReadsShareAPassAndALocalReadSendsNothing(t *testing.T) {
	endpoints, stun := stunServers(t)
	n := netfacts.New(netfacts.Config{Env: env(), Endpoints: endpoints("atlas"), Timings: timings(), Rules: netmodel.Rules{DocumentationGlobal: true}, Log: slog.New(slog.DiscardHandler)})
	h := watch(t, n)
	h.intents <- &netv1.Intent{Watch: true, IntervalMs: time.Hour.Milliseconds()}
	awaitFacts(t, h)
	sent := len(stun.Records(t))
	for _, c := range []struct {
		name    string
		maxAge  time.Duration
		local   bool
		stop    bool
		sends   int
		watched bool
	}{
		{"a Read inside max_age", time.Hour, false, false, 0, true},
		{"a Read with no max_age", 0, false, false, 2, true},
		{"a local Read", 0, true, false, 0, false},
		{"a Read once the watch stops", 0, false, true, 0, false},
	} {
		if c.stop {
			h.intents <- &netv1.Intent{}
			awaitFacts(t, h)
		}
		f, err := n.Facts(t.Context(), []uint16{8484}, c.maxAge, c.local)
		got := len(stun.Records(t)) - sent
		sent += got
		if err != nil || got != c.sends || f.Watched != c.watched {
			t.Errorf("%s sent %d STUN requests and read watched %t, %v; want %d and %t", c.name, got, f.Watched, err, c.sends, c.watched)
		}
	}
	if last := h.published(); last[len(last)-1].GetWatched() {
		t.Error("the watch stopped without publishing that it doesn't watch")
	}
}

// The watch publishes a change only once it held for the wait, never a
// flap inside it, and passes no more once told not to watch.
func TestTheWatchPublishesAChangeThatHeld(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var mu sync.Mutex
		prefix, reads := "2001:db8:1::20/64", 0
		e := env()
		e.Interfaces = func(context.Context) ([]netmodel.Iface, error) {
			mu.Lock()
			defer mu.Unlock()
			reads++
			return []netmodel.Iface{{Name: "eth0", Addrs: []netmodel.IfaceAddr{{Prefix: netip.MustParsePrefix(prefix)}}}}, nil
		}
		set := func(p string) {
			mu.Lock()
			prefix = netip.MustParsePrefix(p).Addr().Next().String() + "/64"
			mu.Unlock()
		}
		epoch := time.Now()
		n := netfacts.New(netfacts.Config{Env: e, Timings: netfacts.DefaultTimings(), Rules: netmodel.Rules{DocumentationGlobal: true}, Log: slog.New(slog.DiscardHandler)})
		h := watch(t, n)
		h.intents <- &netv1.Intent{Watch: true, IntervalMs: time.Minute.Milliseconds(), WaitMs: (30 * time.Second).Milliseconds(), Stun: &netv1.StunList{}}
		a, b, c := "2001:db8:1::/64", "2001:db8:2::/64", "2001:db8:3::/64"
		for _, step := range []struct {
			at   time.Duration
			do   func()
			want []string
		}{
			{29 * time.Second, nil, nil},
			{31 * time.Second, nil, []string{a}},
			{90 * time.Second, func() { set(b) }, []string{a}},
			{130 * time.Second, func() { set(a) }, []string{a}},
			{160 * time.Second, nil, []string{a}},
			{210 * time.Second, func() { set(c) }, []string{a}},
			{271 * time.Second, nil, []string{a, c}},
			{310 * time.Second, func() { h.intents <- &netv1.Intent{} }, []string{a, c, "unwatched"}},
		} {
			time.Sleep(time.Until(epoch.Add(step.at)))
			if step.do != nil {
				step.do()
			}
			synctest.Wait()
			if got := prefixes(t, h); !reflect.DeepEqual(got, step.want) {
				t.Fatalf("at %v published %v, want %v", step.at, got, step.want)
			}
		}
		mu.Lock()
		before := reads
		mu.Unlock()
		time.Sleep(time.Hour)
		synctest.Wait()
		mu.Lock()
		defer mu.Unlock()
		if reads != before {
			t.Errorf("the watch passed %d times after it stopped, want none", reads-before)
		}
	})
}

// prefixes is each summary h published: its first IPv6 prefix, or
// unwatched.
func prefixes(t *testing.T, h *holder) []string {
	t.Helper()
	var out []string
	for _, f := range h.published() {
		s, err := netmodel.ParseSummary(f)
		if err != nil {
			t.Fatal(err)
		}
		p := "unwatched"
		if s.Watched {
			p = s.Prefix6[0].String()
		}
		out = append(out, p)
	}
	return out
}
