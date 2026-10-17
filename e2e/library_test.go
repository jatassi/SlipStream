package e2e

import (
	"net/http"
	"os"
	"path/filepath"
	"strconv"
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
	lossless, small := targets.Targets[0], targets.Targets[1]
	standard := `[{"id":"` + lossless.Profiles[0].ID + `","name":"Standard"}]`
	worldtest.Expect(t, "listTargets", reply, worldtest.Reply{Status: 200, Body: `{"targets":[` +
		`{"id":"` + lossless.ID + `","name":"Lossless","tiers":[["remux-1080p","bluray-1080p"],["webdl-1080p"]],"cutoff":"remux-1080p",` +
		`"filters":[{"attribute":"language","mode":"only","values":["en","original"]}],` +
		`"sizes":[{"quality":"remux-1080p","min":102,"max":null},{"quality":"bluray-1080p","min":20,"max":100},{"quality":"webdl-1080p","min":12.5,"max":null}],` +
		`"profiles":` + standard + `,"contains":[],"createdAt":"` + lossless.CreatedAt + `"},` +
		`{"id":"` + small.ID + `","name":"Small","tiers":[["webdl-720p"],["dvd"]],"cutoff":"dvd","filters":[],` +
		`"sizes":[{"quality":"webdl-720p","min":12.5,"max":null},{"quality":"dvd","min":0,"max":null}],` +
		`"profiles":` + standard + `,"contains":[],"createdAt":"` + small.CreatedAt + `"}]}` + "\n"})

	reply = a.Admin("home", http.MethodGet, "/api/v1/profiles", "")
	worldtest.Expect(t, "listProfiles", reply, worldtest.Reply{Status: 200, Body: `{"profiles":[{"id":"` + lossless.Profiles[0].ID +
		`","name":"Standard","contentType":"music","default":true,"targetIds":["` + small.ID + `","` + lossless.ID + `"]}]}` + "\n"})
}
