package node

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/contenttype"
	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/install"
	"github.com/jatassi/slipstream/internal/node/internal/config"
)

var noTypes contenttype.Set

func TestDefaultsAreAValidTree(t *testing.T) {
	cfg := Defaults(noTypes, install.Dev)
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
		"seed", "name", "torrent.listen", "torrent.downloads", "torrent.outgoing_utp", "http.listen", "http.trusted_origins", "http.ports_to_try", "http.trusted_proxies", "link.listen", "link.advertise", "link.stream_window", "join.code",
		"log.level", "log.max_size_mb", "log.max_files", "log.max_age_days", "log.compress", "log.viewer_lines",
		"endpoints.tmdb.api", "endpoints.tmdb.images", "endpoints.omdb.api",
		"timings.node.stop_deadline", "timings.node.restart_lock_wait", "timings.storage.busy_timeout", "timings.http.read_header", "timings.http.idle",
		"timings.live.heartbeat", "timings.live.retry",
		"timings.link.handshake", "timings.link.keepalive", "timings.link.write_timeout", "timings.link.redial_min", "timings.link.redial_max", "timings.link.fallback_delay", "timings.link.log_poll", "timings.link.log_archive_ttl",
		"timings.link.join_dial", "timings.link.await_dial",
		"timings.role.restart_min", "timings.role.restart_max", "timings.role.beat", "timings.role.stall", "timings.role.probe", "timings.nodes.seen_persist", "timings.nodes.blip", "timings.nodes.settle", "timings.nodes.dial_invite", "timings.nodes.dial_report", "timings.nodes.traffic", "timings.nodes.reach", "timings.works.fetch", "timings.indexers.request", "timings.transfers.chunk",
		"timings.grabs.retry_min", "timings.grabs.retry_max", "timings.grabs.progress", "timings.files.retry_min", "timings.files.retry_max", "timings.downloading.facts", "timings.media.probe", "timings.mail.send", "timings.serving.drain",
		"timings.storing.stat",
		"timings.metadata.attempts", "timings.metadata.try", "timings.metadata.retry_min", "timings.metadata.retry_max", "timings.metadata.wait", "timings.metadata.cache_entries",
		"timings.tmdb.interval", "timings.tmdb.burst", "timings.omdb.interval", "timings.omdb.burst",
	}
	if !slices.Equal(paths, want) {
		t.Fatalf("leaves = %v, want %v", paths, want)
	}

	if _, _, err := LoadConfig(testDataDir(t), nil, noTypes, install.Dev); err != nil {
		t.Fatalf("defaults do not load: %v", err)
	}
}

func TestLoadConfigDrawsASeedOnlyWhenNoneIsSet(t *testing.T) {
	dir := testDataDir(t)
	cfg, keys, err := LoadConfig(dir, nil, noTypes, install.Dev)
	if err != nil {
		t.Fatal(err)
	}
	if !regexp.MustCompile(`^[0-9a-f]{32}$`).MatchString(cfg.Seed) {
		t.Fatalf("drawn seed %q is not 32 hex characters", cfg.Seed)
	}
	if got, want := keys[0], (Key{Key: "seed", Value: cfg.Seed, Default: "", Source: config.SourceRandom, Env: "SLIPSTREAM_SEED"}); got != want {
		t.Fatalf("seed key = %+v, want %+v", got, want)
	}
	if got := overrides(keys); len(got) != 0 {
		t.Fatalf("overrides = %v, want none", got)
	}

	cfg, keys, err = LoadConfig(dir, []string{"SLIPSTREAM_SEED=fixed"}, noTypes, install.Dev)
	if err != nil {
		t.Fatal(err)
	}
	want := []Key{{Key: "seed", Value: "fixed", Default: "", Source: config.SourceEnv, Env: "SLIPSTREAM_SEED"}}
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
	}, noTypes, install.Dev)
	if err != nil {
		t.Fatal(err)
	}
	if got := fmt.Sprint(cfg.HTTP.Listen, cfg.HTTP.TrustedOrigins); got != "127.0.0.1:0 [https://media.example.com http://192.0.2.1:8484]" {
		t.Fatalf("http = %s", got)
	}
	if got := Defaults(noTypes, install.Dev).HTTP.Listen.String(); got != "[::]:8484" {
		t.Fatalf("default http.listen = %s, want [::]:8484", got)
	}
	_, _, err = LoadConfig(testDataDir(t), []string{"SLIPSTREAM_HTTP_TRUSTED_ORIGINS=https://media.example.com,https://Media.Example.com:443/"}, noTypes, install.Dev)
	if err == nil || err.Error() != `env: SLIPSTREAM_HTTP_TRUSTED_ORIGINS: item 2: "https://Media.Example.com:443/": not a canonical origin; write https://media.example.com` {
		t.Fatalf("err = %v, want the bad origin named", err)
	}
}

