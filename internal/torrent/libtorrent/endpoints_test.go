package libtorrent_test

import (
	"fmt"
	"math/rand/v2"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/torrent/engine"
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
	s := &startedFrom{port: uint16(ln.Addr().(*net.TCPAddr).Port), from: map[bencode.InfoHash][]netip.Addr{}} //nolint:gosec // a TCP port fits 16 bits
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

// hostAddrs is the host's IPv4 addresses other than loopback, each of
// which a Node on the wildcard address listens on.
func hostAddrs(t *testing.T) []netip.Addr {
	t.Helper()
	ifs, err := net.Interfaces()
	if err != nil {
		t.Fatal(err)
	}
	var addrs []netip.Addr
	for _, i := range ifs {
		if i.Flags&net.FlagUp == 0 {
			continue
		}
		as, err := i.Addrs()
		if err != nil {
			t.Fatal(err)
		}
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
	}
	return addrs
}

// TestEachTorrentAnnouncesStartedOnce opens an engine on the wildcard
// address, as a Node does by default, and has three torrents announce to
// a tracker at each of the host's addresses in turn. A tracker counts each
// announce as a peer, so it must hear started once per torrent however
// many addresses the host has. A loopback tracker would not show a second
// announce: libtorrent's announce from another address to loopback never
// arrives.
func TestEachTorrentAnnouncesStartedOnce(t *testing.T) {
	hosts := hostAddrs(t)
	if len(hosts) < 2 {
		t.Skipf("this host's only IPv4 addresses besides loopback are %v, so one announce per torrent proves nothing: at most one of its addresses reaches a tracker", hosts)
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
					t.Errorf("the tracker at %s heard started %d times for torrent %s, from %v; the host's addresses are %v", host, len(from), hash, from, hosts)
				}
			}
		})
	}
}
