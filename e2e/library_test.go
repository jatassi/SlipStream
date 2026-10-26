package e2e

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
	"github.com/jatassi/slipstream/internal/storing"
)

func TestTheWorldSeedsTheScenarioLibrary(t *testing.T) {
	t.Parallel()
	s, err := scenario.Load("music")
	if err != nil {
		t.Fatal(err)
	}
	s.Name = "music-library"
	s.Library = &scenario.Library{
		Roots: []scenario.LibraryRoot{{ContentType: "music", Folder: "music"}},
		Targets: []scenario.LibraryTarget{
			{
				Name: "Lossless", Tiers: [][]scenario.QualityKey{{"remux-1080p", "bluray-1080p"}, {"webdl-1080p"}}, Cutoff: "bluray-1080p",
				Filters: []scenario.LibraryFilter{{Attribute: "language", Mode: "only", Values: []string{"en", "original"}}},
				Sizes:   []scenario.LibrarySize{{Quality: "bluray-1080p", Min: 20, Max: new(100.0)}},
			},
			{Name: "Small", Tiers: [][]scenario.QualityKey{{"webdl-720p"}, {"dvd"}}, Cutoff: "dvd"},
		},
		Profiles: []scenario.LibraryProfile{{Name: "Standard", ContentType: "music", Targets: []string{"Small", "Lossless"}}},
	}
	run := worldtest.LaunchE2EScenario(t, s)
	a := api(t, run)
	m := run.Manifest()

	folder := filepath.Join(m.MediaDir, "music")
	if info, err := os.Stat(folder); err != nil || !info.IsDir() {
		t.Fatalf("the media dir holds no folder music: %v", err)
	}
	var roots struct {
		Roots []struct {
			ID       string `json:"id"`
			State    string `json:"state"`
			VolumeID string `json:"volumeId"`
		} `json:"roots"`
	}
	var reply worldtest.Reply
	worldtest.Until(t, "the music root is ready", func() (bool, error) {
		reply = a.Admin("home", http.MethodGet, "/api/v1/roots", "")
		decode(t, "listRoots", reply, &roots)
		return len(roots.Roots) == 1 && roots.Roots[0].State == "ready", nil
	})
	root := roots.Roots[0]
	worldtest.Expect(t, "listRoots", reply, worldtest.Reply{Status: 200, Body: `{"roots":[{"id":"` + root.ID + `","contentType":"music","nodeId":"` + m.Nodes[0].NodeID + `","path":` + strconv.Quote(folder) +
		`,"default":true,"state":"ready","volumeId":` + strconv.Quote(root.VolumeID) + `,"works":0,"fileBytes":0}]}` + "\n"})
	if marker, err := os.ReadFile(filepath.Join(folder, storing.MarkerName)); err != nil || strings.TrimSpace(string(marker)) != root.ID || root.VolumeID == "" {
		t.Fatalf("the ready root's marker reads %q, %v on volume %q; want its id on a volume", marker, err, root.VolumeID)
	}

	var targets struct {
		Targets []struct {
			ID        string                `json:"id"`
			Profiles  []struct{ ID string } `json:"profiles"`
			CreatedAt string                `json:"createdAt"`
		} `json:"targets"`
	}
	reply = a.Admin("home", http.MethodGet, "/api/v1/targets", "")
	decode(t, "listTargets", reply, &targets)
	starter, lossless, small := targets.Targets[0], targets.Targets[1], targets.Targets[2]
	movies, music := starter.Profiles[0].ID, starter.Profiles[1].ID
	standard := `[{"id":"` + lossless.Profiles[0].ID + `","name":"Standard"}]`
	worldtest.Expect(t, "listTargets", reply, worldtest.Reply{Status: 200, Body: `{"targets":[` +
		`{"id":"` + starter.ID + `","name":"1080p","tiers":[["bluray-1080p","remux-1080p"],["hdtv-1080p","webrip-1080p","webdl-1080p"],["bluray-720p"],["hdtv-720p","webrip-720p","webdl-720p"]],` +
		`"cutoff":"bluray-1080p","filters":[],` +
		`"sizes":[{"quality":"bluray-1080p","min":50.8,"max":null},{"quality":"remux-1080p","min":102,"max":null},{"quality":"hdtv-1080p","min":33.8,"max":null},` +
		`{"quality":"webrip-1080p","min":12.5,"max":null},{"quality":"webdl-1080p","min":12.5,"max":null},{"quality":"bluray-720p","min":25.7,"max":null},` +
		`{"quality":"hdtv-720p","min":17.1,"max":null},{"quality":"webrip-720p","min":12.5,"max":null},{"quality":"webdl-720p","min":12.5,"max":null}],` +
		`"profiles":[{"id":"` + movies + `","name":"Movies"},{"id":"` + music + `","name":"Music"}],"contains":[],"createdAt":"` + starter.CreatedAt + `"},` +
		`{"id":"` + lossless.ID + `","name":"Lossless","tiers":[["remux-1080p","bluray-1080p"],["webdl-1080p"]],"cutoff":"remux-1080p",` +
		`"filters":[{"attribute":"language","mode":"only","values":["en","original"]}],` +
		`"sizes":[{"quality":"remux-1080p","min":102,"max":null},{"quality":"bluray-1080p","min":20,"max":100},{"quality":"webdl-1080p","min":12.5,"max":null}],` +
		`"profiles":` + standard + `,"contains":[],"createdAt":"` + lossless.CreatedAt + `"},` +
		`{"id":"` + small.ID + `","name":"Small","tiers":[["webdl-720p"],["dvd"]],"cutoff":"dvd","filters":[],` +
		`"sizes":[{"quality":"webdl-720p","min":12.5,"max":null},{"quality":"dvd","min":0,"max":null}],` +
		`"profiles":` + standard + `,"contains":[],"createdAt":"` + small.CreatedAt + `"}]}` + "\n"})

	reply = a.Admin("home", http.MethodGet, "/api/v1/profiles", "")
	holds := func(ids ...string) string {
		entries := make([]string, len(ids))
		for i, id := range ids {
			entries[i] = `{"targetId":"` + id + `","sourcing":"download"}`
		}
		return `"targets":[` + strings.Join(entries, ",") + `],"editions":{"kind":"any"},"works":0}`
	}
	worldtest.Expect(t, "listProfiles", reply, worldtest.Reply{Status: 200, Body: `{"profiles":[` +
		`{"id":"` + movies + `","name":"Movies","contentType":"movie","default":true,` + holds(starter.ID) + `,` +
		`{"id":"` + music + `","name":"Music","contentType":"music","default":true,` + holds(starter.ID) + `,` +
		`{"id":"` + lossless.Profiles[0].ID + `","name":"Standard","contentType":"music","default":false,` + holds(small.ID, lossless.ID) + `]}` + "\n"})
}

