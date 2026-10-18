package e2e

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"slices"
	"strconv"
	"testing"

	"github.com/jatassi/slipstream/fakes/world"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
)

// sized launches the sizes Scenario, adds the movie key monitored,
// wanting from the Profile named profile, and returns the world and the
// id of the Profile's one Target, named target.
func sized(t *testing.T, key, profile, target string) (w walked, id string) {
	t.Helper()
	run := worldtest.Launch(t, "sizes")
	w = walked{run: run, m: run.Manifest(), a: api(t, run)}
	body := `{"contentType":"movie","key":"` + key + `","monitor":true,"profileId":"` + w.named(t, "/api/v1/profiles", "profiles", profile) + `"}`
	first := decode(t, "addWork", w.a.Admin("home", http.MethodPost, "/api/v1/works", body), &added{})
	var tr tree
	if err := json.Unmarshal(first.Work, &tr); err != nil || !first.Created {
		t.Fatalf("addWork created %t, work %s: %v", first.Created, first.Work, err)
	}
	w.work = tr.ID
	id = w.named(t, "/api/v1/targets", "targets", target)
	return w, id
}

// searchedOnce searches w's Work, checks the searches view lists that one
// search exactly, grabbed of candidates with refused and notes as their
// JSON, and waits for the Grab it made to be imported.
func (w *walked) searchedOnce(t *testing.T, candidates int, refused, notes string) {
	t.Helper()
	id := w.search(t)
	found := w.searched(t, id)
	if found.Outcome == nil || *found.Outcome != "grabbed" || found.GrabID == nil {
		t.Fatalf("the search ended %+v, want a release grabbed", found)
	}
	reply := w.a.Admin("home", http.MethodGet, "/api/v1/works/"+w.work+"/searches", "")
	var list struct {
		Searches []struct {
			ItemID     string `json:"itemId"`
			IndexerID  string `json:"indexerId"`
			StartedAt  string `json:"startedAt"`
			FinishedAt string `json:"finishedAt"`
		} `json:"searches"`
	}
	decode(t, "getWorkSearches", reply, &list)
	if len(list.Searches) != 1 {
		t.Fatalf("getWorkSearches = %s, want one search", reply.Body)
	}
	s := list.Searches[0]
	worldtest.Expect(t, "getWorkSearches", reply, worldtest.Reply{Status: 200, Body: `{"searches":[{"id":"` + id + `","itemId":"` + s.ItemID +
		`","indexerId":"` + s.IndexerID + `","startedAt":"` + s.StartedAt + `","finishedAt":"` + s.FinishedAt + `","outcome":"grabbed","candidates":` +
		strconv.Itoa(candidates) + `,"refused":` + refused + `,"grabId":"` + *found.GrabID + `","notes":` + notes + `}]}` + "\n"})
	w.ended(t, *found.GrabID)
	if w.grab.Stage != "imported" {
		t.Fatalf("the Grab ended %s with %+v after %v, want imported", w.grab.Stage, w.grab.Failure, w.grab.stages())
	}
}

// holdsRelease checks the media dir comes to hold only the file at path,
// the payload of release name.
func (w *walked) holdsRelease(t *testing.T, path, name string) {
	t.Helper()
	holdsOnly(t, w.m, path)
	if _, err := world.MatchRelease(w.m, filepath.Join(w.m.MediaDir, filepath.FromSlash(path)), name); err != nil {
		t.Fatal(err)
	}
}

// The Quiet Orchard runs 104 minutes, so Sized takes 2080 to 10400 MiB of
// WEB-DL: the 14GiB encode is above it and the 1.5GiB one below, though
// both have more seeders than the 8.2GiB one it grabs. No title names its
// audio or channels, so Sized's filters pass every release.
func TestASearchRefusesReleasesOutsideTheTargetsSizesAndGrabsTheOneWithin(t *testing.T) {
	t.Parallel()
	w, target := sized(t, "9000001", "Films", "Sized")
	w.searchedOnce(t, 3, `[`+
		`{"title":"The.Quiet.Orchard.2021.1080p.WEB-DL.H264-BLOAT","reasons":[{"code":"size_above","targetId":"`+target+`"}]},`+
		`{"title":"The.Quiet.Orchard.2021.1080p.WEB-DL.H264-TINY","reasons":[{"code":"size_below","targetId":"`+target+`"}]}]`, `[]`)
	w.holdsRelease(t, "movies/The Quiet Orchard (2021)/The Quiet Orchard (2021) - Sized.mkv", "The.Quiet.Orchard.2021.1080p.WEB-DL.H264-FKW")
}

func TestAWorkWithNoKnownRuntimeGrabsWithItsSizeNotChecked(t *testing.T) {
	t.Parallel()
	w, target := sized(t, "9000002", "Films", "Sized")
	w.searchedOnce(t, 1, `[]`, `[{"code":"size_not_checked","targetId":"`+target+`"}]`)
	w.holdsRelease(t, "movies/The Lantern Keeper (2019)/The Lantern Keeper (2019) - Sized.mkv", "The.Lantern.Keeper.2019.1080p.WEB-DL.H264-FKW")
}

// The Copper Bell's title doesn't name its video codec, so Encoded's
// filter passes it at grab time; its payload probes as H.264, which
// Encoded refuses, so the probed file fills nothing and no file lands.
func TestAFileProbedAtAVideoCodecItsTargetRefusesFillsNothing(t *testing.T) {
	t.Parallel()
	w, target := sized(t, "9000003", "Encodes", "Encoded")
	found := w.searched(t, w.search(t))
	if found.Outcome == nil || *found.Outcome != "grabbed" || found.Candidates != 1 || found.GrabID == nil {
		t.Fatalf("the search ended %+v, want the one listed release grabbed", found)
	}
	w.ended(t, *found.GrabID)
	if w.grab.Stage != "failed" || w.grab.Failure == nil || w.grab.Failure.Code != "fills_nothing" ||
		w.grab.Failure.Message != "a WEBDL-1080p file fills no Target the Work still wants" {
		t.Fatalf("the Grab ended %s with %+v after %v, want failed filling nothing", w.grab.Stage, w.grab.Failure, w.grab.stages())
	}
	if got := w.grab.stages(); !slices.Equal(got[len(got)-2:], []string{"probing", "failed"}) {
		t.Fatalf("the Grab went through %v, want it to fail at its probe", got)
	}
	holdsOnly(t, w.m)
	cells := w.cells(t)
	if got, _ := cells.summary(t); !slices.Equal(got, []string{"Encoded failed wanted"}) || cells.Cells[0].TargetID != target {
		t.Fatalf("after the probe the cells are %q, want Encoded failed and still wanted", got)
	}
}
