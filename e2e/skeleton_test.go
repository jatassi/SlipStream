package e2e

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/fakes/events"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/tracker"
	"github.com/jatassi/slipstream/fakes/world"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
	"github.com/jatassi/slipstream/internal/storing"
)

const (
	placed  = "movies/The Quiet Orchard (2021)/The Quiet Orchard (2021) - 1080p.mkv"
	release = "The.Quiet.Orchard.2021.1080p.WEB-DL.H264-FKW"
)

type grabView struct {
	ID          string  `json:"id"`
	ReleaseSize int64   `json:"releaseSize"`
	NodeID      string  `json:"nodeId"`
	FetchNodeID *string `json:"fetchNodeId"`
	InfoHash    string  `json:"infoHash"`
	Stage       string  `json:"stage"`
	Progress    *struct {
		Done  int64 `json:"done"`
		Total int64 `json:"total"`
	} `json:"progress"`
	Failure *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"failure"`
	History []struct {
		Stage string `json:"stage"`
	} `json:"history"`
}

func (g grabView) stages() []string {
	var out []string
	for _, e := range g.History {
		out = append(out, e.Stage)
	}
	return out
}

// walked is a world an Admin walked from a catalog search to an imported
// file.
type walked struct {
	s    scenario.Scenario
	run  *world.Run
	m    world.Manifest
	a    *worldtest.API
	work string
	grab grabView
}

// walk launches the catalog Scenario name and walks it as its Admin:
// searchCatalog, addWork monitored, searchWork, and the Grab's events and
// stages until it is imported.
func walk(t *testing.T, name string) walked {
	t.Helper()
	return walkFrom(t, begin(t, name))
}

func walkFrom(t *testing.T, w walked) walked {
	t.Helper()
	stream := worldtest.NewMessages(t, w.m, world.AdminIdentity)
	search := w.search(t)
	event(t, stream, "grabs.search-changed", `{"searchId":"`+search+`","workId":"`+w.work+`"}`)
	w.ended(t, grabChanged(t, stream, w.work))
	if w.grab.Stage != "imported" {
		t.Fatalf("the Grab ended %s with %+v after %v, want imported", w.grab.Stage, w.grab.Failure, w.grab.stages())
	}
	w.tookTheAdvertised(t, search)
	return w
}

// tookTheAdvertised checks the search took the 1080p release at the size
// the site advertised. The starter Target accepts the 720p one too and
// ranks it lower, and the movie's known runtime lets it check sizes, so
// nothing is refused and nothing noted.
func (w *walked) tookTheAdvertised(t *testing.T, search string) {
	t.Helper()
	if want := int64(w.s.Releases[0].AdvertisedSize); w.grab.ReleaseSize != want {
		t.Fatalf("the Grab's release size is %d, want the advertised %d", w.grab.ReleaseSize, want)
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
	decode(t, "listSearches", reply, &list)
	if len(list.Searches) != 1 {
		t.Fatalf("listSearches = %s, want one search", reply.Body)
	}
	r := list.Searches[0]
	worldtest.Expect(t, "listSearches", reply, worldtest.Reply{Status: 200, Body: `{"searches":[{"id":"` + search + `","itemId":"` + r.ItemID +
		`","indexerId":"` + r.IndexerID + `","startedAt":"` + r.StartedAt + `","finishedAt":"` + r.FinishedAt + `","outcome":"grabbed","candidates":2,` +
		`"refused":[],"grabId":"` + w.grab.ID + `","notes":[]}]}` + "\n"})
}

// begin launches the catalog Scenario name, and its Admin finds the
// catalog movie and adds it monitored.
func begin(t *testing.T, name string) walked {
	t.Helper()
	s, err := scenario.Load(name)
	if err != nil {
		t.Fatal(err)
	}
	return beginIn(t, s, worldtest.Launch(t, name))
}

func beginIn(t *testing.T, s scenario.Scenario, run *world.Run) walked {
	t.Helper()
	w := walked{s: s, run: run, m: run.Manifest(), a: api(t, run)}

	worldtest.Expect(t, "searchCatalog", w.a.Admin("home", http.MethodGet, "/api/v1/content-types/movie/catalog?query=quiet+orchard", ""), worldtest.Reply{
		Status: 200,
		Body:   `{"candidates":[{"key":"9000001","label":"The Quiet Orchard","fields":[{"key":"year","kind":"number","number":2021}],"posterUrl":null,"workId":null}]}` + "\n",
	})
	first := decode(t, "addWork", w.a.Admin("home", http.MethodPost, "/api/v1/works", `{"contentType":"movie","key":"9000001","monitor":true}`), &added{})
	var tr tree
	if err := json.Unmarshal(first.Work, &tr); err != nil || !first.Created || len(tr.Items) != 1 {
		t.Fatalf("addWork created %t, work %s: %v; want a created Work with one Item", first.Created, first.Work, err)
	}
	w.work = tr.ID
	target := cells(t, w.a, w.work).Cells[0].TargetID
	worldtest.Expect(t, "getWorkCells before the search", w.a.Admin("home", http.MethodGet, "/api/v1/works/"+w.work+"/cells", ""), worldtest.Reply{
		Status: 200,
		Body: `{"cells":[{"itemId":"` + tr.Items[0].ID + `","edition":"any","targetId":"` + target + `","targetName":"1080p","wanted":true,` +
			`"state":"missing"}],"items":[{"itemId":"` + tr.Items[0].ID + `","status":"missing"}]}` + "\n",
	})
	return w
}

// search starts a search of w's Work and returns its run's id.
func (w *walked) search(t *testing.T) string {
	t.Helper()
	reply := w.a.Admin("home", http.MethodPost, "/api/v1/works/"+w.work+"/search", "")
	var started struct {
		SearchIDs []string `json:"searchIds"`
	}
	if err := json.Unmarshal([]byte(reply.Body), &started); err != nil || reply.Status != http.StatusAccepted || len(started.SearchIDs) != 1 {
		t.Fatalf("searchWork = %d %s, want 202 with one search", reply.Status, reply.Body)
	}
	worldtest.Expect(t, "searchWork", reply, worldtest.Reply{Status: http.StatusAccepted, Body: `{"searchIds":["` + started.SearchIDs[0] + `"]}` + "\n"})
	return started.SearchIDs[0]
}

// ended reads the Grab id until it is imported or failed.
func (w *walked) ended(t *testing.T, id string) {
	t.Helper()
	worldtest.Until(t, "the Grab is imported or failed", func() (bool, error) {
		w.grab = *decode(t, "getGrab", w.a.Admin("home", http.MethodGet, "/api/v1/grabs/"+id, ""), &grabView{})
		return w.grab.Stage == "imported" || w.grab.Stage == "failed", nil
	})
}

var grabChangedData = regexp.MustCompile(`^\{"grabId":"([0-9a-f-]{36})","workId":"([0-9a-f-]{36})"\}$`)

// grabChanged reads the stream to the next grabs.changed and returns its
// Grab, which must be work's.
func grabChanged(t *testing.T, s *worldtest.Messages, work string) string {
	t.Helper()
	data := s.Next("grabs.changed")
	m := grabChangedData.FindStringSubmatch(data)
	if len(m) != 3 || m[2] != work {
		t.Fatalf("grabs.changed carried %s, want a Grab of Work %s", data, work)
	}
	return m[1]
}

type cellList struct {
	Cells []struct {
		ItemID   string `json:"itemId"`
		TargetID string `json:"targetId"`
		State    string `json:"state"`
		File     *struct {
			ID   string `json:"id"`
			Size int64  `json:"size"`
		} `json:"file"`
	} `json:"cells"`
}

func cells(t *testing.T, a *worldtest.API, work string) *cellList {
	t.Helper()
	got := decode(t, "getWorkCells", a.Admin("home", http.MethodGet, "/api/v1/works/"+work+"/cells", ""), &cellList{})
	if len(got.Cells) != 1 {
		t.Fatalf("getWorkCells = %+v, want one cell", got)
	}
	return got
}

func TestSkeletonHome(t *testing.T) {
	t.Parallel()
	w := walk(t, "home")
	seedbox := w.m.Nodes[1]
	if want := []string{"fetching", "downloading", "probing", "transferring", "importing", "imported"}; !slices.Equal(w.grab.stages(), want) {
		t.Fatalf("the Grab's history is %v, want %v", w.grab.stages(), want)
	}
	if w.grab.NodeID != seedbox.NodeID {
		t.Fatalf("the Grab's Node is %s, want seedbox %s", w.grab.NodeID, seedbox.NodeID)
	}
	checkPlaced(t, w.m, 1)
	if entries, err := os.ReadDir(filepath.Join(w.m.MediaDir, "movies", ".slipstream-staging")); err != nil || len(entries) != 0 {
		t.Fatalf("the staging folder holds %v, %v; want nothing", entries, err)
	}
	checkAnnounces(t, w, seedbox)
	checkUpgradable(t, w)
}

func TestSkeletonSingleMachine(t *testing.T) {
	t.Parallel()
	w := walk(t, "single-machine")
	if want := []string{"fetching", "downloading", "probing", "importing", "imported"}; !slices.Equal(w.grab.stages(), want) {
		t.Fatalf("the Grab's history is %v, want %v", w.grab.stages(), want)
	}
	if w.grab.NodeID != w.m.Nodes[0].NodeID {
		t.Fatalf("the Grab's Node is %s, want home %s", w.grab.NodeID, w.m.Nodes[0].NodeID)
	}
	checkPlaced(t, w.m, 2)
	if _, err := os.Stat(filepath.Join(w.m.MediaDir, "movies", ".slipstream-staging")); !os.IsNotExist(err) {
		t.Fatalf("the root has a staging folder (%v), want none: no Transfer ran", err)
	}
	checkAnnounces(t, w, w.m.Nodes[0])
	checkUpgradable(t, w)
}

// checkPlaced checks the media dir comes to hold only the imported file
// at its name, the catalog payload's bytes with links links.
func checkPlaced(t *testing.T, m world.Manifest, links uint64) {
	t.Helper()
	holdsOnly(t, m, placed)
	info, err := world.MatchRelease(m, filepath.Join(m.MediaDir, filepath.FromSlash(placed)), release)
	if err != nil {
		t.Fatal(err)
	}
	if info.Links != links {
		t.Fatalf("the placed file has %d links, want %d", info.Links, links)
	}
}

// holdsOnly waits for the media dir to hold exactly paths: an import
// commits before the disk queue gives the file its name.
func holdsOnly(t *testing.T, m world.Manifest, paths ...string) {
	t.Helper()
	var got []string
	worldtest.Until(t, "the media dir holds exactly "+strings.Join(paths, ", "), func() (bool, error) {
		got = filesUnder(t, m.MediaDir)
		return slices.Equal(got, paths), nil
	})
}

// checkUpgradable checks the WEB-DL file fills the starter Target below
// its Bluray-1080p cutoff tier, so the cell stays a Want.
func checkUpgradable(t *testing.T, w walked) {
	t.Helper()
	c := cells(t, w.a, w.work).Cells[0]
	if c.File == nil {
		t.Fatalf("getWorkCells = %+v, want a filled cell", c)
	}
	worldtest.Expect(t, "getWorkCells after the import", w.a.Admin("home", http.MethodGet, "/api/v1/works/"+w.work+"/cells", ""), worldtest.Reply{
		Status: 200,
		Body: `{"cells":[{"itemId":"` + c.ItemID + `","edition":"any","targetId":"` + c.TargetID + `","targetName":"1080p","wanted":true,"state":"upgradable",` +
			`"file":{"id":"` + c.File.ID + `","path":"The Quiet Orchard (2021)/The Quiet Orchard (2021) - 1080p.mkv","size":` + strconv.FormatInt(c.File.Size, 10) +
			`,"quality":"webdl-1080p","probe":{"width":1920,"height":1080}}}],"items":[{"itemId":"` + c.ItemID + `","status":"upgradable"}]}` + "\n",
	})
}

// checkAnnounces checks the announces capture holds the downloader
// announcing the grabbed release with the Scenario's passkey: started,
// then completed with left=0, and left=0 on any later announce while it
// seeds. libtorrent's min_announce_interval holds the next regular
// announce 5 minutes past the tracker's 5s interval, so the test does not
// wait for one. Every other announce is a seeder's, from the world, so no
// other Node announced, and every announce names the grabbed release, so
// the 720p decoy was never fetched.
func checkAnnounces(t *testing.T, w walked, downloader world.Node) {
	t.Helper()
	var all []events.Record
	var ours []tracker.Announce
	worldtest.Until(t, "the downloader's completed announce is captured", func() (bool, error) {
		var err error
		all, err = events.Select(w.m.RunDir, events.Query{Kind: tracker.Announces.Kind})
		ours = nil
		for _, r := range all {
			if r.Node == downloader.Name {
				ours = append(ours, announce(t, r))
			}
		}
		return slices.ContainsFunc(ours, func(an tracker.Announce) bool { return an.Event == "completed" }), err
	})
	if ours[0].Event != "started" {
		t.Fatalf("the downloader's first announce = %+v, want started", ours[0])
	}
	passkey := tracker.Passkey(w.s)
	completed := false
	for _, an := range ours {
		if an.Passkey != passkey || an.Failure != "" {
			t.Fatalf("the downloader announced %+v, want passkey %s accepted", an, passkey)
		}
		completed = completed || an.Event == "completed"
		if completed && an.Left != 0 {
			t.Fatalf("the downloader announced %+v at or after completed, want left=0", an)
		}
	}
	for _, r := range all {
		if r.Node != downloader.Name && r.Node != events.World {
			t.Fatalf("the tracker heard %s from %s, a Node that downloads nothing", r.Data, r.Node)
		}
		if an := announce(t, r); an.InfoHash != w.grab.InfoHash {
			t.Fatalf("the tracker heard %+v, want only the grabbed release %s", an, w.grab.InfoHash)
		}
	}
}

func announce(t *testing.T, r events.Record) tracker.Announce {
	t.Helper()
	var an tracker.Announce
	if err := json.Unmarshal(r.Data, &an); err != nil {
		t.Fatalf("announce record %d: %v", r.Seq, err)
	}
	return an
}

func filesUnder(t *testing.T, dir string) []string {
	t.Helper()
	var out []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || d.Name() == storing.MarkerName {
			return err
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return fmt.Errorf("relative to the media dir: %w", err)
		}
		out = append(out, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}
