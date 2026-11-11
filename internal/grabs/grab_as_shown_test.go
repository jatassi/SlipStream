package grabs_test

import (
	"slices"
	"testing"
	"testing/synctest"

	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/identity"
	rel "github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/searching"
	"github.com/jatassi/slipstream/internal/targets"
)

// rowPicks is the pick of each row of the plan that grabs.
func rowPicks(v grabs.ManualView) []grabs.ManualPick {
	var out []grabs.ManualPick
	for _, r := range v.Plan.Rows {
		if p := r.Pick; p != nil {
			out = append(out, picked(p.Release, r.Cells, p.Lifts))
		}
	}
	return out
}

func (a atticWorld) offer(t *testing.T, title string) grabs.Offer {
	t.Helper()
	offers, err := a.g.ManualReleases(t.Context(), identity.Admin{}, a.work)
	if err != nil {
		t.Fatal(err)
	}
	i := slices.IndexFunc(offers, func(o grabs.Offer) bool { return o.Release.Title == title })
	if i < 0 {
		t.Fatalf("All Releases lists no %s", title)
	}
	return offers[i]
}

func liftsOf(ws []grabs.Worded) []string {
	out := make([]string, len(ws))
	for i, w := range ws {
		out[i] = w.Words
	}
	return out
}

func TestEveryRowOfAPlanWhoseLaterReleaseAlsoFillsAnEarlierCellIsGrabbed(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		a := newTwinWorld(t, twin{
			goals: []targets.Spec{
				{Name: "Disc", Scores: targets.Scores{Increment: 1}, Cutoff: rel.Bluray1080p, Tiers: [][]rel.Quality{{rel.Bluray1080p}, {rel.WEBDL1080p}}},
				{Name: "Web", Scores: targets.Scores{Increment: 1}, Cutoff: rel.WEBDL1080p, Tiers: [][]rel.Quality{{rel.WEBDL1080p}}},
			},
			sites: []twinSite{{name: "Better Site", priority: 10, releases: []searching.Release{listing(bluRay, 9, "a"), listing(webDL, 9, "b")}}},
		})
		v := a.searchedByHand(t)
		var rows []string
		for _, r := range v.Plan.Rows {
			if r.Pick != nil {
				rows = append(rows, r.Pick.Release.Title+": "+r.Outcome)
			}
		}
		want := []string{bluRay + ": It gets you Disc for T01.", webDL + ": It gets you Web for T01."}
		if v.Plan.Headline != "2 Grabs get you everything." || !slices.Equal(rows, want) {
			t.Fatalf("the plan is %q with rows %q, want 2 Grabs with rows %q", v.Plan.Headline, rows, want)
		}
		for _, r := range a.grabbed(t, v, rowPicks(v)...) {
			made(t, r)
		}
		if n := a.grabRows(t); n != 2 {
			t.Fatalf("grabbing the plan made %d Grabs, want 2", n)
		}
	})
}

func TestAReleaseShowsEveryLiftItsGrabNeeds(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const lq = "The.Attic.2020.1080p.WEB-DL-LQ"
		least := targets.Scores{Minimum: 1, Increment: 1}
		a := newTwinWorld(t, twin{
			goals: []targets.Spec{
				{Name: "Disc", Scores: least, Cutoff: rel.Bluray1080p, Tiers: [][]rel.Quality{{rel.Bluray1080p}, {rel.WEBDL1080p}}},
				{Name: "Web", Scores: least, Cutoff: rel.WEBDL1080p, Tiers: [][]rel.Quality{{rel.WEBDL1080p}}},
			},
			sites: []twinSite{{name: "Better Site", priority: 10, releases: []searching.Release{listing(lq, 9, "a")}}},
		})
		v := a.searchedByHand(t)
		want := []string{"Score 0 is below the minimum of 1 at Disc.", "Score 0 is below the minimum of 1 at Web."}
		o := a.offer(t, lq)
		if got := liftsOf(o.Grab.Lifts); !slices.Equal(got, want) || o.Grab.Outcome != "It gets you Disc and Web for T01." {
			t.Fatalf("All Releases offers %s getting past %q for %q, want %q for both cells", lq, got, o.Grab.Outcome, want)
		}
		_, c := choice(t, v, lq)
		if got := liftsOf(c.Lifts); !slices.Equal(got, want) {
			t.Fatalf("Other choices offers %s getting past %q, want %q", lq, got, want)
		}
		made(t, a.grabbed(t, v, picked(o.Release, o.Grab.Cells, o.Grab.Lifts))[0])
	})
}

