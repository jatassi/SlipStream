package e2e

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/jatassi/slipstream/fakes/world"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
)

// sized launches the sizes Scenario, adds the movie key monitored and
// returns the world and the id of its one Target, Sized.
func sized(t *testing.T, key string) (w walked, target string) {
	t.Helper()
	run := worldtest.Launch(t, "sizes")
	w = walked{run: run, m: run.Manifest(), a: api(t, run)}
	first := decode(t, "addWork", w.a.Admin("home", http.MethodPost, "/api/v1/works", `{"contentType":"movie","key":"`+key+`","monitor":true}`), &added{})
	var tr tree
	if err := json.Unmarshal(first.Work, &tr); err != nil || !first.Created {
		t.Fatalf("addWork created %t, work %s: %v", first.Created, first.Work, err)
	}
	w.work = tr.ID
	target = w.named(t, "/api/v1/targets", "targets", "Sized")
	return w, target
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
	w, target := sized(t, "9000001")
	w.searchedOnce(t, 3, `[`+
		`{"title":"The.Quiet.Orchard.2021.1080p.WEB-DL.H264-BLOAT","reasons":[{"code":"size_above","targetId":"`+target+`"}]},`+
		`{"title":"The.Quiet.Orchard.2021.1080p.WEB-DL.H264-TINY","reasons":[{"code":"size_below","targetId":"`+target+`"}]}]`, `[]`)
	w.holdsRelease(t, "movies/The Quiet Orchard (2021)/The Quiet Orchard (2021) - Sized.mkv", "The.Quiet.Orchard.2021.1080p.WEB-DL.H264-FKW")
}

func TestAWorkWithNoKnownRuntimeGrabsWithItsSizeNotChecked(t *testing.T) {
	t.Parallel()
	w, target := sized(t, "9000002")
	w.searchedOnce(t, 1, `[]`, `[{"code":"size_not_checked","targetId":"`+target+`"}]`)
	w.holdsRelease(t, "movies/The Lantern Keeper (2019)/The Lantern Keeper (2019) - Sized.mkv", "The.Lantern.Keeper.2019.1080p.WEB-DL.H264-FKW")
}