func TestAFreshControllerWantsTheStarterForItsContentType(t *testing.T) {
	t.Parallel()
	run := worldtest.Launch(t, "single-machine")
	a := api(t, run)

	var targets struct {
		Targets []struct {
			ID       string `json:"id"`
			Name     string `json:"name"`
			Profiles []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"profiles"`
		} `json:"targets"`
	}
	decode(t, "listTargets", a.Admin("home", http.MethodGet, "/api/v1/targets", ""), &targets)
	if len(targets.Targets) != 1 || targets.Targets[0].Name != "1080p" {
		t.Fatalf("listTargets = %+v, want the starter 1080p alone", targets.Targets)
	}
	starter := targets.Targets[0]

	reply := a.Admin("home", http.MethodGet, "/api/v1/profiles", "")
	var profiles struct {
		Profiles []struct {
			ID string `json:"id"`
		} `json:"profiles"`
	}
	decode(t, "listProfiles", reply, &profiles)
	if len(profiles.Profiles) != 1 || len(starter.Profiles) != 1 || starter.Profiles[0].ID != profiles.Profiles[0].ID {
		t.Fatalf("listProfiles = %s with the starter in %+v, want one default holding it", reply.Body, starter.Profiles)
	}
	movies := profiles.Profiles[0]
	worldtest.Expect(t, "listProfiles", reply, worldtest.Reply{Status: 200, Body: `{"profiles":[{"id":"` + movies.ID + `","name":"Movies","contentType":"movie","default":true,` +
		`"targets":[{"targetId":"` + starter.ID + `","sourcing":"download"}],"editions":{"kind":"any"},"works":0}]}` + "\n"})

	first := decode(t, "addWork", a.Admin("home", http.MethodPost, "/api/v1/works", `{"contentType":"movie","key":"9000001","monitor":true}`), &added{})
	var tr tree
	if err := json.Unmarshal(first.Work, &tr); err != nil || !first.Created || len(tr.Items) != 1 {
		t.Fatalf("addWork created %t, work %s: %v; want a created Work with one Item", first.Created, first.Work, err)
	}
	worldtest.Expect(t, "getWorkWants", a.Admin("home", http.MethodGet, "/api/v1/works/"+tr.ID+"/wants", ""), worldtest.Reply{Status: 200,
		Body: `{"profileId":"` + movies.ID + `","profileName":"Movies","editions":{"kind":"any"},"from":"profile"}` + "\n"})
	worldtest.Expect(t, "getWorkCells", a.Admin("home", http.MethodGet, "/api/v1/works/"+tr.ID+"/cells", ""), worldtest.Reply{Status: 200,
		Body: `{"cells":[{"itemId":"` + tr.Items[0].ID + `","edition":"any","targetId":"` + starter.ID + `","targetName":"1080p","wanted":true,"state":"missing"}],` +
			`"items":[{"itemId":"` + tr.Items[0].ID + `","status":"missing"}],` +
			`"summary":{"status":"missing","cells":{"missing":1},"items":1,"itemStatus":{"missing":1}},"needsYou":[]}` + "\n"})
}

func TestSeveralRootsPerTypeWithOneDefault(t *testing.T) {
	t.Parallel()
	s, err := scenario.Load("single-machine")
	if err != nil {
		t.Fatal(err)
	}
	s.Name = "two-movie-roots"
	s.Library.Roots = []scenario.LibraryRoot{{ContentType: "movie", Folder: "movies"}, {ContentType: "movie", Folder: "films"}}
	s.TMDB.Movies = append(s.TMDB.Movies, scenario.TMDBMovie{ID: 9000002, IMDbID: "tt9000002", Title: "The Loud Meadow", Released: "2022-05-20", Runtime: 98 * time.Minute})
	run := worldtest.LaunchE2EScenario(t, s)
	a := api(t, run)

	type listed struct {
		ID      string `json:"id"`
		Path    string `json:"path"`
		Default bool   `json:"default"`
		Works   int    `json:"works"`
	}
	list := func(what string) []listed {
		var out struct {
			Roots []listed `json:"roots"`
		}
		decode(t, what, a.Admin("home", http.MethodGet, "/api/v1/roots", ""), &out)
		return out.Roots
	}
	expect := func(what string, want ...listed) {
		t.Helper()
		got := list(what)
		if len(got) != len(want) {
			t.Fatalf("%s = %+v, want %+v", what, got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("%s = %+v, want %+v", what, got, want)
			}
		}
	}
	roots := list("listRoots")
	if len(roots) != 2 {
		t.Fatalf("listRoots = %+v, want the two movie roots", roots)
	}
	movies, films := roots[0], roots[1]
	if movies.Path != filepath.Join(run.Manifest().MediaDir, "movies") || !movies.Default || films.Default {
		t.Fatalf("listRoots = %+v, want movies first and the default", roots)
	}

	orchard := decode(t, "addWork", a.Admin("home", http.MethodPost, "/api/v1/works", `{"contentType":"movie","key":"9000001"}`), &added{})
	movies.Works = 1
	expect("listRoots after adding a Work", movies, films)

	if r := a.Admin("home", http.MethodPost, "/api/v1/roots/"+films.ID+"/default", ""); r.Status != http.StatusNoContent {
		t.Fatalf("makeDefaultRoot = %d %s", r.Status, r.Body)
	}
	movies.Default, films.Default = false, true
	expect("listRoots after Make default", movies, films)

	decode(t, "addWork into movies", a.Admin("home", http.MethodPost, "/api/v1/works", `{"contentType":"movie","key":"9000002","rootId":"`+movies.ID+`"}`), &added{})
	movies.Works = 2
	expect("listRoots after adding into movies", movies, films)

	var tr tree
	if err := json.Unmarshal(orchard.Work, &tr); err != nil {
		t.Fatal(err)
	}
	worldtest.Expect(t, "setWorkRoot", a.Admin("home", http.MethodPatch, "/api/v1/works/"+tr.ID, `{"rootId":"`+films.ID+`"}`),
		worldtest.Reply{Status: 200, Body: `{"workId":"` + tr.ID + `","rootId":"` + films.ID + `"}` + "\n"})
	movies.Works, films.Works = 1, 1
	expect("listRoots after setWorkRoot", movies, films)
}
