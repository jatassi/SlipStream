package sim_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/world/sim"
	"github.com/jatassi/slipstream/fakes/world/sim/simtest"
)

type manualRelease struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type manualLift struct {
	Code     string `json:"code"`
	TargetID string `json:"targetId,omitempty"`
}

type manualOffer struct {
	Release manualRelease `json:"release"`
	Lifts   []manualLift  `json:"lifts"`
	Edition *string       `json:"edition"`
}

type manualSheet struct {
	ID        string `json:"id"`
	Searching bool   `json:"searching"`
	Plan      struct {
		Rows []struct {
			Pick    *manualOffer  `json:"pick"`
			Choices []manualOffer `json:"choices"`
		} `json:"rows"`
	} `json:"plan"`
}

// searchByHand starts a Manual search of work and waits in fake time for
// every Indexer to respond.
func searchByHand(t *testing.T, w *sim.World, work, body string) manualSheet {
	t.Helper()
	path := "/api/v1/works/" + work + "/manual-search"
	if r := w.API().Admin("home", http.MethodPost, path, body); r.Status != http.StatusAccepted {
		t.Fatalf("startManualSearch = %d %s, want 202", r.Status, r.Body)
	}
	for range simtest.Settled / time.Second {
		v := simtest.Decode[manualSheet](t, "getManualSearch", w.API().Admin("home", http.MethodGet, path, ""))
		if !v.Searching {
			return v
		}
		time.Sleep(time.Second)
	}
	t.Fatalf("the Manual search still searches after %s", simtest.Settled)
	return manualSheet{}
}

// offered is the plan's pick or choice of title.
func (v manualSheet) offered(t *testing.T, title string) manualOffer {
	t.Helper()
	for _, r := range v.Plan.Rows {
		if r.Pick != nil && r.Pick.Release.Title == title {
			return *r.Pick
		}
		for _, c := range r.Choices {
			if c.Release.Title == title {
				return c
			}
		}
	}
	t.Fatalf("the plan offers no %s: %+v", title, v.Plan.Rows)
	return manualOffer{}
}

// grabByHand grabs o from Manual search v and returns its Grab.
func grabByHand(t *testing.T, w *sim.World, work string, v manualSheet, o manualOffer) string {
	t.Helper()
	pick := map[string]any{"releaseId": o.Release.ID, "lifts": o.Lifts}
	if o.Edition != nil {
		pick["edition"] = *o.Edition
	}
	body, err := json.Marshal(map[string]any{"manualSearchId": v.ID, "picks": []any{pick}})
	if err != nil {
		t.Fatal(err)
	}
	out := simtest.Decode[struct {
		Results []struct {
			GrabID  *string         `json:"grabId"`
			Problem json.RawMessage `json:"problem"`
		} `json:"results"`
	}](t, "grabFromManualSearch", w.API().Admin("home", http.MethodPost, "/api/v1/works/"+work+"/manual-search/grabs", string(body)))
	if len(out.Results) != 1 || out.Results[0].GrabID == nil {
		t.Fatalf("grabFromManualSearch answered %+v, want a Grab", out.Results)
	}
	return *out.Results[0].GrabID
}

type grabRecord struct {
	simtest.GrabView
	Source    string   `json:"source"`
	GrabbedBy *string  `json:"grabbedBy"`
	Overrides []string `json:"overrides"`
}

// awaitGrab waits in fake time until Grab id is done.
func awaitGrab(t *testing.T, w *sim.World, id string) grabRecord {
	t.Helper()
	for range simtest.Settled / time.Second {
		g := simtest.Decode[grabRecord](t, "getGrab", w.API().Admin("home", http.MethodGet, "/api/v1/grabs/"+id, ""))
		if g.Stage == "imported" || g.Stage == "failed" {
			return g
		}
		time.Sleep(time.Second)
	}
	t.Fatalf("Grab %s didn't finish within %s", id, simtest.Settled)
	return grabRecord{}
}