const goldenCode = "SLP1-CC6WT-AB6RG-SPD48-JAH4B-QCJV9-XJ6AS-K7D1M-PMTVC-DNQ6Y-W3HE9-SG213-Z0000-58D1S-QG852-G"

func TestAJoinCodeIsAlwaysRedacted(t *testing.T) {
	cfg, keys, err := LoadConfig(testDataDir(t), []string{"SLIPSTREAM_JOIN_CODE=" + strings.ToLower(goldenCode)}, noTypes, install.Dev)
	if err != nil {
		t.Fatal(err)
	}
	want := []Key{{Key: "join.code", Value: "SLP1-[redacted]", Default: "", Source: config.SourceEnv, Env: "SLIPSTREAM_JOIN_CODE"}}
	if cfg.Join.Code.String() != goldenCode || !slices.Equal(overrides(keys), want) {
		t.Fatalf("join.code = %s, overrides %v; want the code read and %v", cfg.Join.Code, overrides(keys), want)
	}

	damaged := goldenCode[:len(goldenCode)-1] + "H"
	_, _, err = LoadConfig(testDataDir(t), []string{"SLIPSTREAM_JOIN_CODE=" + damaged}, noTypes, install.Dev)
	if want := "env: SLIPSTREAM_JOIN_CODE: the Join Code is mistyped or cut short"; err == nil || err.Error() != want || ExitCode(err) != ExitUsage {
		t.Fatalf("err = %v, want %q with exit %d", err, want, ExitUsage)
	}
}

func TestTheLinkAndRoleSectionsHaveTheirDefaults(t *testing.T) {
	cfg := Defaults(noTypes, install.Dev)
	got := fmt.Sprint(cfg.Link, cfg.Timings.Link, cfg.Timings.Role)
	if want := "{[::]:8485 [] 16777216} {10s 25s 10s 1s 1m0s 300ms 20s 10m0s 20s 10m0s} {1s 1m0s 15s 45s 10m0s}"; got != want {
		t.Fatalf("link, timings.link, timings.role = %s, want %s", got, want)
	}
	if host, _ := os.Hostname(); cfg.Name != host {
		t.Fatalf("name = %q, want the host name %q", cfg.Name, host)
	}
}

