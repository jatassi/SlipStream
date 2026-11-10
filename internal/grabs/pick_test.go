package grabs_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"testing/synctest"

	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/grabs/grabstest"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/searching"
	"github.com/jatassi/slipstream/internal/storage"
)

// picked is the pick of r, filling cells and getting past lifts, as the
// sheet sends it.
func picked(r grabs.ReleaseDetails, cells []files.CellKey, lifts []grabs.Worded) grabs.ManualPick {
	p := grabs.ManualPick{Release: r.Ref, Cells: cells}
	for _, l := range lifts {
		p.Lifts = append(p.Lifts, grabs.LiftKey{Code: l.Code, Target: l.Target})
	}
	return p
}

// rowPick is the pick of the plan's first row.
func rowPick(t *testing.T, v grabs.ManualView) grabs.ManualPick {
	t.Helper()
	r := v.Plan.Rows[0]
	if r.Pick == nil {
		t.Fatalf("the plan's first row grabs nothing: %+v", r)
	}
	return picked(r.Pick.Release, r.Cells, r.Pick.Lifts)
}

// choice is the pick of the first row's other choice titled title.
func choice(t *testing.T, v grabs.ManualView, title string) (grabs.ManualPick, grabs.Choice) {
	t.Helper()
	for _, c := range v.Plan.Rows[0].Choices {
		if c.Release.Title == title {
			return picked(c.Release, c.Cells, c.Lifts), c
		}
	}
	t.Fatalf("the first row has no choice %s: %+v", title, v.Plan.Rows[0].Choices)
	return grabs.ManualPick{}, grabs.Choice{}
}

func (a atticWorld) grabbed(t *testing.T, v grabs.ManualView, picks ...grabs.ManualPick) []grabs.PickResult {
	t.Helper()
	out, err := a.g.GrabFromManual(t.Context(), identity.Admin{}, a.work, v.ID, picks)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != len(picks) {
		t.Fatalf("%d picks have %d results", len(picks), len(out))
	}
	return out
}

func (a atticWorld) grabRows(t *testing.T) int {
	t.Helper()
	var n int
	if err := a.lib.QueryRowContext(t.Context(), `SELECT count(*) FROM grabs_grab`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func (a atticWorld) view(t *testing.T, id grabs.GrabID) grabs.View {
	t.Helper()
	v, err := a.g.Get(t.Context(), identity.Admin{}, id)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func made(t *testing.T, r grabs.PickResult) grabs.GrabID {
	t.Helper()
	if r.Err != nil || r.Grab.IsZero() {
		t.Fatalf("the pick was refused with %+v, want a Grab", r.Err)
	}
	return r.Grab
}

func refusedWith(t *testing.T, r grabs.PickResult, code grabs.PickCode, words string) {
	t.Helper()
	if r.Err == nil || r.Err.Code != code || r.Err.Words != words || !r.Grab.IsZero() {
		t.Fatalf("the pick answered %+v, %+v; want %s: %s", r.Grab, r.Err, code, words)
	}
}

func TestTwoPicksOfOneReleaseFromTwoIndexersMakeOneGrab(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		a := newTwinWorld(t, twin{sites: []twinSite{
			{name: "Better Site", priority: 10, releases: []searching.Release{listing(bluRay, 9, "a")}},
			{name: "Other Site", priority: 25, releases: []searching.Release{listing(bluRay, 9, "a")}},
		}})
		v := a.searchedByHand(t)
		first := rowPick(t, v)
		copied := first
		copied.Release.Position = 1 - first.Release.Position
		out := a.grabbed(t, v, first, copied)
		if made(t, out[0]) != made(t, out[1]) || a.grabRows(t) != 1 {
			t.Fatalf("the two picks answered %v and %v with %d Grabs, want one Grab for both", out[0].Grab, out[1].Grab, a.grabRows(t))
		}
		g := a.view(t, out[0].Grab)
		if g.Source != grabs.FromManual || g.Manual != v.ID || g.Release.Title != bluRay || a.names[g.Ticket.Indexer] != "Better Site" {
			t.Fatalf("the Grab is %s from %v of %s on %s, want a Manual Grab of this search taking Better Site's copy",
				g.Source, g.Manual, g.Release.Title, a.names[g.Ticket.Indexer])
		}
	})
}

func TestTwoPicksForOneCellAreBothRefused(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		a := newTwinWorld(t, twin{sites: []twinSite{{name: "Better Site", priority: 10, releases: []searching.Release{
			listing(bluRay, 9, "a"), listing(webDL, 9, "b"),
		}}}})
		v := a.searchedByHand(t)
		other, _ := choice(t, v, webDL)
		out := a.grabbed(t, v, rowPick(t, v), other)
		for _, r := range out {
			refusedWith(t, r, grabs.PickConflict, "Two rows fill HD for T01. Untick one of them.")
		}
		if n := a.grabRows(t); n != 0 {
			t.Fatalf("the refused picks made %d Grabs, want none", n)
		}
	})
}

