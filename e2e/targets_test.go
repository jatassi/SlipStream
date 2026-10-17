package e2e

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/site"
	"github.com/jatassi/slipstream/fakes/tracker"
	"github.com/jatassi/slipstream/fakes/world"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
)

const bluray = "The.Quiet.Orchard.2021.1080p.BluRay.x264-FKW"

type filledCells struct {
	Cells []struct {
		TargetID   string `json:"targetId"`
		TargetName string `json:"targetName"`
		Wanted     bool   `json:"wanted"`
		State      string `json:"state"`
		File       *struct {
			ID      string `json:"id"`
			Path    string `json:"path"`
			Quality string `json:"quality"`
		} `json:"file"`
	} `json:"cells"`
}

// summary is each cell as "<Target> <state> wanted=<bool> <quality>",
// and the one file id every filled cell holds.
func (c filledCells) summary(t *testing.T) (lines []string, file string) {
	t.Helper()
	for _, cell := range c.Cells {
		line := cell.TargetName + " " + cell.State
		if cell.Wanted {
			line += " wanted"
		}
		if cell.File != nil {
			line += " " + cell.File.Quality + " " + cell.File.Path
			if file != "" && cell.File.ID != file {
				t.Fatalf("the cells hold files %s and %s, want one file", file, cell.File.ID)
			}
			file = cell.File.ID
		}
		lines = append(lines, line)
	}
	return lines, file
}

// grabbedTargets launches the targets Scenario, adds its movie monitored
// and searches it: the one listed release, the WEB-DL, is grabbed for
// both of the Profile's Targets and imported.
func grabbedTargets(t *testing.T) walked {
	t.Helper()
	s, err := scenario.Load("targets")
	if err != nil {
		t.Fatal(err)
	}
	run := worldtest.Launch(t, "targets")
	w := walked{s: s, run: run, m: run.Manifest(), a: api(t, run)}
	first := decode(t, "addWork", w.a.Admin("home", http.MethodPost, "/api/v1/works", `{"contentType":"movie","key":"9000001","monitor":true}`), &added{})
	var tr tree
	if err := json.Unmarshal(first.Work, &tr); err != nil || !first.Created {
		t.Fatalf("addWork created %t, work %s: %v", first.Created, first.Work, err)
	}
	w.work = tr.ID
	if got, _ := w.cells(t).summary(t); !slices.Equal(got, []string{"1080p missing wanted", "HD missing wanted"}) {
		t.Fatalf("before the search the cells are %q, want both Targets missing", got)
	}
	found := w.searched(t, w.search(t))
	if found.Outcome == nil || *found.Outcome != "grabbed" || found.Candidates != 1 || found.GrabID == nil {
		t.Fatalf("the search ended %+v, want the one listed release grabbed", found)
	}
	w.ended(t, *found.GrabID)
	if w.grab.Stage != "imported" {
		t.Fatalf("the Grab ended %s with %+v after %v, want imported", w.grab.Stage, w.grab.Failure, w.grab.stages())
	}
	return w
}

func (w *walked) cells(t *testing.T) filledCells {
	t.Helper()
	return *decode(t, "getWorkCells", w.a.Admin("home", http.MethodGet, "/api/v1/works/"+w.work+"/cells", ""), &filledCells{})
}

// holds checks the media dir comes to hold only the file at placed, the
// payload of release name.
func (w *walked) holds(t *testing.T, name string) {
	t.Helper()
	holdsOnly(t, w.m, placed)
	if _, err := world.MatchRelease(w.m, filepath.Join(w.m.MediaDir, filepath.FromSlash(placed)), name); err != nil {
		t.Fatal(err)
	}
}

func TestOneFileFillsTwoTargetsAndSitsOnDiskOnce(t *testing.T) {
	t.Parallel()
	w := grabbedTargets(t)
	w.holds(t, release)
	name := "The Quiet Orchard (2021)/The Quiet Orchard (2021) - 1080p.mkv"
	got, _ := w.cells(t).summary(t)
	if want := []string{"1080p upgradable wanted webdl-1080p " + name, "HD upgradable wanted webdl-1080p " + name}; !slices.Equal(got, want) {
		t.Fatalf("after the import the cells are %q, want %q: one WEB-DL file, named after the first Target, filling both", got, want)
	}
}

type searchRefusals struct {
	Searches []struct {
		ID      string `json:"id"`
		Refused []struct {
			Title   string `json:"title"`
			Reasons []struct {
				Code     string `json:"code"`
				TargetID string `json:"targetId"`
			} `json:"reasons"`
		} `json:"refused"`
	} `json:"searches"`
}

