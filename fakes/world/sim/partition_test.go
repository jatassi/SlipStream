package sim_test

import (
	"slices"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/world/sim"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
	"github.com/jatassi/slipstream/internal/backoff"
	"github.com/jatassi/slipstream/internal/contenttype"
	"github.com/jatassi/slipstream/internal/node"
	"github.com/jatassi/slipstream/internal/role"
)

// seedbox is seedbox's row in listNodes on home.
func seedbox(t *testing.T, a *worldtest.API) worldtest.Row {
	t.Helper()
	for _, r := range a.Nodes("home") {
		if r.Name == "seedbox" {
			return r
		}
	}
	t.Fatal("listNodes on home has no row seedbox")
	return worldtest.Row{}
}

func standing(t *testing.T, w *sim.World, name string) role.Standing {
	t.Helper()
	snap, ok := w.Node(name).Probe().Snapshot()
	if !ok {
		t.Fatalf("%s's Probe has no snapshot", name)
	}
	return snap.Standing
}

// redialAt is when the Keep loop of the Node seeded seed lands its first
// dial at or after heal, its session to the Controller having ended at
// down: it waits a backoff drawn from the Node's link stream before each
// attempt, and each attempt before heal blocks on the stalled path for the
// whole handshake.
func redialAt(seed string, down, heal time.Time) time.Time {
	timings := node.Defaults(contenttype.Set{}).Timings.Link
	r := node.NewStreams(seed).Stream("link")
	at := down
	for n := 0; ; n++ {
		at = at.Add(backoff.Delay(timings.RedialMin, timings.RedialMax, n, r))
		if !at.Before(heal) {
			return at
		}
		at = at.Add(timings.Handshake)
	}
}

// TestPartitionThenRedial stalls the path between home and seedbox past
// keepalive and write_timeout: seedbox's Probe turns Unreachable and home
// lists it down. Roles set while it is down reach it once the path heals,
// at the instant seedbox's seeded link stream picks for its re-dial.
func TestPartitionThenRedial(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := sim.Launch(t, home(t), sim.WithProbe())
		a := w.API()
		synctest.Wait()
		stalled := time.Now()
		w.Path("home", "seedbox").Set(fault.Stall{})
		link := node.Defaults(contenttype.Set{}).Timings.Link
		time.Sleep(link.Keepalive + link.WriteTimeout)
		synctest.Wait()
		down := stalled.Add(link.Keepalive + link.WriteTimeout)
		if got := standing(t, w, "seedbox"); got != role.Unreachable {
			t.Fatalf("seedbox's Probe stands %s after the stall, want unreachable", got)
		}
		box := seedbox(t, a)
		if box.Link.State != "down" || box.Link.Since != down.Format(time.RFC3339Nano) {
			t.Fatalf("listNodes shows seedbox %s since %s, want down since %s", box.Link.State, box.Link.Since, down.Format(time.RFC3339Nano))
		}

		set := a.SetRoles("home", box.ID, "searching")
		if set.Link.State != "down" || set.Facts != nil {
			t.Fatalf("setNodeRoles on a down Node answered link %s with facts %+v, want down at once", set.Link.State, set.Facts)
		}
		heal := down.Add(time.Minute)
		time.Sleep(time.Until(heal))
		w.Path("home", "seedbox").Set(fault.None{})
		redial := redialAt("home/seedbox", down, heal)
		if want := "2000-01-01T00:01:38.400755643Z"; redial.Format(time.RFC3339Nano) != want {
			t.Fatalf("seedbox's link stream re-dials at %s, want %s", redial.Format(time.RFC3339Nano), want)
		}
		time.Sleep(time.Until(redial) - time.Nanosecond)
		synctest.Wait()
		if box := seedbox(t, a); box.Link.State != "down" {
			t.Fatalf("seedbox is %s before its re-dial at %s", box.Link.State, redial.Format(time.RFC3339Nano))
		}
		time.Sleep(time.Nanosecond)
		synctest.Wait()
		box = seedbox(t, a)
		if box.Link.State != "up" || box.Link.Since != redial.Format(time.RFC3339Nano) {
			t.Fatalf("listNodes shows seedbox %s since %s, want up since %s", box.Link.State, box.Link.Since, redial.Format(time.RFC3339Nano))
		}
		if want := []string{"searching", "probe"}; box.Facts == nil || !slices.Equal(box.Facts.Assigned, want) || box.Facts.AppliedVersion != box.IntentVersion {
			t.Fatalf("seedbox's facts %+v at intentVersion %s, want %v applied", box.Facts, box.IntentVersion, want)
		}
		if got := standing(t, w, "seedbox"); got != role.Live {
			t.Fatalf("seedbox's Probe stands %s after the re-dial, want live", got)
		}
	})
}