func TestAPickForAHeldCellIsRefusedUnlessItCancelsTheGrabThere(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		a := newTwinWorld(t, twin{sites: []twinSite{{name: "Better Site", priority: 10, releases: []searching.Release{
			listing(bluRay, 9, "a"), listing(webDL, 9, "b"),
		}}}})
		v := a.searchedByHand(t)
		other, _ := choice(t, v, webDL)
		held := made(t, a.grabbed(t, v, rowPick(t, v))[0])

		refused := a.grabbed(t, v, other)[0]
		refusedWith(t, refused, grabs.CellHeld, "A Grab is already downloading HD for T01. Cancel it to grab this one.")
		if refused.Err.Held != held {
			t.Fatalf("cell_held names %v, want the held Grab %v", refused.Err.Held, held)
		}

		other.Cancel = &held
		replaced := made(t, a.grabbed(t, v, other)[0])
		if g := a.view(t, held); g.Stage != grabs.Failed || g.Failed.Code != grabs.Canceled || g.Failed.Message != canceledByKey {
			t.Fatalf("the held Grab is %s with %+v, want failed as canceled through the API key", g.Stage, g.Failed)
		}
		if g := a.view(t, replaced); g.Stage != grabs.Fetching || g.Release.Title != webDL {
			t.Fatalf("the new Grab is %s of %s, want fetching %s", g.Stage, g.Release.Title, webDL)
		}
	})
}

func TestARetriedGrabAnswersTheGrabItMade(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		a := newTwinWorld(t, twin{sites: []twinSite{{name: "Better Site", priority: 10, releases: []searching.Release{listing(bluRay, 9, "a")}}}})
		v := a.searchedByHand(t)
		first := made(t, a.grabbed(t, v, rowPick(t, v))[0])
		if again := made(t, a.grabbed(t, v, rowPick(t, v))[0]); again != first || a.grabRows(t) != 1 {
			t.Fatalf("the retried pick answered %v with %d Grabs, want %v alone", again, a.grabRows(t), first)
		}
	})
}

func TestAGrabFromAReplacedManualSearchIsRefusedWhole(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		a := newTwinWorld(t, twin{sites: []twinSite{{name: "Better Site", priority: 10, releases: []searching.Release{listing(bluRay, 9, "a")}}}})
		old := a.searchedByHand(t)
		if _, _, err := a.g.StartManualSearch(t.Context(), identity.Admin{}, a.work, grabs.ManualScope{Again: true}); err != nil {
			t.Fatal(err)
		}
		_, err := a.g.GrabFromManual(t.Context(), identity.Admin{}, a.work, old.ID, []grabs.ManualPick{rowPick(t, old)})
		if !errors.Is(err, grabs.ErrManualSearchReplaced) || a.grabRows(t) != 0 {
			t.Fatalf("the grab = %v with %d Grabs, want ErrManualSearchReplaced and none", err, a.grabRows(t))
		}
	})
}

