package world_test

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"go.yaml.in/yaml/v3"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/world"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
	"github.com/jatassi/slipstream/internal/node"
	"github.com/jatassi/slipstream/internal/node/endpoint"
)

func TestEachNodesConfigNamesItsOwnFronts(t *testing.T) {
	t.Parallel()
	movies := echo
	movies.Endpoints = func(e *node.EndpointsBlock, at fake.Address) { e.TMDB.API = endpoint.MustParseURL(at.URL("/3")) }
	s := stubScenario()
	s.Nodes = append(s.Nodes, scenario.Node{Name: "seedbox"})
	ctx := worldtest.Context(t)
	run, err := world.LaunchWith(ctx, world.Options{Binary: stub{}.build(t), Scenario: s, RunDir: filepath.Join(t.TempDir(), "run"), Origin: worldtest.Origin(t)}, []fake.Spec{movies})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = run.Stop(ctx) })
	m := run.Manifest()
	for _, n := range m.Nodes {
		i := slices.IndexFunc(m.Fronts, func(f world.Front) bool { return f.Target == "echo" && f.Node == n.Name })
		if i < 0 {
			t.Fatalf("fronts %+v name none for %s", m.Fronts, n.Name)
		}
		if got, want := tmdbAPI(t, filepath.Join(n.DataDir, "config.yaml")), m.Fronts[i].URL+"/3"; got != want {
			t.Errorf("node %s's endpoints.tmdb.api = %s, want its own front %s", n.Name, got, want)
		}
	}
}

func tmdbAPI(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var config struct {
		Endpoints struct {
			TMDB struct {
				API string `yaml:"api"`
			} `yaml:"tmdb"`
		} `yaml:"endpoints"`
	}
	if err := yaml.Unmarshal(data, &config); err != nil {
		t.Fatal(err)
	}
	return config.Endpoints.TMDB.API
}
