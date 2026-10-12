package config_test

import (
	"net/netip"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/node/endpoint"
	"github.com/jatassi/slipstream/internal/node/internal/config"
)

type tree struct {
	Name      string           `yaml:"name"`
	Retries   int              `yaml:"retries"`
	Verbose   bool             `yaml:"verbose"`
	Peers     []netip.AddrPort `yaml:"peers"`
	Endpoints struct {
		Tracker endpoint.URL        `yaml:"tracker"`
		STUN    []endpoint.HostPort `yaml:"stun"`
	} `yaml:"endpoints"`
	Timings struct {
		Link struct {
			Heartbeat time.Duration `yaml:"heartbeat"`
		} `yaml:"link"`
	} `yaml:"timings"`
}

func defaults() tree {
	var t tree
	t.Name = "default"
	t.Retries = 3
	t.Endpoints.Tracker = endpoint.MustParseURL("https://tracker.example.org/announce")
	t.Endpoints.STUN = []endpoint.HostPort{endpoint.MustParseHostPort("stun.example.net:3478")}
	t.Timings.Link.Heartbeat = 10 * time.Second
	return t
}

func writeFile(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func load(t *testing.T, dir string, environ ...string) (tree, []config.Key, error) {
	t.Helper()
	cfg := defaults()
	keys, err := config.Load(&cfg, dir, environ)
	return cfg, keys, err
}

func keyOf(t *testing.T, keys []config.Key, path string) config.Key {
	t.Helper()
	i := slices.IndexFunc(keys, func(k config.Key) bool { return k.Key == path })
	if i < 0 {
		t.Fatalf("no key %s in %v", path, keys)
	}
	return keys[i]
}

func TestLayersOverrideInOrder(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "config.yaml", "name: from-yaml\n")
	writeFile(t, dir, ".env", "SLIPSTREAM_NAME=from-dotenv\n")
	environ := []string{"SLIPSTREAM_NAME=from-env", "PATH=/bin"}

	steps := []struct {
		remove string
		want   config.Key
	}{
		{"", config.Key{Key: "name", Value: "from-env", Default: "default", Source: config.SourceEnv}},
		{"env", config.Key{Key: "name", Value: "from-dotenv", Default: "default", Source: config.SourceDotEnv}},
		{".env", config.Key{Key: "name", Value: "from-yaml", Default: "default", Source: config.SourceYAML}},
		{"config.yaml", config.Key{Key: "name", Value: "default", Default: "default", Source: config.SourceDefault}},
	}
	for _, step := range steps {
		switch step.remove {
		case "":
		case "env":
			environ = nil
		default:
			if err := os.Remove(filepath.Join(dir, step.remove)); err != nil {
				t.Fatal(err)
			}
		}
		cfg, keys, err := load(t, dir, environ...)
		if err != nil {
			t.Fatalf("after removing %q: %v", step.remove, err)
		}
		if got := keyOf(t, keys, "name"); got != step.want {
			t.Fatalf("after removing %q: key = %+v, want %+v", step.remove, got, step.want)
		}
		if cfg.Name != step.want.Value {
			t.Fatalf("after removing %q: cfg.Name = %q, want %q", step.remove, cfg.Name, step.want.Value)
		}
	}
}

func TestLeafTypesDecodeFromEveryLayer(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "config.yaml", `retries: 7
peers:
  - 127.0.0.1:1
  - "[::1]:2"
endpoints:
  tracker: http://127.0.0.1:9000/announce
timings:
  link:
    heartbeat: 250ms
`)
	writeFile(t, dir, ".env", "# fakes\r\n\r\nSLIPSTREAM_VERBOSE=true\r\n")
	cfg, keys, err := load(t, dir, "SLIPSTREAM_ENDPOINTS_STUN=127.0.0.1:3478,localhost:3479")
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, k := range keys {
		got[k.Key] = k.Value + " " + string(k.Source)
	}
	want := map[string]string{
		"name":                   "default default",
		"retries":                "7 config.yaml",
		"verbose":                "true .env",
		"peers":                  "127.0.0.1:1,[::1]:2 config.yaml",
		"endpoints.tracker":      "http://127.0.0.1:9000/announce config.yaml",
		"endpoints.stun":         "127.0.0.1:3478,localhost:3479 env",
		"timings.link.heartbeat": "250ms config.yaml",
	}
	if !mapsEqual(got, want) {
		t.Fatalf("keys = %v, want %v", got, want)
	}
	if cfg.Timings.Link.Heartbeat != 250*time.Millisecond || cfg.Peers[1] != netip.MustParseAddrPort("[::1]:2") {
		t.Fatalf("cfg = %+v", cfg)
	}
}

func mapsEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

func TestEveryProblemIsReportedTogether(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "config.yaml", `name: x
nam: y
timings:
  link:
    heartbeat: 5
    hartbeat: 1s
name: z
endpoints: 3
`)
	writeFile(t, dir, ".env", "SLIPSTREAM_RETRIES=many\nRETRIES=4\nnonsense\nSLIPSTREAM_DATA_DIR=/elsewhere\n")
	_, _, err := load(t, dir,
		"SLIPSTREAM_VERBSE=true",
		"SLIPSTREAM_DATA_DIR=/data",
		"SLIPSTREAM_TIMINGS_LINK_HEARTBEAT=-1s",
		"SLIPSTREAM_PEERS=127.0.0.1",
	)
	want := `config.yaml:2: nam: unknown key
config.yaml:5: timings.link.heartbeat: not a duration: time: missing unit in duration "5"
config.yaml:6: timings.link.hartbeat: unknown key
config.yaml:7: name: duplicate key, first set on line 1
config.yaml:8: endpoints: expected a mapping
.env:1: SLIPSTREAM_RETRIES: not an integer: strconv.ParseInt: parsing "many": invalid syntax
.env:2: RETRIES: missing the SLIPSTREAM_ prefix
.env:3: nonsense: not a KEY=value line
env: SLIPSTREAM_VERBSE: unknown key
env: SLIPSTREAM_PEERS: item 1: "127.0.0.1": not an ip:port
env: timings.link.heartbeat: duration -1s is not positive`
	if err == nil || err.Error() != want {
		t.Fatalf("err =\n%v\nwant\n%s", err, want)
	}
}

func TestEndpointsMustNameAHost(t *testing.T) {
	_, _, err := load(t, t.TempDir(), "SLIPSTREAM_ENDPOINTS_STUN=", "SLIPSTREAM_ENDPOINTS_TRACKER=/announce")
	want := `env: SLIPSTREAM_ENDPOINTS_TRACKER: "/announce": not an absolute URL with a scheme and a host
env: endpoints.stun: endpoint list is empty`
	if err == nil || err.Error() != want {
		t.Fatalf("err =\n%v\nwant\n%s", err, want)
	}

	var empty tree
	_, err = config.Load(&empty, t.TempDir(), nil)
	want = `default: endpoints.tracker: endpoint is empty
default: endpoints.stun: endpoint list is empty
default: timings.link.heartbeat: duration 0s is not positive`
	if err == nil || err.Error() != want {
		t.Fatalf("err =\n%v\nwant\n%s", err, want)
	}
}

func TestWalkNamesLeavesByPath(t *testing.T) {
	cfg := defaults()
	var got []string
	for _, l := range config.Walk(&cfg) {
		got = append(got, l.Path+"="+l.String()+" "+l.Env())
	}
	want := []string{
		"name=default SLIPSTREAM_NAME",
		"retries=3 SLIPSTREAM_RETRIES",
		"verbose=false SLIPSTREAM_VERBOSE",
		"peers= SLIPSTREAM_PEERS",
		"endpoints.tracker=https://tracker.example.org/announce SLIPSTREAM_ENDPOINTS_TRACKER",
		"endpoints.stun=stun.example.net:3478 SLIPSTREAM_ENDPOINTS_STUN",
		"timings.link.heartbeat=10s SLIPSTREAM_TIMINGS_LINK_HEARTBEAT",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("leaves =\n%v\nwant\n%v", got, want)
	}
}

func TestWalkPanicsOutsideTheLeafSet(t *testing.T) {
	defer func() {
		if r := recover(); r != "config: odd.ratio has type float64, outside the leaf set" {
			t.Fatalf("recovered %v", r)
		}
	}()
	var bad struct {
		Odd struct {
			Ratio float64 `yaml:"ratio"`
		} `yaml:"odd"`
	}
	config.Walk(&bad)
}
