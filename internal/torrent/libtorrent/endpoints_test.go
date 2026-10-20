package libtorrent_test

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/torrent/engine"
	"github.com/jatassi/slipstream/internal/torrent/libtorrent"
)

// startedFrom is a tracker on every address of the host that logs which
// address each torrent's started announce came from.
type startedFrom struct {
	port uint16

	mu   sync.Mutex
	from map[bencode.InfoHash][]netip.Addr
}

func wildcardTracker(t *testing.T) *startedFrom {
	t.Helper()
	ln, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp4", "0.0.0.0:0")
	if err != nil {
		t.Fatal(err)
	}
	at, err := netip.ParseAddrPort(ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	s := &startedFrom{port: at.Port(), from: map[bencode.InfoHash][]netip.Addr{}}
	srv := httptest.NewUnstartedServer(s)
	srv.Listener = ln
	srv.Start()
	t.Cleanup(srv.Close)
	return s
}

func (s *startedFrom) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	if q.Get("event") == "started" {
		var hash bencode.InfoHash
		copy(hash[:], q.Get("info_hash"))
		from, err := netip.ParseAddrPort(r.RemoteAddr)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		s.mu.Lock()
		s.from[hash] = append(s.from[hash], from.Addr().Unmap())
		s.mu.Unlock()
	}
	body, err := bencode.Encode(map[string]any{"interval": int64(1800), "min interval": int64(300), "peers": ""})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_, _ = w.Write(body)
}

// heard is the address each torrent's started came from, once started
// came for n torrents and no other announce came for quiet after it.
func (s *startedFrom) heard(t *testing.T, n int, quiet time.Duration) map[bencode.InfoHash][]netip.Addr {
	t.Helper()
	read := func() (map[bencode.InfoHash][]netip.Addr, int) {
		s.mu.Lock()
		defer s.mu.Unlock()
		got, total := map[bencode.InfoHash][]netip.Addr{}, 0
		for h, from := range s.from {
			got[h] = from
			total += len(from)
		}
		return got, total
	}
	for deadline := time.Now().Add(60 * time.Second); ; time.Sleep(50 * time.Millisecond) {
		got, total := read()
		if len(got) == n {
			time.Sleep(quiet)
			if _, after := read(); after == total {
				return got
			}
			continue
		}
		if time.Now().After(deadline) {
			t.Fatalf("the tracker heard started from %d of %d torrents", len(got), n)
		}
	}
}

// routeIfaceAddrs is the IPv4 addresses, other than loopback and
// link-local, of the interface the host reaches the internet from. An
// engine on 0.0.0.0 binds its tracker connections to that interface, so a
// tracker elsewhere on the host is out of its reach.
func routeIfaceAddrs(t *testing.T) []netip.Addr {
	t.Helper()
	route := routeAddr(t)
	ifs, err := net.Interfaces()
	if err != nil {
		t.Fatal(err)
	}
	for _, i := range ifs {
		as, err := i.Addrs()
		if err != nil {
			t.Fatal(err)
		}
		var addrs []netip.Addr
		for _, a := range as {
			n, ok := a.(*net.IPNet)
			if !ok {
				continue
			}
			addr, ok := netip.AddrFromSlice(n.IP)
			if addr = addr.Unmap(); ok && addr.Is4() && !addr.IsLoopback() && !addr.IsLinkLocalUnicast() {
				addrs = append(addrs, addr)
			}
		}
		if slices.Contains(addrs, route) {
			return addrs
		}
	}
	t.Fatalf("no interface holds the default route's address %s", route)
	return nil
}

