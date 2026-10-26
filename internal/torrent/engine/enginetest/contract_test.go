package enginetest_test

import (
	"fmt"
	"net/netip"
	"os"
	"path/filepath"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/node/endpoint"
	"github.com/jatassi/slipstream/internal/torrent/engine"
	"github.com/jatassi/slipstream/internal/torrent/engine/enginecontract"
	"github.com/jatassi/slipstream/internal/torrent/engine/enginetest"
)

func TestTheEngineContract(t *testing.T) { enginecontract.Run(t, &worlds{}) }

// swarmRate is how fast the World's swarm serves a seeded release.
const swarmRate = 1 << 20

// worlds runs each subtest in a World of its own inside a synctest bubble.
type worlds struct {
	w        *enginetest.World
	trackers int
	ports    uint16
}

func (h *worlds) Within(t *testing.T, f func(*testing.T)) {
	synctest.Test(t, func(t *testing.T) {
		h.w = enginetest.NewWorld()
		f(t)
	})
}

func (h *worlds) Open(t *testing.T, cfg engine.Config) engine.Engine {
	if !cfg.ListenInterface.IsValid() {
		cfg.ListenInterface = netip.MustParseAddrPort("127.0.0.1:0")
	}
	e, err := h.w.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = e.Close() })
	return e
}

func (h *worlds) Tracker(_ *testing.T, rules enginetest.TrackerRules) enginecontract.Tracker {
	h.trackers++
	url := fmt.Sprintf("https://tracker%d.example/announce", h.trackers)
	h.w.Tracker(url, rules)
	return enginecontract.Tracker{
		URL:       url,
		Announces: func() []enginetest.Announce { return h.w.Announces(url) },
		Answer:    func() { h.w.Answer(url) },
	}
}

func (h *worlds) Seed(t *testing.T, meta []byte, dir string) func(engine.Engine, engine.Handle) {
	m, err := bencode.DecodeMetainfo(meta)
	if err != nil {
		t.Fatal(err)
	}
	var payload []byte
	for _, f := range m.Info.Files {
		data, err := os.ReadFile(filepath.Join(append([]string{dir, m.Info.Name}, f.Path...)...))
		if err != nil {
			t.Fatal(err)
		}
		payload = append(payload, data...)
	}
	h.w.Payload(m.InfoHash, payload)
	h.w.Transfer(m.InfoHash, swarmRate, 0)
	h.w.Serve(m)
	return func(e engine.Engine, th engine.Handle) {
		if err := e.ConnectPeer(th, netip.MustParseAddrPort("127.0.0.9:6881")); err != nil {
			t.Fatal(err)
		}
	}
}

func (h *worlds) TakePort(*testing.T) netip.AddrPort {
	h.ports++
	ap := netip.AddrPortFrom(netip.MustParseAddr("127.0.0.1"), 7000+h.ports)
	h.w.TakePort(ap)
	return ap
}

func (h *worlds) Incoming(t *testing.T, e engine.Engine) netip.Addr {
	to, err := e.ListenAddr()
	if err != nil {
		t.Fatal(err)
	}
	from := netip.MustParseAddr("127.0.0.1")
	h.w.Incoming(to, from)
	return from
}

func (h *worlds) DHT(*testing.T) enginecontract.DHT {
	return enginecontract.DHT{Routers: []endpoint.HostPort{endpoint.MustParseHostPort("router.example:6881")}, Lookups: h.w.Lookups}
}

// Until polls each fake second for a fake hour.
func (h *worlds) Until(t *testing.T, what string, cond func() bool) {
	t.Helper()
	for deadline := time.Now().Add(time.Hour); !cond(); time.Sleep(time.Second) {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting until %s", what)
		}
	}
}
