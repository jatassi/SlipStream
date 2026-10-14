package e2e

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/netip"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"testing"

	"github.com/zeebo/blake3"

	"github.com/jatassi/slipstream/fakes/tracker"
	"github.com/jatassi/slipstream/fakes/world"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
	"github.com/jatassi/slipstream/internal/api/apitest"
)

type grabView struct {
	ID       string `json:"id"`
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

var grabsEvent = regexp.MustCompile(`(?m)^event: (grabs\.[a-z-]+)\ndata: (.*)$`)

// awaitGrab reads the stream until a grabs.changed leaves the Work's Grab
// done, and returns it with the grabs events read, in order.
func awaitGrab(t *testing.T, a *worldtest.API, s *apitest.Stream) (g grabView, seen []string) {
	t.Helper()
	ctx := worldtest.Context(t)
	for {
		msg, err := s.Next(ctx)
		if err != nil {
			t.Fatalf("the stream ended before the Grab finished, after %v: %v", seen, err)
		}
		m := grabsEvent.FindStringSubmatch(msg)
		if m == nil {
			continue
		}
		seen = append(seen, m[1])
		if m[1] != "grabs.changed" {
			continue
		}
		var changed struct {
			GrabID string `json:"grabId"`
		}
		if err := json.Unmarshal([]byte(m[2]), &changed); err != nil {
			t.Fatal(err)
		}
		g = *decode(t, "getGrab", a.Admin("home", http.MethodGet, "/api/v1/grabs/"+changed.GrabID, ""), &grabView{})
		if g.Stage == "imported" || g.Stage == "failed" {
			return g, seen
		}
	}
}

func TestASearchedMovieIsDownloadedProbedAndImported(t *testing.T) {
	t.Parallel()
	run := worldtest.Launch(t, "single-machine")
	m := run.Manifest()
	a := api(t, run)
	stream := a.Stream("home", a.SignIn("home"))

	worldtest.Expect(t, "searchCatalog", a.Admin("home", http.MethodGet, "/api/v1/content-types/movie/catalog?query=quiet+orchard", ""), worldtest.Reply{
		Status: 200,
		Body:   `{"candidates":[{"key":"9000001","label":"The Quiet Orchard","fields":[{"key":"year","kind":"number","number":2021}],"workId":null}]}` + "\n",
	})
	var tr tree
	if first := decode(t, "addWork", a.Admin("home", http.MethodPost, "/api/v1/works", `{"contentType":"movie","key":"9000001","monitor":true}`), &added{}); json.Unmarshal(first.Work, &tr) != nil {
		t.Fatalf("addWork = %s", first.Work)
	}
	reply := a.Admin("home", http.MethodPost, "/api/v1/works/"+tr.ID+"/search", "")
	var started struct {
		SearchIDs []string `json:"searchIds"`
	}
	if err := json.Unmarshal([]byte(reply.Body), &started); err != nil || reply.Status != http.StatusAccepted || len(started.SearchIDs) != 1 {
		t.Fatalf("searchWork = %d %s, want 202 with one search", reply.Status, reply.Body)
	}

	g, seen := awaitGrab(t, a, stream)
	if want := []string{"fetching", "downloading", "probing", "importing", "imported"}; g.Stage != "imported" || !slices.Equal(g.stages(), want) {
		t.Fatalf("the Grab ended %s (failure %+v) after %v, want imported after %v", g.Stage, g.Failure, g.stages(), want)
	}
	if seen[0] != "grabs.search-changed" || !slices.Contains(seen, "grabs.changed") {
		t.Fatalf("the stream read %v, want grabs.search-changed first, then grabs.changed", seen)
	}
	checkPlaced(t, m)
	checkCells(t, a, tr.ID)
	checkAnnounces(t, m, g.InfoHash)
}

// checkPlaced checks the media dir holds only the imported file: the
// catalog payload, hard linked to the seeding download.
func checkPlaced(t *testing.T, m world.Manifest) {
	t.Helper()
	placed := "movies/The Quiet Orchard (2021)/The Quiet Orchard (2021) - 1080p.mkv"
	if got := filesUnder(t, m.MediaDir); !slices.Equal(got, []string{placed}) {
		t.Fatalf("the media dir holds %v, want exactly [%s]", got, placed)
	}
	file := filepath.Join(m.MediaDir, filepath.FromSlash(placed))
	release := filepath.Join(m.ScratchDir, "releases", "The.Quiet.Orchard.2021.1080p.WEB-DL.H264-FKW.mkv")
	if got, want := sum(t, file), sum(t, release); got != want {
		t.Fatalf("the placed file's BLAKE3 is %x, want the catalog payload's %x", got, want)
	}
	if n := links(t, file); n != 2 {
		t.Fatalf("the placed file has %d links, want 2: itself and the seeding download", n)
	}
}

func checkCells(t *testing.T, a *worldtest.API, work string) {
	t.Helper()
	var cells struct {
		Cells []struct {
			TargetName string `json:"targetName"`
			State      string `json:"state"`
			File       *struct {
				Path  string `json:"path"`
				Probe struct {
					Width  int `json:"width"`
					Height int `json:"height"`
				} `json:"probe"`
			} `json:"file"`
		} `json:"cells"`
	}
	decode(t, "getWorkCells", a.Admin("home", http.MethodGet, "/api/v1/works/"+work+"/cells", ""), &cells)
	if c := cells.Cells; len(c) != 1 || c[0].TargetName != "1080p" || c[0].State != "met" || c[0].File == nil ||
		c[0].File.Path != "The Quiet Orchard (2021)/The Quiet Orchard (2021) - 1080p.mkv" || c[0].File.Probe.Width != 1920 || c[0].File.Probe.Height != 1080 {
		t.Fatalf("getWorkCells = %+v, want 1080p met by the 1920x1080 file", c)
	}
}

// checkAnnounces checks the Controller announced only the grabbed release:
// started first, then completed with nothing left.
func checkAnnounces(t *testing.T, m world.Manifest, infoHash string) {
	t.Helper()
	home := netip.MustParseAddrPort(m.Nodes[0].Listeners["torrent"]).Port()
	var ours []tracker.Announce
	worldtest.Until(t, "the tracker hears the Controller complete", func() (bool, error) {
		ours = ours[:0]
		for _, an := range records(t, m) {
			if an.Port == home {
				ours = append(ours, an)
			}
		}
		done := slices.IndexFunc(ours, func(an tracker.Announce) bool { return an.Event == "completed" })
		return done >= 0, nil
	})
	if ours[0].Event != "started" || ours[0].InfoHash != infoHash {
		t.Fatalf("the Controller's first announce = %+v, want started for %s", ours[0], infoHash)
	}
	completed := false
	for _, an := range ours {
		if an.InfoHash != infoHash {
			t.Fatalf("the Controller announced %+v, want only the 1080p release %s", an, infoHash)
		}
		completed = completed || an.Event == "completed"
		if completed && an.Left != 0 {
			t.Fatalf("the Controller announced %+v at or after completed, want left=0", an)
		}
	}
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

// records reads fake-tracker's announce records over the run CA.
func records(t *testing.T, m world.Manifest) []tracker.Announce {
	t.Helper()
	pem, err := os.ReadFile(m.CA.File)
	if err != nil {
		t.Fatal(err)
	}
	roots := x509.NewCertPool()
	roots.AppendCertsFromPEM(pem)
	client := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12}}}
	defer client.CloseIdleConnections()
	i := slices.IndexFunc(m.Fakes, func(f world.Fake) bool { return f.Name == tracker.Fake.Name })
	req, err := http.NewRequestWithContext(worldtest.Context(t), http.MethodGet, m.Fakes[i].URL+"/records", http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	res, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()
	var got []tracker.Announce
	if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	return got
}