func TestASearchUpgradesTheHeldWEBDLToTheBlurayAndDeletesIt(t *testing.T) {
	t.Parallel()
	w := grabbedTargets(t)
	w.holds(t, release)
	_, webdl := w.cells(t).summary(t)
	if _, err := w.run.Cue(worldtest.Context(t), site.List.Name, json.RawMessage(`{"release":"`+bluray+`"}`)); err != nil {
		t.Fatal(err)
	}

	id := w.search(t)
	found := w.searched(t, id)
	if found.Outcome == nil || *found.Outcome != "grabbed" || found.Candidates != 2 || found.GrabID == nil {
		t.Fatalf("the upgrade search ended %+v, want the Bluray of two candidates grabbed", found)
	}
	list := decode(t, "getWorkSearches", w.a.Admin("home", http.MethodGet, "/api/v1/works/"+w.work+"/searches", ""), &searchRefusals{})
	cells := w.cells(t)
	for _, s := range list.Searches {
		if s.ID != id {
			continue
		}
		if len(s.Refused) != 1 || s.Refused[0].Title != release || len(s.Refused[0].Reasons) != 2 {
			t.Fatalf("the upgrade search refused %+v, want the held WEB-DL alone", s.Refused)
		}
		for i, r := range s.Refused[0].Reasons {
			if r.Code != "not_an_upgrade" || r.TargetID != cells.Cells[i].TargetID {
				t.Fatalf("the WEB-DL was refused for %+v, want not_an_upgrade at both Targets", s.Refused[0].Reasons)
			}
		}
	}
	w.ended(t, *found.GrabID)
	if w.grab.Stage != "imported" {
		t.Fatalf("the upgrade Grab ended %s with %+v after %v, want imported", w.grab.Stage, w.grab.Failure, w.grab.stages())
	}

	w.holds(t, bluray)
	name := "The Quiet Orchard (2021)/The Quiet Orchard (2021) - 1080p.mkv"
	got, file := w.cells(t).summary(t)
	if want := []string{"1080p met bluray-1080p " + name, "HD met bluray-1080p " + name}; !slices.Equal(got, want) || file == webdl {
		t.Fatalf("after the upgrade the cells are %q, want %q in a new file", got, want)
	}
}

type impactView struct {
	Works    int `json:"works"`
	OwnList  int `json:"ownList"`
	Unwanted struct {
		Files int   `json:"files"`
		Bytes int64 `json:"bytes"`
	} `json:"unwanted"`
	Renamed     int `json:"renamed"`
	NewlyWanted int `json:"newlyWanted"`
	Downloading int `json:"downloading"`
}

func (i impactView) String() string {
	return fmt.Sprintf("works=%d ownList=%d unwanted=%d renamed=%d newlyWanted=%d downloading=%d",
		i.Works, i.OwnList, i.Unwanted.Files, i.Renamed, i.NewlyWanted, i.Downloading)
}

type changeView struct {
	Impact impactView `json:"impact"`
}

// put sends the change and checks the Impact it answers, a preview's or
// a save's, is want.
func (w *walked) put(t *testing.T, path, body, want string) {
	t.Helper()
	got := decode(t, "PUT "+path, w.a.Admin("home", http.MethodPut, path, body), &changeView{}).Impact
	if got.String() != want {
		t.Fatalf("PUT %s answered the Impact %s, want %s", path, got, want)
	}
}

// named reads the id of the record called name at the list path.
func (w *walked) named(t *testing.T, path, member, name string) string {
	t.Helper()
	var list map[string][]struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	decode(t, "GET "+path, w.a.Admin("home", http.MethodGet, path, ""), &list)
	for _, r := range list[member] {
		if r.Name == name {
			return r.ID
		}
	}
	t.Fatalf("GET %s lists no %s named %s", path, member, name)
	return ""
}

// created reads the id of the record r answers created.
func created(t *testing.T, r worldtest.Reply) string {
	t.Helper()
	var v struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(r.Body), &v); err != nil || r.Status != http.StatusCreated {
		t.Fatalf("the create = %d %s, want 201: %v", r.Status, r.Body, err)
	}
	return v.ID
}

func films(targets ...string) string {
	var entries []string
	for _, id := range targets {
		entries = append(entries, `{"targetId":"`+id+`","sourcing":"download"}`)
	}
	return `{"name":"Films","contentType":"movie","targets":[` + strings.Join(entries, ",") + `],"editions":{"kind":"any"}}`
}