// rules is a Target's body, as updateTarget takes it.
type rules struct {
	Name    string          `json:"name"`
	Tiers   [][]string      `json:"tiers"`
	Cutoff  string          `json:"cutoff"`
	Filters json.RawMessage `json:"filters"`
	Sizes   []struct {
		Quality string   `json:"quality"`
		Min     float64  `json:"min"`
		Max     *float64 `json:"max"`
	} `json:"sizes"`
	Scores struct {
		Formats      json.RawMessage `json:"formats"`
		Minimum      int             `json:"minimum"`
		UpgradeUntil int             `json:"upgradeUntil"`
		Increment    int             `json:"increment"`
	} `json:"scores"`
	Propers string `json:"propers"`
}

// editTarget changes the Scenario's one Target as edit does to its rules.
func editTarget(t *testing.T, w *sim.World, edit func(*rules)) {
	t.Helper()
	list := simtest.Decode[struct {
		Targets []struct {
			ID string `json:"id"`
			rules
		} `json:"targets"`
	}](t, "listTargets", w.API().Admin("home", http.MethodGet, "/api/v1/targets", ""))
	target := list.Targets[0]
	edit(&target.rules)
	body, err := json.Marshal(target.rules)
	if err != nil {
		t.Fatal(err)
	}
	if r := w.API().Admin("home", http.MethodPut, "/api/v1/targets/"+target.ID, string(body)); r.Status != http.StatusOK {
		t.Fatalf("updateTarget = %d %s, want 200", r.Status, r.Body)
	}
}

// listed is the Scenario's 1080p release renamed name, listed at size
// unless size is 0.
func listed(s scenario.Scenario, name string, size scenario.Bytes) scenario.Release {
	r := s.Releases[0]
	r.Name, r.Seeded = name, true
	if size > 0 {
		r.AdvertisedSize = size
	}
	return r
}

// lifting is a Release grabbed past the soft refusal lift: title, listed
// at size, or the Scenario's own when it is fkw, with the Target as edit
// leaves it. overrides is what its Grab says it was grabbed past.
type lifting struct {
	lift, title string
	size        scenario.Bytes
	untagged    bool
	edit        func(*rules)
	overrides   string
}

const fkw = "The.Quiet.Orchard.2021.1080p.WEB-DL.H264-FKW"

// TestAGrabPastEachLiftableRefusalImports grabs from Manual search past
// each soft refusal it may lift. Each Grab imports, and its record says
// who grabbed it and what it was grabbed past.
func TestAGrabPastEachLiftableRefusalImports(t *testing.T) {
	t.Parallel()
	for _, c := range []lifting{
		{lift: "wrong_title", title: "The.Quiet.Orchard.2019.1080p.WEB-DL.H264-FKW", untagged: true,
			overrides: "although its name gives The Quiet Orchard (2019), which isn't this Work."},
		{lift: "size_below", title: "The.Quiet.Orchard.2021.1080p.WEB-DL.H264-SML", size: 100 << 20,
			overrides: "although it's smaller than 1080p allows."},
		{lift: "size_above", title: fkw, edit: func(r *rules) {
			for i := range r.Sizes {
				if r.Sizes[i].Quality == "webdl-1080p" {
					r.Sizes[i].Max = new(20.0)
				}
			}
		}, overrides: "although it's larger than 1080p allows."},
		{lift: "score_below_minimum", title: fkw, edit: func(r *rules) { r.Scores.Minimum = 10 },
			overrides: "although score 0 is below the minimum of 10 at 1080p."},
	} {
		t.Run(c.lift, func(t *testing.T) {
			t.Parallel()
			synctest.Test(t, c.imports)
		})
	}
}

func (c lifting) imports(t *testing.T) {
	s := simtest.Home(t)
	if c.title != fkw {
		r := listed(s, c.title, c.size)
		r.Untagged = c.untagged
		s.Releases = append(s.Releases, r)
	}
	w := sim.Launch(t, s, sim.WithWalk())
	work := simtest.AddOrchard(t, w)
	if c.edit != nil {
		editTarget(t, w, c.edit)
	}
	v := searchByHand(t, w, work, "")
	o := v.offered(t, c.title)
	var codes []string
	for _, l := range o.Lifts {
		codes = append(codes, l.Code)
	}
	if !slices.Equal(codes, []string{c.lift}) {
		t.Fatalf("%s gets past %v, want %s alone", c.title, codes, c.lift)
	}
	g := awaitGrab(t, w, grabByHand(t, w, work, v, o))
	simtest.ExpectImported(t, g.GrabView)
	want := []string{"It was grabbed through the API key " + c.overrides}
	if g.Source != "manual" || g.GrabbedBy == nil || *g.GrabbedBy != "the API key" || !slices.Equal(g.Overrides, want) {
		t.Fatalf("the Grab is from %s by %v with overrides %q, want a Manual Grab through the API key with %q", g.Source, g.GrabbedBy, g.Overrides, want)
	}
}

