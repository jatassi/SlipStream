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
		Roots:    []scenario.LibraryRoot{{ContentType: "music", Folder: "music"}},
		Targets:  []scenario.LibraryTarget{{Name: "Lossless", Resolutions: []string{"1080p"}}, {Name: "Small", Resolutions: []string{"720p", "480p"}}},
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
			ID        string `json:"id"`
			CreatedAt string `json:"createdAt"`
		} `json:"targets"`
	}
	reply = a.Admin("home", http.MethodGet, "/api/v1/targets", "")
	decode(t, "listTargets", reply, &targets)
	lossless, small := targets.Targets[0], targets.Targets[1]
	worldtest.Expect(t, "listTargets", reply, worldtest.Reply{Status: 200, Body: `{"targets":[` +
		`{"id":"` + lossless.ID + `","name":"Lossless","resolutions":["1080p"],"createdAt":"` + lossless.CreatedAt + `"},` +
		`{"id":"` + small.ID + `","name":"Small","resolutions":["720p","480p"],"createdAt":"` + small.CreatedAt + `"}]}` + "\n"})

	var profiles struct {
		Profiles []struct{ ID string } `json:"profiles"`
	}
	reply = a.Admin("home", http.MethodGet, "/api/v1/profiles", "")
	decode(t, "listProfiles", reply, &profiles)
	worldtest.Expect(t, "listProfiles", reply, worldtest.Reply{Status: 200, Body: `{"profiles":[{"id":"` + profiles.Profiles[0].ID +
		`","name":"Standard","contentType":"music","default":true,"targetIds":["` + small.ID + `","` + lossless.ID + `"]}]}` + "\n"})
}
