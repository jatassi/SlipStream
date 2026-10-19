package enginetest_test

import (
	"fmt"
	"net/netip"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/torrent/engine"
	"github.com/jatassi/slipstream/internal/torrent/engine/enginecontract"
	"github.com/jatassi/slipstream/internal/torrent/engine/enginetest"
)

func TestTheEngineContract(t *testing.T) { enginecontract.Run(t, &worlds{}) }

// worlds runs each subtest in a World of its own inside a synctest bubble.
type worlds struct {
	w        *enginetest.World
	trackers int
}

func (h *worlds) Within(t *testing.T, f func(*testing.T)) {
	synctest.Test(t, func(t *testing.T) {
		h.w = enginetest.NewWorld()
		f(t)
	})
}

func (h *worlds) Open(t *testing.T) engine.Engine {
	e, err := h.w.Open(engine.Config{ListenInterface: netip.MustParseAddrPort("127.0.0.1:0")})
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

// Until polls each fake second for a fake hour.
func (h *worlds) Until(t *testing.T, what string, cond func() bool) {
	t.Helper()
	for deadline := time.Now().Add(time.Hour); !cond(); time.Sleep(time.Second) {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting until %s", what)
		}
	}
}