// TestEachTorrentAnnouncesStartedOnce opens an engine on the wildcard
// address, as a Node does by default, and has three torrents announce to
// a tracker at each address of the interface the default route leaves
// from, in turn. A tracker counts each announce as a peer, so it must hear
// started once per torrent however many addresses that interface has. A
// loopback tracker would not show a second announce: libtorrent's announce
// from another address to loopback never arrives.
func TestEachTorrentAnnouncesStartedOnce(t *testing.T) {
	hosts := routeIfaceAddrs(t)
	if len(hosts) < 2 {
		t.Skipf("the default route's interface has only %v, so one announce per torrent proves nothing", hosts)
	}
	for _, host := range hosts {
		t.Run(host.String(), func(t *testing.T) {
			tracker := wildcardTracker(t)
			url := "http://" + netip.AddrPortFrom(host, tracker.port).String() + "/announce"
			e := openWith(t, engine.Config{ListenInterface: netip.MustParseAddrPort("0.0.0.0:0")})
			const torrents = 3
			for i := range torrents {
				name := fmt.Sprintf("Endpoint %d.mkv", i)
				dir := t.TempDir()
				payload := make([]byte, 256<<10)
				_, _ = rand.NewChaCha8([32]byte{byte(i)}).Read(payload)
				if err := os.WriteFile(filepath.Join(dir, name), payload, 0o600); err != nil {
					t.Fatal(err)
				}
				started(t, e, engine.AddParams{Metainfo: trackedTorrent(t, name, payload, 64<<10, url), SavePath: dir, SeedMode: true})
			}
			for hash, from := range tracker.heard(t, torrents, 2*time.Second) {
				if len(from) != 1 {
					t.Errorf("the tracker at %s heard started %d times for torrent %s, from %v; the default route's interface has %v", host, len(from), hash, from, hosts)
				}
			}
		})
	}
}

// routeAddr is the address this host reaches the internet from over IPv4,
// which an engine on 0.0.0.0 listens on.
func routeAddr(t *testing.T) netip.Addr {
	t.Helper()
	conn, err := (&net.Dialer{}).DialContext(t.Context(), "udp4", "198.41.0.4:53")
	if err != nil {
		t.Skipf("this host has no IPv4 route to the internet: %v", err)
	}
	defer func() { _ = conn.Close() }()
	ap, err := netip.ParseAddrPort(conn.LocalAddr().String())
	if err != nil {
		t.Fatal(err)
	}
	return ap.Addr()
}

// seedTo adds a seeding torrent named name that announces to url, and
// resumes it.
func seedTo(t *testing.T, e engine.Engine, name, url string) engine.Handle {
	t.Helper()
	dir := t.TempDir()
	payload := make([]byte, 256<<10)
	_, _ = rand.NewChaCha8([32]byte{name[0]}).Read(payload)
	if err := os.WriteFile(filepath.Join(dir, name), payload, 0o600); err != nil {
		t.Fatal(err)
	}
	return started(t, e, engine.AddParams{Metainfo: trackedTorrent(t, name, payload, 64<<10, url), SavePath: dir, SeedMode: true})
}

// TestATrackerOnTheDefaultRouteSeesOneHealthyAnnounce opens an engine on
// 0.0.0.0, as a Node does by default, and has a torrent announce to a
// tracker at the host's own address. The engine listens on the address the
// host reaches the internet from, and nowhere else, so the tracker list
// shows the one announce it answered, with no failure.
func TestATrackerOnTheDefaultRouteSeesOneHealthyAnnounce(t *testing.T) {
	route := routeAddr(t)
	tracker := wildcardTracker(t)
	url := "http://" + netip.AddrPortFrom(route, tracker.port).String() + "/announce"
	e := openWith(t, engine.Config{ListenInterface: netip.MustParseAddrPort("0.0.0.0:0")})
	if ap, err := e.ListenAddr(); err != nil || ap.Addr() != route {
		t.Fatalf("on 0.0.0.0 the engine listens on %s, %v; want the default route's %s", ap, err, route)
	}
	h := seedTo(t, e, "Route.mkv", url)
	waitFor(t, e.Events(), func(r engine.TrackerReplied) bool { return r.Torrent == h })
	if err := e.PostTrackers(h); err != nil {
		t.Fatal(err)
	}
	list := waitFor(t, e.Events(), func(l engine.TrackerList) bool { return l.Torrent == h })
	if len(list.Trackers) != 1 {
		t.Fatalf("the torrent lists %+v, want its one tracker", list.Trackers)
	}
	if got := list.Trackers[0]; got.Endpoints != 1 || got.LastError != nil || got.Fails != 0 || got.Message != "" || !got.StartSent {
		t.Fatalf("the tracker on the default route reads %+v, want one endpoint whose started was answered, with no failure", got)
	}
	for hash, from := range tracker.heard(t, 1, 2*time.Second) {
		if len(from) != 1 || from[0] != route {
			t.Fatalf("the tracker heard started for torrent %s from %v, want once from %s", hash, from, route)
		}
	}
}