func TestAGrabIsRefusedWholeWhileReconciling(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		a := newTwinWorld(t, twin{sites: []twinSite{{name: "Better Site", priority: 10, releases: []searching.Release{listing(bluRay, 9, "a")}}}})
		v := a.searchedByHand(t)
		a.cfg.Reconciling = func(context.Context, storage.Reader[storage.Library]) (bool, error) { return true, nil }
		_, err := a.open().GrabFromManual(t.Context(), identity.Admin{}, a.work, v.ID, []grabs.ManualPick{rowPick(t, v)})
		if !errors.Is(err, grabs.ErrReconciling) || a.grabRows(t) != 0 {
			t.Fatalf("the grab = %v with %d Grabs, want ErrReconciling and none", err, a.grabRows(t))
		}
	})
}

func TestAGrabThroughTheAPIKeyRecordsNoAccount(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		a := newTwinWorld(t, twin{sites: []twinSite{{name: "Better Site", priority: 10, releases: []searching.Release{listing(bluRay, 9, "a")}}}})
		v := a.searchedByHand(t)
		g := a.view(t, made(t, a.grabbed(t, v, rowPick(t, v))[0]))
		if g.By == nil || g.By.Account != nil || g.By.Words() != "the API key" {
			t.Fatalf("the Grab is by %+v, want the API key with no account", g.By)
		}
		var n int
		if err := a.lib.QueryRowContext(t.Context(), `SELECT count(*) FROM grabs_grab WHERE grabbed_by IS NULL AND grabbed_by_name IS NULL`).Scan(&n); err != nil || n != 1 {
			t.Fatalf("%d Grabs (%v) record no account and no name, want 1", n, err)
		}
	})
}

func TestAPickPastASoftRefusalNamesItsLiftAndRecordsIt(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const atticus = "The.Atticus.2020.1080p.BluRay.x264-GRP"
		a := newTwinWorld(t, twin{sites: []twinSite{{name: "Better Site", priority: 10, releases: []searching.Release{
			listing(webDL, 9, "a"), listing(atticus, 9, "b"),
		}}}})
		v := a.searchedByHand(t)
		lifted, c := choice(t, v, atticus)
		if len(c.Lifts) != 1 {
			t.Fatalf("the choice gets past %+v, want wrong_title alone", c.Lifts)
		}
		bare := lifted
		bare.Lifts = nil
		refusedWith(t, a.grabbed(t, v, bare)[0], grabs.PickChanged, "What this Release gets you changed since the plan. Check the plan and grab again.")

		g := a.view(t, made(t, a.grabbed(t, v, lifted)[0]))
		want := []string{"It was grabbed through the API key although its name gives The Atticus (2020), which isn't this Work."}
		if !slices.Equal(g.Overrides, want) {
			t.Fatalf("the Grab's overrides are %q, want %q", g.Overrides, want)
		}
	})
}

func TestAHardRefusedPickIsNotGrabbable(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const hd720 = "The.Attic.2020.720p.WEB-DL-GRP"
		a := newTwinWorld(t, twin{sites: []twinSite{{name: "Better Site", priority: 10, releases: []searching.Release{
			listing(webDL, 9, "a"), listing(hd720, 9, "b"),
		}}}})
		v := a.searchedByHand(t)
		offers, err := a.g.ManualReleases(t.Context(), identity.Admin{}, a.work)
		if err != nil {
			t.Fatal(err)
		}
		i := slices.IndexFunc(offers, func(o grabs.Offer) bool { return o.Release.Title == hd720 })
		refusedWith(t, a.grabbed(t, v, picked(offers[i].Release, offers[i].Grab.Cells, nil))[0], grabs.NotGrabbable, "HD doesn't accept WEBDL-720p.")
	})
}

