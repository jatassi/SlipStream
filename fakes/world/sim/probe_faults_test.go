package sim_test

import (
	"testing"
	"testing/synctest"

	"github.com/jatassi/slipstream/fakes/world/sim"
	"github.com/jatassi/slipstream/fakes/world/sim/simtest"
)

// TestAProbeTimesOutThenFailsThenReads plays a timeout, then an ffprobe
// error, on the release's file: the Grab's Import holds the file on each
// and probes it again, and the third probe reads it and imports it.
func TestAProbeTimesOutThenFailsThenReads(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		key := release + ".mkv"
		probes := simtest.NewProbes()
		probes.Play(key, simtest.ProbeTimeout, simtest.ProbeUnreadable)
		w := sim.Launch(t, simtest.Home(t), sim.WithWalk(), probes.Option())
		simtest.Search(t, w, simtest.AddOrchard(t, w))
		simtest.ExpectImported(t, simtest.Await(t, w))
		if n := probes.Probed(key); n != 3 {
			t.Fatalf("the Nodes probed %s %d times, want 3", key, n)
		}
	})
}
