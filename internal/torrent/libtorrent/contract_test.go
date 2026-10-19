package libtorrent_test

import (
	"testing"
	"time"

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

func (sessions) Until(t *testing.T, what string, cond func() bool) {
	t.Helper()
	for deadline := time.Now().Add(60 * time.Second); !cond(); time.Sleep(10 * time.Millisecond) {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting until %s", what)
		}
	}
}
