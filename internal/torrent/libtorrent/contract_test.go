package libtorrent_test

import (
	"bytes"
	"errors"
	"io"
	"net"
	"net/netip"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jatassi/slipstream/fakes/dht/krpc"
	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/node/endpoint"
	"github.com/jatassi/slipstream/internal/torrent/engine"
	"github.com/jatassi/slipstream/internal/torrent/engine/enginecontract"
	"github.com/jatassi/slipstream/internal/torrent/engine/enginetest"
)

func TestTheEngineContract(t *testing.T) { enginecontract.Run(t, sessions{}) }

// sessions is libtorrent sessions on loopback against loopback trackers,
// on the wall clock.
type sessions struct{}

func (sessions) Within(t *testing.T, f func(*testing.T)) { f(t) }

func (sessions) Open(t *testing.T, cfg engine.Config) engine.Engine { return openWith(t, cfg) }

func (sessions) Tracker(t *testing.T, rules enginetest.TrackerRules) enginecontract.Tracker {
	a, url := loopbackTracker(t, rules)
	return enginecontract.Tracker{URL: url, Announces: a.Announces, Answer: a.Answer}
}

// Seed has a session of its own seed the release.
func (sessions) Seed(t *testing.T, meta []byte, dir string) func(engine.Engine, engine.Handle) {
	seeder := open(t)
	started(t, seeder, engine.AddParams{Metainfo: meta, SavePath: dir, SeedMode: true})
	addr, err := seeder.ListenAddr()
	if err != nil {
		t.Fatal(err)
	}
	return func(e engine.Engine, h engine.Handle) {
		if err := e.ConnectPeer(h, addr); err != nil {
			t.Fatal(err)
		}
	}
}

func (sessions) TakePort(t *testing.T) netip.AddrPort {
	l, err := new(net.ListenConfig).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = l.Close() })
	return netip.MustParseAddrPort(l.Addr().String())
}

// Incoming connects and sends bytes that are no handshake, plain or
// encrypted, so the session hangs up: once it has, it took the connection.
func (sessions) Incoming(t *testing.T, e engine.Engine) netip.Addr {
	to, err := e.ListenAddr()
	if err != nil {
		t.Fatal(err)
	}
	c, err := (&net.Dialer{Timeout: 10 * time.Second}).DialContext(t.Context(), "tcp", to.String())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Close() }()
	if err := c.SetDeadline(time.Now().Add(60 * time.Second)); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Write(bytes.Repeat([]byte{0xff}, 2048)); err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(io.Discard, c); errors.Is(err, os.ErrDeadlineExceeded) {
		t.Fatalf("the session kept a connection that sent no handshake: %v", err)
	}
	return netip.MustParseAddrPort(c.LocalAddr().String()).Addr()
}

// DHT is fake-dht on loopback.
func (sessions) DHT(t *testing.T) enginecontract.DHT {
	var mu sync.Mutex
	lookups := map[bencode.InfoHash]int{}
	srv := krpc.New(krpc.Config{
		Peers: func(string, bencode.InfoHash) []netip.AddrPort { return nil },
		Record: func(q krpc.Query) {
			mu.Lock()
			defer mu.Unlock()
			if q.Method == "get_peers" {
				lookups[q.InfoHash]++
			}
		},
	})
	t.Cleanup(func() { _ = srv.Close() })
	addr, err := srv.Listen("engine", netip.MustParseAddrPort("127.0.0.1:0"))
	if err != nil {
		t.Fatal(err)
	}
	return enginecontract.DHT{
		Routers: []endpoint.HostPort{endpoint.MustParseHostPort(addr.String())},
		Lookups: func(h bencode.InfoHash) int {
			mu.Lock()
			defer mu.Unlock()
			return lookups[h]
		},
	}
}

func (sessions) Until(t *testing.T, what string, cond func() bool) {
	t.Helper()
	for deadline := time.Now().Add(60 * time.Second); !cond(); time.Sleep(10 * time.Millisecond) {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting until %s", what)
		}
	}
}
