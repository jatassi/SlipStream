package node

import (
	"regexp"
	"slices"
	"testing"

	"github.com/jatassi/slipstream/internal/node/internal/config"
)

func TestDefaultsAreAValidTree(t *testing.T) {
	cfg := Defaults()
	seen := map[string]string{}
	var paths []string
	for _, leaf := range config.Walk(&cfg) {
		if other, dup := seen[leaf.Env()]; dup {
			t.Fatalf("%s and %s share the environment name %s", other, leaf.Path, leaf.Env())
		}
		seen[leaf.Env()] = leaf.Path
		paths = append(paths, leaf.Path)
	}
	want := []string{"seed", "torrent.listen", "timings.node.stop_deadline"}
	if !slices.Equal(paths, want) {
		t.Fatalf("leaves = %v, want %v", paths, want)
	}

	if _, _, err := LoadConfig(testDataDir(t), nil); err != nil {
		t.Fatalf("defaults do not load: %v", err)
	}
}

func TestLoadConfigDrawsASeedOnlyWhenNoneIsSet(t *testing.T) {
	dir := testDataDir(t)
	cfg, keys, err := LoadConfig(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !regexp.MustCompile(`^[0-9a-f]{32}$`).MatchString(cfg.Seed) {
		t.Fatalf("drawn seed %q is not 32 hex characters", cfg.Seed)
	}
	if got, want := keys[0], (Key{Key: "seed", Value: cfg.Seed, Default: "", Source: config.SourceRandom}); got != want {
		t.Fatalf("seed key = %+v, want %+v", got, want)
	}
	if got := overrides(keys); len(got) != 0 {
		t.Fatalf("overrides = %v, want none", got)
	}

	cfg, keys, err = LoadConfig(dir, []string{"SLIPSTREAM_SEED=fixed"})
	if err != nil {
		t.Fatal(err)
	}
	want := []Key{{Key: "seed", Value: "fixed", Default: "", Source: config.SourceEnv}}
	if cfg.Seed != "fixed" || !slices.Equal(overrides(keys), want) {
		t.Fatalf("seed = %q, overrides = %v, want fixed and %v", cfg.Seed, overrides(keys), want)
	}
}

func testDataDir(t *testing.T) DataDir {
	t.Helper()
	d, err := ParseDataDir(shortTempDir(t))
	if err != nil {
		t.Fatal(err)
	}
	return d
}