func TestARowPinnedAfterACellNothingFillsGrabsTheCellsItShows(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const lq = "The.Attic.2020.1080p.WEB-DL-LQ"
		least := targets.Scores{Minimum: 1, Increment: 1}
		a := newTwinWorld(t, twin{
			goals: []targets.Spec{
				{Name: "Disc", Scores: least, Cutoff: rel.Bluray1080p, Tiers: [][]rel.Quality{{rel.Bluray1080p}, {rel.WEBDL1080p}}},
				{Name: "Web", Scores: least, Cutoff: rel.WEBDL1080p, Tiers: [][]rel.Quality{{rel.WEBDL1080p}}},
			},
			sites: []twinSite{{name: "Better Site", priority: 10, releases: []searching.Release{listing(lq, 9, "a")}}},
		})
		v := a.searchedByHand(t)
		if len(v.Plan.Rows) != 2 || len(v.Plan.Rows[1].Choices) != 1 {
			t.Fatalf("the plan is %+v, want two cells nothing fills, the second offering %s", v.Plan.Rows, lq)
		}
		v, err := a.g.ManualSearch(t.Context(), identity.Admin{}, a.work, []grabs.Pin{v.Plan.Rows[1].Choices[0].Pin})
		if err != nil {
			t.Fatal(err)
		}
		r := v.Plan.Rows[1]
		if r.Pick == nil || r.Outcome != "It gets you Web for T01." || !slices.Equal(liftsOf(r.Pick.Lifts), []string{"Score 0 is below the minimum of 1 at Web."}) {
			t.Fatalf("the pinned row is %+v, want %s for Web alone", r, lq)
		}
		made(t, a.grabbed(t, v, rowPicks(v)...)[0])
	})
}

func TestALaterRowPinnedToAReleaseThatAlsoFillsAnEarlierCellGrabsBothRows(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const other = "The.Attic.2020.1080p.WEB-DL-OTHER"
		a := newTwinWorld(t, twin{
			goals: []targets.Spec{
				{Name: "Disc", Scores: targets.Scores{Increment: 1}, Cutoff: rel.Bluray1080p, Tiers: [][]rel.Quality{{rel.Bluray1080p}, {rel.WEBDL1080p}}},
				{Name: "Web", Scores: targets.Scores{Increment: 1}, Cutoff: rel.WEBDL1080p, Tiers: [][]rel.Quality{{rel.WEBDL1080p}}},
			},
			sites: []twinSite{{name: "Better Site", priority: 10, releases: []searching.Release{listing(bluRay, 9, "a"), listing(webDL, 9, "b"), listing(other, 3, "c")}}},
		})
		v := a.searchedByHand(t)
		i := slices.IndexFunc(v.Plan.Rows[1].Choices, func(c grabs.Choice) bool { return c.Release.Title == other })
		if len(v.Plan.Rows) != 2 || i < 0 {
			t.Fatalf("the plan is %+v, want the Web row offering %s", v.Plan.Rows, other)
		}
		v, err := a.g.ManualSearch(t.Context(), identity.Admin{}, a.work, []grabs.Pin{v.Plan.Rows[1].Choices[i].Pin})
		if err != nil {
			t.Fatal(err)
		}
		r := v.Plan.Rows[1]
		if r.Pick == nil || r.Pick.Release.Title != other || r.Outcome != "It gets you Web for T01." {
			t.Fatalf("the pinned row is %+v, want %s for Web alone", r, other)
		}
		for _, r := range a.grabbed(t, v, rowPicks(v)...) {
			made(t, r)
		}
		if n := a.grabRows(t); n != 2 {
			t.Fatalf("grabbing the plan made %d Grabs, want 2", n)
		}
	})
}