func TestRemoveFromBlocklistAndGrab(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		a := newTwinWorld(t, twin{sites: []twinSite{{name: "Better Site", priority: 10, releases: []searching.Release{
			listing(webDL, 9, "a"), listing(bluRay, 9, "b"),
		}}}})
		grabstest.Blocked(t, a.lib, entry(hashed("b"), bluRay))
		v := a.searchedByHand(t)
		blocked, c := choice(t, v, bluRay)
		if c.Release.Blocked == "" {
			t.Fatalf("the BluRay choice reads %+v, want it on the Blocklist", c.Release)
		}
		refusedWith(t, a.grabbed(t, v, blocked)[0], grabs.PickChanged, "This Release is on the Blocklist now. Remove it from the Blocklist to grab it.")

		blocked.Unblock = true
		g := a.view(t, made(t, a.grabbed(t, v, blocked)[0]))
		if want := []string{"It was removed from the Blocklist through the API key to grab it."}; !slices.Equal(g.Overrides, want) {
			t.Fatalf("the Grab's overrides are %q, want %q", g.Overrides, want)
		}
		if page := a.page(t, grabs.BlocklistQuery{}); page.Total != 0 {
			t.Fatalf("the Blocklist holds %+v, want nothing", page.Entries)
		}
	})
}

// TestAPickSentToTheRoutedClientRecordsNoRouting sends a pick to the
// client Tracker routing would pick: the record says where it went, and
// names no other client routing would have sent it to.
func TestAPickSentToTheRoutedClientRecordsNoRouting(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		a := newTwinWorld(t, twin{sites: []twinSite{{name: "Better Site", priority: 10, releases: []searching.Release{listing(bluRay, 9, "a")}}}})
		v := a.searchedByHand(t)
		p := rowPick(t, v)
		home := a.home(t)
		p.Client = &downloadclients.Choice{BuiltInOn: home}
		g := a.view(t, made(t, a.grabbed(t, v, p)[0]))
		if g.Sent == nil || g.Sent.BuiltInOn != home {
			t.Fatalf("the Grab is sent to %+v, want home's built-in client", g.Sent)
		}
		want := "It was sent to the built-in client on home through the API key."
		if !slices.Equal(g.Overrides, []string{want}) {
			t.Fatalf("the Grab's overrides are %q, want %q", g.Overrides, want)
		}
	})
}

// home is the world's one Node.
func (a atticWorld) home(t *testing.T) role.NodeID {
	t.Helper()
	r, ok := a.cfg.Roster.(*roster)
	if !ok {
		t.Fatalf("the world's roster is %T", a.cfg.Roster)
	}
	return r.node
}

// TestAReleasesDetailsNameTheTrackerItsIndexerReaches reads a plan row's
// Release from a Torznab feed whose items name no source: its Details name
// the Tracker the feed reaches.
func TestAReleasesDetailsNameTheTrackerItsIndexerReaches(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		a := newTwinWorld(t, twin{sites: []twinSite{{name: "Better Site", priority: 10, releases: []searching.Release{listing(bluRay, 9, "a")}}}})
		v := a.searchedByHand(t)
		if got := v.Plan.Rows[0].Pick.Release.Tracker; got != "Better Site" {
			t.Fatalf("the row's Release names the Tracker %q, want Better Site", got)
		}
	})
}

// TestACellAGrabFetchesReadsItsReleaseInWords grabs the plan's row and
// reads the Manual search again: the cell is left out of the plan and
// names the Grab's Release in the words its row used, as does the Grab.
func TestACellAGrabFetchesReadsItsReleaseInWords(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		a := newTwinWorld(t, twin{sites: []twinSite{{name: "Better Site", priority: 10, releases: []searching.Release{listing(bluRay, 9, "a")}}}})
		v := a.searchedByHand(t)
		words := v.Plan.Rows[0].Pick.Release.Words
		id := made(t, a.grabbed(t, v, rowPick(t, v))[0])

		after, err := a.g.ManualSearch(t.Context(), identity.Admin{}, a.work, nil)
		if err != nil {
			t.Fatal(err)
		}
		if want := "A Grab is fetching " + words + " for HD."; len(after.Plan.Held) != 1 || after.Plan.Held[0].Words != want {
			t.Fatalf("the held cells read %+v, want one reading %q", after.Plan.Held, want)
		}
		if g := a.view(t, id); g.ReleaseWords != words {
			t.Fatalf("the Grab's Release reads %q, want %q", g.ReleaseWords, words)
		}
	})
}
