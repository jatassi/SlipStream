package sim_test

import (
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/world/sim"
	"github.com/jatassi/slipstream/fakes/world/sim/simtest"
)

// TestAProbeTimesOutThenFailsThenReads plays a timeout, then an ffprobe
// error, on the release's file: the first Grab fails on the timeout, the
// second on ffprobe's words, and the third reads the file and imports it.
func TestAProbeTimesOutThenFailsThenReads(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		key := release + ".mkv"
		probes := simtest.NewProbes()
		probes.Play(key, simtest.ProbeTimeout, simtest.ProbeUnreadable)
		w := sim.Launch(t, simtest.Home(t), sim.WithWalk(), probes.Option())
		work := simtest.AddOrchard(t, w)
		seen := map[string]bool{}
		for _, want := range []string{"reading the file timed out", "ffprobe can't read the file: Invalid data found when processing input"} {
			simtest.Search(t, w, work)
			g := awaitNew(t, w, seen)
			expectFailed(t, g, "probing", "probe_failed", want)
		}
		simtest.Search(t, w, work)
		simtest.ExpectImported(t, awaitNew(t, w, seen))
		if n := probes.Probed(key); n != 3 {
			t.Fatalf("the Nodes probed %s %d times, want 3", key, n)
		}
	})
}

// awaitNew waits in fake time until a Grab not in seen is done, and
// returns it, adding it to seen.
func awaitNew(t *testing.T, w *sim.World, seen map[string]bool) simtest.GrabView {
	t.Helper()
	for range simtest.Settled / time.Second {
		for _, g := range simtest.GrabsListed(t, w) {
			if !seen[g.ID] && (g.Stage == "imported" || g.Stage == "failed") {
				seen[g.ID] = true
				return g
			}
		}
		time.Sleep(time.Second)
	}
	t.Fatalf("no new Grab finished within %s: %+v", simtest.Settled, simtest.GrabsListed(t, w))
	return simtest.GrabView{}
}
