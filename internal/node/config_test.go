package node

import (
	"fmt"
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
	want := []string{
		"seed", "torrent.listen", "http.listen", "http.trusted_origins",
		"timings.node.stop_deadline", "timings.storage.busy_timeout", "timings.http.read_header", "timings.http.idle",
		"timings.live.heartbeat", "timings.live.retry",
	}
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

func TestHTTPConfigReadsListenAndTrustedOrigins(t *testing.T) {
	cfg, _, err := LoadConfig(testDataDir(t), []string{
		"SLIPSTREAM_HTTP_LISTEN=127.0.0.1:0",
		"SLIPSTREAM_HTTP_TRUSTED_ORIGINS=https://media.example.com,http://192.0.2.1:8484",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := fmt.Sprint(cfg.HTTP.Listen, cfg.HTTP.TrustedOrigins); got != "127.0.0.1:0 [https://media.example.com http://192.0.2.1:8484]" {
		t.Fatalf("http = %s", got)
	}
	if got := Defaults().HTTP.Listen.String(); got != "[::]:8484" {
		t.Fatalf("default http.listen = %s, want [::]:8484", got)
	}
	_, _, err = LoadConfig(testDataDir(t), []string{"SLIPSTREAM_HTTP_TRUSTED_ORIGINS=https://media.example.com/app"})
	if err == nil || err.Error() != `env: SLIPSTREAM_HTTP_TRUSTED_ORIGINS: item 1: "https://media.example.com/app": not scheme://host[:port]` {
		t.Fatalf("err = %v, want the bad origin named", err)
	}
}