// TestAnEngineWithNoRouteListensOnceOneAppears opens an engine on 0.0.0.0
// while the host has no route to the internet: it listens nowhere and says
// why, until the route appears.
func TestAnEngineWithNoRouteListensOnceOneAppears(t *testing.T) {
	var mu sync.Mutex
	var at netip.Addr
	route := func() (netip.Addr, error) {
		mu.Lock()
		defer mu.Unlock()
		if !at.IsValid() {
			return at, errors.New("network is unreachable")
		}
		return at, nil
	}
	e := openRouted(t, route)
	failed := waitFor(t, e.Events(), func(engine.ListenFailed) bool { return true })
	if !errors.Is(failed.Err, engine.ErrNoRoute) {
		t.Fatalf("with no route the engine failed to listen with %v, want %v", failed.Err, engine.ErrNoRoute)
	}
	if ap, err := e.ListenAddr(); !errors.Is(err, engine.ErrNotListening) {
		t.Fatalf("with no route the engine listens on %s, %v; want %v", ap, err, engine.ErrNotListening)
	}
	loopback := netip.MustParseAddr("127.0.0.1")
	mu.Lock()
	at = loopback
	mu.Unlock()
	listensOn(t, e, func(ap netip.AddrPort) bool { return ap.Addr() == loopback })
}

// TestTheListenerFollowsTheDefaultRoute moves the default route, as a VPN
// that comes up does: the engine moves its listener to the route's new
// address on the same port, and its next announce is a started from there,
// the tracker never having heard from that address.
func TestTheListenerFollowsTheDefaultRoute(t *testing.T) {
	host := routeAddr(t)
	var mu sync.Mutex
	at := netip.MustParseAddr("127.0.0.1")
	route := func() (netip.Addr, error) {
		mu.Lock()
		defer mu.Unlock()
		return at, nil
	}
	e := openRouted(t, route)
	before, err := e.ListenAddr()
	if err != nil || before.Addr() != at {
		t.Fatalf("on a route from %s the engine listens on %s, %v", at, before, err)
	}
	tracker := wildcardTracker(t)
	h := seedTo(t, e, "Moving.mkv", "http://"+netip.AddrPortFrom(host, tracker.port).String()+"/announce")
	tracker.heard(t, 1, 0)
	mu.Lock()
	at = host
	mu.Unlock()
	listensOn(t, e, func(ap netip.AddrPort) bool { return ap == netip.AddrPortFrom(host, before.Port()) })
	if err := e.Reannounce(h, 0, 0, true); err != nil {
		t.Fatal(err)
	}
	for hash, from := range tracker.heard(t, 1, 2*time.Second) {
		if want := []netip.Addr{before.Addr(), host}; !slices.Equal(from, want) {
			t.Fatalf("the tracker heard started for torrent %s from %v, want from %v", hash, from, want)
		}
	}
}

func openRouted(t *testing.T, route func() (netip.Addr, error)) engine.Engine {
	t.Helper()
	cfg := engine.Config{
		ListenInterface: netip.MustParseAddrPort("0.0.0.0:0"),
		RouteCheck:      50 * time.Millisecond,
		Rand:            rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64())), //nolint:gosec // picks listen ports
	}
	e, err := libtorrent.OpenRouted(cfg, route)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() {
		if err := e.Close(); err != nil {
			t.Errorf("close: %v", err)
		}
	})
	return e
}

func listensOn(t *testing.T, e engine.Engine, want func(netip.AddrPort) bool) {
	t.Helper()
	for deadline := time.Now().Add(10 * time.Second); ; time.Sleep(20 * time.Millisecond) {
		ap, err := e.ListenAddr()
		if err == nil && want(ap) {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("the engine listens on %s, %v", ap, err)
		}
	}
}
