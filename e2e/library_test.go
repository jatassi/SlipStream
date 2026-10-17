package e2e

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
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
		Roots []struct{ ID string } `json:"roots"`
	}
	reply := a.Admin("home", http.MethodGet, "/api/v1/roots", "")
	decode(t, "listRoots", reply, &roots)
	worldtest.Expect(t, "listRoots", reply, worldtest.Reply{Status: 200, Body: `{"roots":[{"id":"` + roots.Roots[0].ID + `","contentType":"music","nodeId":"` + m.Nodes[0].NodeID + `","path":` + strconv.Quote(folder) + `}]}` + "\n"})

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
		Body: `{"cells":[{"itemId":"` + tr.Items[0].ID + `","edition":"standard","targetId":"` + starter.ID + `","targetName":"1080p","wanted":true,"state":"missing"}],` +
			`"items":[{"itemId":"` + tr.Items[0].ID + `","status":"missing"}]}` + "\n"})
}
