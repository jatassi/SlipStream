package tracker_test

import (
	"encoding/hex"
	"encoding/json"
	"slices"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/fake/faketest"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/tracker"
)

const second = "-LT2000-secondseeder"

// handoff is one account's torrent seeding from seeder, then from second
// too, then seeder stopping while second goes on.
var handoff = []send{
	{0, "heron", seeder, "started", 0, 0, 0},
	{0, "heron", second, "started", 0, 0, 0},
	{10 * time.Minute, "heron", seeder, "stopped", 0, 0, 0},
	{10 * time.Minute, "heron", second, "", 0, 0, 0},
}

func TestARustStopKeepsTheHistoryActiveWhileAnotherPeerSeeds(t *testing.T) {
	t.Run("php", func(t *testing.T) {
		got, _ := seedingRecords(t, catalog(), scenario.Tracker{Mode: scenario.UNIT3D}, handoff, 0)
		expectRecords(t, got, []string{
			"0s heron 0s seeder active",
			"10m0s heron 600s seeder",
			"20m0s heron 600s seeder active",
		})
	})
	t.Run("rust", func(t *testing.T) {
		got, _ := seedingRecords(t, catalog(), scenario.Tracker{Mode: scenario.UNIT3D, Variant: scenario.Rust}, handoff, 0)
		expectRecords(t, got, []string{
			"0s heron 0s seeder active",
			"10m0s heron 600s seeder active",
			"20m0s heron 1200s seeder active",
		})
	})
	t.Run("rust with no other seeder", func(t *testing.T) {
		got, _ := seedingRecords(t, catalog(), scenario.Tracker{Mode: scenario.UNIT3D, Variant: scenario.Rust}, []send{handoff[0], handoff[2]}, 0)
		expectRecords(t, got, []string{"0s heron 0s seeder active", "10m0s heron 600s seeder"})
	})
}

func TestThePeersCaptureHoldsEachStartedAndStopped(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f := faketest.InMemory(t, tracker.Fake, scenario.Scenario{Seed: "tracker"}, opts...)
		start := time.Now()
		for _, a := range []send{
			{0, "heron", seeder, "started", 0, 0, 0},
			{time.Minute, "heron", seeder, "", 0, 0, 0},
			{time.Minute, "heron", seeder, "stopped", 0, 0, 0},
			{time.Minute, "heron", second, "started", 0, 0, 0},
		} {
			time.Sleep(a.after)
			get(t, f, a.query())
		}
		var got []string
		for _, r := range f.Records(t) {
			if r.Kind != tracker.Peers.Kind {
				continue
			}
			var p tracker.Peer
			if err := json.Unmarshal(r.Data, &p); err != nil {
				t.Fatal(err)
			}
			id, _ := hex.DecodeString(p.PeerID)
			line := []string{r.Node, p.Release, string(id), "started " + p.StartedAt.Sub(start).String()}
			if p.StoppedAt != nil {
				line = append(line, "stopped "+p.StoppedAt.Sub(start).String())
			}
			got = append(got, strings.Join(line, " "))
		}
		want := []string{
			"seedbox orchard " + seeder + " started 0s",
			"seedbox orchard " + seeder + " started 0s stopped 2m0s",
			"seedbox orchard " + second + " started 3m0s",
		}
		if !slices.Equal(got, want) {
			t.Errorf("peers records =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
		}
	})
}

func TestUNIT3DClearsAWindowAfterTheScenariosClearAfter(t *testing.T) {
	tr := unit3dTracker
	tr.ClearAfter = time.Minute
	announces := []announce{
		{0, passkey, orchard, seeder, 6881, "started", 0, replied(1, 0, 0, "2400", "1800", "")},
		{2 * time.Minute, passkey, orchard, seeder, 6881, "stopped", 0, replied(0, 0, 0, "2400", "1800", "")},
		{time.Minute, passkey, orchard, seeder, 6881, "started", 0, replied(1, 0, 0, "2400", "1800", "")},
	}
	inMemory(t, scenario.Scenario{Seed: "tracker", Tracker: tr}, announces, []string{accepted(announces[0]), accepted(announces[1]), accepted(announces[2])})
}
