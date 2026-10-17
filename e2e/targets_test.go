package e2e

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"slices"
	"testing"

	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/site"
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
