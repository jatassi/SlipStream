package e2e

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"slices"
	"testing"

	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/world"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
	"github.com/jatassi/slipstream/internal/node"
)

type reply struct {
	status int
	body   string
}

func get(ctx context.Context, t *testing.T, target, apiKey string) reply {
	t.Helper()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	if apiKey != "" {
		req.Header.Set("X-Api-Key", apiKey)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	return reply{res.StatusCode, string(body)}
}

func expect(t *testing.T, what string, got, want reply) {
	t.Helper()
	if got != want {
		t.Fatalf("%s = %d %q, want %d %q", what, got.status, got.body, want.status, want.body)
	}
}

func TestFirstRun(t *testing.T) {
	t.Parallel()
	run := worldtest.Launch(t, "first-run")
	m := run.Manifest()
	worldtest.SetupCode(t, run, "home")
	home := m.Nodes[0]
	if home.Setup != scenario.SetupNone || home.Claim != world.ClaimUnclaimed || m.Admin != nil || m.APIKey != "" {
		t.Fatalf("manifest: setup %s, claim %s, admin %+v, apiKey %q; want an Unclaimed Node and no Admin", home.Setup, home.Claim, m.Admin, m.APIKey)
	}
	ctx := worldtest.Context(t)
	expect(t, "GET /api/v1/status", get(ctx, t, home.URL+"/api/v1/status", ""), reply{200, `{"claim":"unclaimed"}` + "\n"})
}

func TestSingleMachine(t *testing.T) {
	t.Parallel()
	run := worldtest.Launch(t, "single-machine")
	m := run.Manifest()
	home := m.Nodes[0]
	ctx := worldtest.Context(t)
	if !regexp.MustCompile(`^ssk_[A-Za-z0-9_-]{43}$`).MatchString(m.APIKey) {
		t.Fatalf("apiKey = %q", m.APIKey)
	}

	sys := get(ctx, t, home.URL+"/api/v1/system", m.APIKey)
	var report struct {
		DataDir   string `json:"dataDir"`
		PID       int    `json:"pid"`
		Seed      string `json:"seed"`
		Listeners struct {
			HTTP string `json:"http"`
		} `json:"listeners"`
	}
	if err := json.Unmarshal([]byte(sys.body), &report); err != nil || sys.status != http.StatusOK {
		t.Fatalf("GET /api/v1/system with the API key = %d %s, want 200", sys.status, sys.body)
	}
	if report.DataDir != home.DataDir || report.PID != home.PID || report.Seed != "single-machine/home" || report.Listeners.HTTP != home.Listeners["http"] {
		t.Fatalf("getSystem = %s\nwant dataDir %s, pid %d, seed single-machine/home, listeners.http %s", sys.body, home.DataDir, home.PID, home.Listeners["http"])
	}
	if addr := netip.MustParseAddrPort(report.Listeners.HTTP).Addr(); addr != netip.MustParseAddr("127.0.0.2") {
		t.Fatalf("listeners.http on %s, want 127.0.0.2", addr)
	}
	expect(t, "GET /api/v1/system without a key", get(ctx, t, home.URL+"/api/v1/system", ""), reply{401, `{"status":401,"code":"signed_out","title":"Sign in to continue"}` + "\n"})
	expect(t, "GET /api/v1/status", get(ctx, t, home.URL+"/api/v1/status", ""), reply{200, `{"claim":"controller"}` + "\n"})

	want := node.Key{Key: "timings.node.stop_deadline", Value: "5s", Default: "30s", Source: "config.yaml"}
	if !slices.Contains(home.Overrides, want) {
		t.Fatalf("overrides = %+v, want %+v among them", home.Overrides, want)
	}
	if census := resolve(ctx, t, m.Binary.Path, home.DataDir).Census; len(census) != 0 {
		t.Fatalf("census = %v, want it empty", census)
	}
}

// resolve runs slipstream config on a Node's data dir, as the launch did.
func resolve(ctx context.Context, t *testing.T, binary, dataDir string) node.Resolved {
	t.Helper()
	cmd := exec.CommandContext(ctx, binary, "config", "--data-dir", dataDir)
	cmd.Env = []string{"SystemRoot=" + os.Getenv("SystemRoot")}
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("slipstream config: %v", err)
	}
	var r node.Resolved
	if err := json.Unmarshal(out, &r); err != nil {
		t.Fatal(err)
	}
	return r
}

func TestTwoNodesOwnAddresses(t *testing.T) {
	t.Parallel()
	run := worldtest.LaunchScenario(t, scenario.Scenario{
		Name:  "two-nodes",
		Seed:  "two-nodes",
		Nodes: []scenario.Node{{Name: "a"}, {Name: "b"}},
	})
	type want struct{ name, addr, host string }
	for i, w := range []want{{"a", "127.0.0.2", "a.localhost"}, {"b", "127.0.0.3", "b.localhost"}} {
		n := run.Manifest().Nodes[i]
		if n.Name != w.name || n.Address != w.addr || n.Host != w.host {
			t.Fatalf("node %d = %s on %s at %s, want %s on %s at %s", i, n.Name, n.Address, n.Host, w.name, w.addr, w.host)
		}
		for _, listener := range []string{"http", "torrent"} {
			if addr := netip.MustParseAddrPort(n.Listeners[listener]).Addr().String(); addr != w.addr {
				t.Fatalf("node %s listener %s on %s, want %s", n.Name, listener, addr, w.addr)
			}
		}
		u, err := url.Parse(n.URL)
		if err != nil {
			t.Fatal(err)
		}
		if wantBrowser := "http://" + w.host + ":" + u.Port(); n.BrowserURL != wantBrowser {
			t.Fatalf("node %s browserUrl = %s, want %s", n.Name, n.BrowserURL, wantBrowser)
		}
	}
}