func TestLinkAdvertiseIsAPeerAddressRecord(t *testing.T) {
	cfg, keys, err := LoadConfig(testDataDir(t), []string{"SLIPSTREAM_LINK_ADVERTISE=203.0.113.5:443,[2001:db8::5]:443"}, noTypes, install.Dev)
	if err != nil {
		t.Fatal(err)
	}
	want := []Key{{Key: "link.advertise", Value: "203.0.113.5:443,[2001:db8::5]:443", Default: "", Source: config.SourceEnv, Env: "SLIPSTREAM_LINK_ADVERTISE"}}
	if got := fmt.Sprint(cfg.Link.Advertise); got != "[203.0.113.5:443 [2001:db8::5]:443]" || !slices.Equal(overrides(keys), want) {
		t.Fatalf("link.advertise = %s, overrides %v; want both addresses and %v", got, overrides(keys), want)
	}

	dir := testDataDir(t)
	if err := os.WriteFile(filepath.Join(dir.Path(), "config.yaml"), []byte("link:\n  advertise:\n    - 203.0.113.5:0\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, _, err = LoadConfig(dir, nil, noTypes, install.Dev)
	if want := "config.yaml:2: link.advertise: address 203.0.113.5:0 is not a literal IP and port"; err == nil || err.Error() != want || ExitCode(err) != ExitUsage {
		t.Fatalf("err = %v, want %q with exit %d", err, want, ExitUsage)
	}

	nine := "SLIPSTREAM_LINK_ADVERTISE=192.0.2.1:1,192.0.2.2:1,192.0.2.3:1,192.0.2.4:1,192.0.2.5:1,192.0.2.6:1,192.0.2.7:1,192.0.2.8:1,192.0.2.9:1"
	_, _, err = LoadConfig(testDataDir(t), []string{nine}, noTypes, install.Dev)
	if want := "env: link.advertise: a peer address record holds 1 to 8 addresses, not 9"; err == nil || err.Error() != want {
		t.Fatalf("err = %v, want %q", err, want)
	}
}

func fixtureTypes() contenttype.Set { return contenttype.NewSet(contenttypetest.New().Def()) }

func TestAContentTypesEndpointsAreKeysOfTheirOwn(t *testing.T) {
	types := fixtureTypes()
	cfg := Defaults(types, install.Dev)
	var paths []string
	for _, leaf := range config.Walk(&cfg) {
		if strings.HasPrefix(leaf.Path, "endpoints.content_types.") {
			paths = append(paths, leaf.Path+" "+leaf.Env())
		}
	}
	if want := []string{"endpoints.content_types.fixture.api SLIPSTREAM_ENDPOINTS_CONTENT_TYPES_FIXTURE_API"}; !slices.Equal(paths, want) {
		t.Fatalf("endpoints leaves = %q, want %q", paths, want)
	}

	env := Env{DataDir: testDataDir(t), Types: types}
	resolved, err := Resolve(env)
	if err != nil {
		t.Fatal(err)
	}
	key := Key{Key: "endpoints.content_types.fixture.api", Value: "https://fixture.invalid/v1", Default: "https://fixture.invalid/v1", Source: config.SourceDefault, Env: "SLIPSTREAM_ENDPOINTS_CONTENT_TYPES_FIXTURE_API"}
	if !slices.Contains(resolved.Keys, key) || !slices.Equal(resolved.Census, []string{"endpoints.tmdb.api", "endpoints.tmdb.images", "endpoints.omdb.api", "endpoints.content_types.fixture.api"}) || !slices.Equal(resolved.ContentTypes, []string{"fixture"}) {
		t.Fatalf("resolved = %+v; want %+v, the key in the census and contentTypes [fixture]", resolved, key)
	}

	env.Environ = []string{"SLIPSTREAM_ENDPOINTS_CONTENT_TYPES_FIXTURE_API=https://127.0.0.1:8443/v1"}
	resolved, err = Resolve(env)
	if err != nil {
		t.Fatal(err)
	}
	key = Key{Key: "endpoints.content_types.fixture.api", Value: "https://127.0.0.1:8443/v1", Default: "https://fixture.invalid/v1", Source: config.SourceEnv, Env: "SLIPSTREAM_ENDPOINTS_CONTENT_TYPES_FIXTURE_API"}
	if !slices.Contains(resolved.Keys, key) || !slices.Equal(resolved.Census, []string{"endpoints.tmdb.api", "endpoints.tmdb.images", "endpoints.omdb.api"}) {
		t.Fatalf("resolved = %+v; want %+v and only the core endpoints in the census", resolved, key)
	}
	cfg, _, err = LoadConfig(env.DataDir, env.Environ, types, install.Dev)
	if err != nil {
		t.Fatal(err)
	}
	if got, ok := cfg.Endpoints.ContentTypes.section(contenttypetest.ID).(*contenttypetest.Endpoints); !ok || got.API.String() != "https://127.0.0.1:8443/v1" {
		t.Fatalf("loaded section = %v, want api at the env value", got)
	}
}

func TestAnUnregisteredContentTypesKeyIsUnknown(t *testing.T) {
	types := fixtureTypes()
	dir := testDataDir(t)
	if err := os.WriteFile(filepath.Join(dir.Path(), "config.yaml"), []byte("endpoints:\n  content_types:\n    fixture:\n      api: https://127.0.0.1:1/\n    other:\n      url: https://127.0.0.1:2/\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, _, err := LoadConfig(dir, []string{"SLIPSTREAM_ENDPOINTS_CONTENT_TYPES_OTHER_URL=https://127.0.0.1:3/"}, types, install.Dev)
	want := "config.yaml:5: endpoints.content_types.other: unknown key\nenv: SLIPSTREAM_ENDPOINTS_CONTENT_TYPES_OTHER_URL: unknown key"
	if err == nil || err.Error() != want || ExitCode(err) != ExitUsage {
		t.Fatalf("err = %v, want %q with exit %d", err, want, ExitUsage)
	}
	_, _, err = LoadConfig(dir, nil, noTypes, install.Dev)
	if want := "config.yaml:2: endpoints.content_types: unknown key"; err == nil || err.Error() != want {
		t.Fatalf("with no types, err = %v, want %q", err, want)
	}
}
