package e2e

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"testing"

	"github.com/zeebo/blake3"

	"github.com/jatassi/slipstream/fakes/events"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/tracker"
	"github.com/jatassi/slipstream/fakes/world"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
	"github.com/jatassi/slipstream/internal/api/apitest"
)

const (
	placed  = "movies/The Quiet Orchard (2021)/The Quiet Orchard (2021) - 1080p.mkv"
	release = "The.Quiet.Orchard.2021.1080p.WEB-DL.H264-FKW"
)

type grabView struct {
	ID       string `json:"id"`
	NodeID   string `json:"nodeId"`
	InfoHash string `json:"infoHash"`
	Stage    string `json:"stage"`
	Failure  *struct {
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
	s, err := scenario.Load(name)
	if err != nil {
		t.Fatal(err)
	}
	run := worldtest.Launch(t, name)
	w := walked{s: s, m: run.Manifest(), a: api(t, run)}
	stream := w.a.Stream("home", w.a.SignIn("home"))

	worldtest.Expect(t, "searchCatalog", w.a.Admin("home", http.MethodGet, "/api/v1/content-types/movie/catalog?query=quiet+orchard", ""), worldtest.Reply{
		Status: 200,
		Body:   `{"candidates":[{"key":"9000001","label":"The Quiet Orchard","fields":[{"key":"year","kind":"number","number":2021}],"workId":null}]}` + "\n",
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
		Body: `{"cells":[{"itemId":"` + tr.Items[0].ID + `","edition":"standard","targetId":"` + target + `","targetName":"1080p","wanted":true,` +
			`"state":"missing"}]}` + "\n",
	})

	reply := w.a.Admin("home", http.MethodPost, "/api/v1/works/"+w.work+"/search", "")
	var started struct {
		SearchIDs []string `json:"searchIds"`
	}
	if err := json.Unmarshal([]byte(reply.Body), &started); err != nil || reply.Status != http.StatusAccepted || len(started.SearchIDs) != 1 {
		t.Fatalf("searchWork = %d %s, want 202 with one search", reply.Status, reply.Body)
	}
	worldtest.Expect(t, "searchWork", reply, worldtest.Reply{Status: http.StatusAccepted, Body: `{"searchIds":["` + started.SearchIDs[0] + `"]}` + "\n"})
	event(t, stream, "grabs.search-changed", `{"searchId":"`+started.SearchIDs[0]+`","workId":"`+w.work+`"}`)
	id := grabChanged(t, stream, w.work)

	worldtest.Until(t, "the Grab is imported or failed", func() (bool, error) {
		w.grab = *decode(t, "getGrab", w.a.Admin("home", http.MethodGet, "/api/v1/grabs/"+id, ""), &grabView{})
		return w.grab.Stage == "imported" || w.grab.Stage == "failed", nil
	})
	if w.grab.Stage != "imported" {
		t.Fatalf("the Grab ended %s with %+v after %v, want imported", w.grab.Stage, w.grab.Failure, w.grab.stages())
	}
	return w
}

var grabChangedData = regexp.MustCompile(`(?m)^event: grabs\.changed\ndata: \{"grabId":"([0-9a-f-]{36})","workId":"([0-9a-f-]{36})"\}$`)

// grabChanged reads the stream to the next grabs.changed and returns its
// Grab, which must be work's.
func grabChanged(t *testing.T, s *apitest.Stream, work string) string {
	t.Helper()
	ctx := worldtest.Context(t)
	for {
		msg, err := s.Next(ctx)
		if err != nil {
			t.Fatalf("the stream ended before grabs.changed: %v", err)
		}
		m := grabChangedData.FindStringSubmatch(msg)
		if m == nil {
			continue
		}
		if m[2] != work {
			t.Fatalf("the stream read %q, want grabs.changed for Work %s", msg, work)
		}
		return m[1]
	}
}

type cellList struct {
	Cells []struct {
		ItemID   string `json:"itemId"`
		TargetID string `json:"targetId"`
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
	checkMet(t, w)
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
	checkMet(t, w)
}

// checkPlaced checks the media dir holds only the imported file, the
// catalog payload's bytes with links links.
func checkPlaced(t *testing.T, m world.Manifest, links uint64) {
	t.Helper()
	if got := filesUnder(t, m.MediaDir); !slices.Equal(got, []string{placed}) {
		t.Fatalf("the media dir holds %v, want exactly [%s]", got, placed)
	}
	file := filepath.Join(m.MediaDir, filepath.FromSlash(placed))
	if got, want := sum(t, file), sum(t, filepath.Join(m.ScratchDir, "releases", release+".mkv")); got != want {
		t.Fatalf("the placed file's BLAKE3 is %x, want the catalog payload's %x", got, want)
	}
	if n := linkCount(t, file); n != links {
		t.Fatalf("the placed file has %d links, want %d", n, links)
	}
}

func checkMet(t *testing.T, w walked) {
	t.Helper()
	c := cells(t, w.a, w.work).Cells[0]
	if c.File == nil {
		t.Fatalf("getWorkCells = %+v, want a met cell", c)
	}
	worldtest.Expect(t, "getWorkCells after the import", w.a.Admin("home", http.MethodGet, "/api/v1/works/"+w.work+"/cells", ""), worldtest.Reply{
		Status: 200,
		Body: `{"cells":[{"itemId":"` + c.ItemID + `","edition":"standard","targetId":"` + c.TargetID + `","targetName":"1080p","wanted":false,"state":"met",` +
			`"file":{"id":"` + c.File.ID + `","path":"The Quiet Orchard (2021)/The Quiet Orchard (2021) - 1080p.mkv","size":` + strconv.FormatInt(c.File.Size, 10) +
			`,"probe":{"width":1920,"height":1080}}}]}` + "\n",
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
		if err != nil || d.IsDir() {
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

func sum(t *testing.T, path string) [32]byte {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	h := blake3.New()
	if _, err := io.Copy(h, f); err != nil {
		t.Fatal(err)
	}
	var out [32]byte
	copy(out[:], h.Sum(nil))
	return out
}