func TestTakingTheTargetsAwayKeepsOrDeletesTheFileAsTheSaveChooses(t *testing.T) {
	t.Parallel()
	w := grabbedTargets(t)
	w.holds(t, release)
	profile := "/api/v1/profiles/" + w.named(t, "/api/v1/profiles", "profiles", "Films")
	starter, hd := w.named(t, "/api/v1/targets", "targets", "1080p"), w.named(t, "/api/v1/targets", "targets", "HD")
	unwanted := "works=1 ownList=0 unwanted=1 renamed=0 newlyWanted=0 downloading=0"

	w.put(t, profile+"?preview=true&unwanted=keep", films(), unwanted)
	holdsOnly(t, w.m, placed)
	w.put(t, profile+"?unwanted=keep", films(), unwanted)
	holdsOnly(t, w.m, "movies/The Quiet Orchard (2021)/The Quiet Orchard (2021) - 1080p (extra 1).mkv")
	if got, _ := w.cells(t).summary(t); len(got) != 0 {
		t.Fatalf("with no Targets the cells are %q, want none", got)
	}

	w.put(t, profile, films(starter, hd), "works=1 ownList=0 unwanted=0 renamed=1 newlyWanted=2 downloading=0")
	w.holds(t, release)

	if r := w.a.Admin("home", http.MethodPut, profile+"?confirm=0", films()); r.Status != http.StatusConflict || !strings.Contains(r.Body, `"code":"impact_changed"`) {
		t.Fatalf("a delete confirming no files = %d %s, want 409 impact_changed", r.Status, r.Body)
	}
	w.holds(t, release)
	w.put(t, profile+"?confirm=1", films(), unwanted)
	holdsOnly(t, w.m)
}

func TestMovingTheWorkToAnotherProfileRenamesItsFileAfterTheTargetItFills(t *testing.T) {
	t.Parallel()
	w := grabbedTargets(t)
	w.holds(t, release)
	web := created(t, w.a.Admin("home", http.MethodPost, "/api/v1/targets", `{"name":"Web","tiers":[["webdl-1080p"]],"cutoff":"webdl-1080p"}`))
	small := created(t, w.a.Admin("home", http.MethodPost, "/api/v1/profiles",
		`{"name":"Small","contentType":"movie","targets":[{"targetId":"`+web+`","sourcing":"download"}],"editions":{"kind":"any"}}`))
	wants := "/api/v1/works/" + w.work + "/wants"
	body := `{"profileId":"` + small + `","editions":null}`
	moved := "works=1 ownList=0 unwanted=0 renamed=1 newlyWanted=0 downloading=0"

	w.put(t, wants+"?preview=true", body, moved)
	w.holds(t, release)
	w.put(t, wants+"?confirm=0", body, moved)
	name := "The Quiet Orchard (2021)/The Quiet Orchard (2021) - Web.mkv"
	holdsOnly(t, w.m, "movies/"+name)
	if got, _ := w.cells(t).summary(t); !slices.Equal(got, []string{"Web met webdl-1080p " + name}) {
		t.Fatalf("after the move the cells are %q, want the WEB-DL filling Web", got)
	}
	worldtest.Expect(t, "getWorkWants", w.a.Admin("home", http.MethodGet, wants, ""), worldtest.Reply{
		Status: 200, Body: `{"profileId":"` + small + `","profileName":"Small","editions":{"kind":"any"},"from":"profile"}` + "\n",
	})
}

func TestAnUpgradeInFlightWhenTheTargetsGoEndsWithNoFile(t *testing.T) {
	t.Parallel()
	w := grabbedTargets(t)
	w.holds(t, release)
	profile := "/api/v1/profiles/" + w.named(t, "/api/v1/profiles", "profiles", "Films")
	w.setFault(t, fault.Rule{ID: "bluray-hold", Target: tracker.SeedersTarget(bluray), Origin: "home"}, "stall")
	if _, err := w.run.Cue(worldtest.Context(t), site.List.Name, json.RawMessage(`{"release":"`+bluray+`"}`)); err != nil {
		t.Fatal(err)
	}
	found := w.searched(t, w.search(t))
	if found.Outcome == nil || *found.Outcome != "grabbed" || found.GrabID == nil {
		t.Fatalf("the upgrade search ended %+v, want the Bluray grabbed", found)
	}
	worldtest.Until(t, "the Bluray Grab is downloading", func() (bool, error) {
		w.grab = *decode(t, "getGrab", w.a.Admin("home", http.MethodGet, "/api/v1/grabs/"+*found.GrabID, ""), &grabView{})
		return w.grab.Stage == "downloading", nil
	})

	gone := "works=1 ownList=0 unwanted=1 renamed=0 newlyWanted=0 downloading=1"
	w.put(t, profile+"?preview=true", films(), gone)
	w.put(t, profile+"?confirm=1", films(), gone)
	holdsOnly(t, w.m)

	if _, err := w.run.ClearFault(worldtest.Context(t), "bluray-hold"); err != nil {
		t.Fatal(err)
	}
	w.ended(t, *found.GrabID)
	if w.grab.Stage != "failed" || w.grab.Failure == nil || w.grab.Failure.Code != "fills_nothing" {
		t.Fatalf("the Bluray Grab ended %s with %+v after %v, want failed fills_nothing", w.grab.Stage, w.grab.Failure, w.grab.stages())
	}
	holdsOnly(t, w.m)
}
