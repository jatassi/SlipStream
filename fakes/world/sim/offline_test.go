package sim_test

import (
	"slices"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/world/sim"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
	"github.com/jatassi/slipstream/internal/contenttype"
	"github.com/jatassi/slipstream/internal/install"
	"github.com/jatassi/slipstream/internal/node"
	"github.com/jatassi/slipstream/internal/nodes"
)

// homeHolds is the Roles home's own holder half reports it holds.
func homeHolds(t *testing.T, a *worldtest.API) []string {
	t.Helper()
	for _, r := range a.Nodes("home") {
		if r.Self {
			if r.Facts == nil {
				t.Fatal("home has no facts of its own")
			}
			return r.Facts.Assigned
		}
	}
	t.Fatal("listNodes on home has no row of its own")
	return nil
}

// TestHomeCoversSearchingForALapsedSeedbox cuts seedbox off. Gone past the
// offline grace from its last Sync frame, it lapses, and home's own node
// intent holds Searching, but not Downloading, whose torrents wait. Once
// seedbox is back and has settled, home lets Searching go again.
func TestHomeCoversSearchingForALapsedSeedbox(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := sim.Launch(t, home(t))
		a := w.API()
		synctest.Wait()
		own := homeHolds(t, a)
		if slices.Contains(own, "searching") || slices.Contains(own, "downloading") {
			t.Fatalf("home holds %v with seedbox in step, want neither Searching nor Downloading", own)
		}
		flip(t, w.Path("home", "seedbox"), fault.Stall{})
		link := node.Defaults(contenttype.Set{}, install.Dev).Timings.Link
		time.Sleep(link.Keepalive + link.WriteTimeout)
		synctest.Wait()
		box := seedbox(t, a)
		if box.Link.State != "down" {
			t.Fatalf("seedbox is %s after the stall, want down", box.Link.State)
		}
		seen, err := time.Parse(time.RFC3339Nano, box.Link.Since)
		if err != nil {
			t.Fatal(err)
		}
		lapse := seen.Add(nodes.OfflineGrace.Default)
		time.Sleep(time.Until(lapse) - time.Nanosecond)
		synctest.Wait()
		if got := homeHolds(t, a); !slices.Equal(got, own) {
			t.Fatalf("home holds %v just before seedbox lapses, want %v", got, own)
		}
		time.Sleep(time.Nanosecond)
		synctest.Wait()
		covering := homeHolds(t, a)
		if !slices.Contains(covering, "searching") || slices.Contains(covering, "downloading") {
			t.Fatalf("home holds %v once seedbox lapsed, want Searching and not Downloading", covering)
		}

		w.Path("home", "seedbox").Clear("")
		for seedbox(t, a).Link.State != "up" {
			time.Sleep(time.Second)
			synctest.Wait()
		}
		if got := homeHolds(t, a); !slices.Contains(got, "searching") {
			t.Fatalf("home holds %v while seedbox settles, want it still covering Searching", got)
		}
		settle := node.Defaults(contenttype.Set{}, install.Dev).Timings.Nodes.Settle
		time.Sleep(settle)
		synctest.Wait()
		if got := homeHolds(t, a); !slices.Equal(got, own) {
			t.Fatalf("home holds %v once seedbox settled, want %v", got, own)
		}
	})
}