// TestAGrabAsAnotherEditionKeepsItsFillAfterARefill wants only the
// Director's Cut of The Quiet Orchard and grabs its Standard 1080p release
// as that Edition. Its file fills the Director's Cut cell, and still does
// after a change to the Target refills the Work.
func TestAGrabAsAnotherEditionKeepsItsFillAfterARefill(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		w := sim.Launch(t, simtest.Home(t), sim.WithWalk())
		work := simtest.AddOrchard(t, w)
		if r := w.API().Admin("home", http.MethodPut, "/api/v1/works/"+work+"/wants", `{"editions":{"kind":"cuts","cuts":["directors-cut"]}}`); r.Status != http.StatusOK {
			t.Fatalf("setWorkWants = %d %s, want 200", r.Status, r.Body)
		}
		v := searchByHand(t, w, work, "")
		o := v.offered(t, fkw)
		if o.Edition == nil || *o.Edition != "directors-cut" {
			t.Fatalf("the plan grabs it as %v, want the Director's Cut", o.Edition)
		}
		g := awaitGrab(t, w, grabByHand(t, w, work, v, o))
		simtest.ExpectImported(t, g.GrabView)
		if want := []string{"It was grabbed through the API key as the Director's Cut Edition."}; !slices.Equal(g.Overrides, want) {
			t.Fatalf("the Grab's overrides are %q, want %q", g.Overrides, want)
		}
		filled := func() string {
			var cut, edition string
			err := w.Node("home").Half().Library.QueryRowContext(t.Context(),
				`SELECT f.cut, fl.edition FROM files_fill fl JOIN files_file f ON f.id = fl.file_id`).Scan(&cut, &edition)
			if err != nil {
				t.Fatal(err)
			}
			return fmt.Sprintf("%s file filling %s", cut, edition)
		}
		if got := filled(); got != "directors-cut file filling directors-cut" {
			t.Fatalf("the library holds a %s, want the Director's Cut file filling its cell", got)
		}
		editTarget(t, w, func(r *rules) { r.Name = "Full HD" })
		synctest.Wait()
		if got := filled(); got != "directors-cut file filling directors-cut" {
			t.Fatalf("after a refill the library holds a %s, want the Director's Cut file still filling its cell", got)
		}
	})
}

// TestARepackFromAnotherGroupGrabbedPastItsRefusalImports imports the
// 1080p release from FKW, then grabs a repack of it from another group
// past repack_other_group. The repack imports in its place.
func TestARepackFromAnotherGroupGrabbedPastItsRefusalImports(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		const repack = "The.Quiet.Orchard.2021.1080p.WEB-DL.REPACK.H264-OTH"
		s := simtest.Home(t)
		s.Releases = append(s.Releases, listed(s, repack, 0))
		w := sim.Launch(t, s, sim.WithWalk())
		work := simtest.AddOrchard(t, w)
		v := searchByHand(t, w, work, "")
		simtest.ExpectImported(t, awaitGrab(t, w, grabByHand(t, w, work, v, v.offered(t, fkw))).GrabView)

		v = searchByHand(t, w, work, `{"again":true}`)
		o := v.offered(t, repack)
		if len(o.Lifts) != 1 || o.Lifts[0].Code != "repack_other_group" {
			t.Fatalf("the repack gets past %+v, want repack_other_group alone", o.Lifts)
		}
		g := awaitGrab(t, w, grabByHand(t, w, work, v, o))
		simtest.ExpectImported(t, g.GrabView)
		want := []string{"It was grabbed through the API key although it's a repack from OTH, and the file at 1080p is from FKW."}
		if !slices.Equal(g.Overrides, want) {
			t.Fatalf("the Grab's overrides are %q, want %q", g.Overrides, want)
		}
	})
}
