package world_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/netip"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"go.yaml.in/yaml/v3"

	"github.com/jatassi/slipstream/fakes/events"
	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/internal/nodeenv"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/testbin"
	"github.com/jatassi/slipstream/fakes/world"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
	"github.com/jatassi/slipstream/internal/node"
	"github.com/jatassi/slipstream/web"
)

func TestMain(m *testing.M) {
	os.Exit(testbin.Run(m))
}

type stub struct {
	resolved node.Resolved
	torrent  string
	runExit  string
	// successor restarts the stub as Windows does.
	successor bool
	noTools   bool
}

func productionKeys() []node.Key {
	return []node.Key{
		{Key: "seed", Value: "stub/home", Default: "", Source: "config.yaml"},
		{Key: "torrent.listen", Value: "127.0.0.2:0", Default: "0.0.0.0:6881", Source: "config.yaml"},
		{Key: "http.listen", Value: "127.0.0.2:0", Default: "[::]:8484", Source: "config.yaml"},
		{Key: "link.listen", Value: "127.0.0.2:0", Default: "[::]:8485", Source: "config.yaml"},
		{Key: "timings.node.stop_deadline", Value: "5s", Default: "30s", Source: "config.yaml"},
	}
}

func (s stub) build(t *testing.T) string {
	t.Helper()
	r := s.resolved
	if r.Keys == nil {
		r = node.Resolved{Keys: productionKeys(), Census: []string{}, ContentTypes: []string{}}
	}
	resolved, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	flags := "-X=main.resolved=" + string(resolved) + " -X=main.torrent=" + s.torrent + " -X=main.runExit=" + s.runExit
	if s.successor {
		flags += " -X=main.successor=1"
	}
	bin := testbin.Build(t, t.TempDir(), "./fakes/world/testdata/stubnode", "-ldflags", flags)
	if s.noTools {
		return bin
	}
	for _, tool := range []string{"ffmpeg", "ffprobe"} {
		if err := os.WriteFile(filepath.Join(filepath.Dir(bin), tool+filepath.Ext(bin)), nil, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return bin
}

func stubScenario() scenario.Scenario {
	return scenario.Scenario{
		Name:    "stub",
		Seed:    "stub",
		Nodes:   []scenario.Node{{Name: "home"}},
		Timings: node.TimingsBlock{Node: node.Timings{StopDeadline: 5 * time.Second}},
	}
}

var echo = fake.Spec{
	Name: "echo",
	TLS:  fake.FrontTLS,
	New: func(_ scenario.Scenario, env fake.Env) (fake.Server, error) {
		return fake.HTTP(env, fake.Routes{"echo": {Pattern: "/", Handler: func(w http.ResponseWriter, r *http.Request) {
			_, _ = io.WriteString(w, "echo to "+fake.OriginOf(r.Context())+"\n")
		}}}), nil
	},
	Endpoints: func(*node.EndpointsBlock, fake.Address) {},
	Online:    fake.NoOnlineRun("a test fixture serves no real service"),
}

func TestLaunchAndStopOnTheStub(t *testing.T) {
	t.Parallel()
	bin := stub{}.build(t)
	runDir := filepath.Join(t.TempDir(), "run")
	ctx := worldtest.Context(t)
	run, err := world.LaunchWith(ctx, world.Options{Binary: bin, Scenario: stubScenario(), RunDir: runDir, Origin: worldtest.Origin(t)}, []fake.Spec{echo})
	if err != nil {
		t.Fatal(err)
	}
	m := run.Manifest()
	checkLaunched(t, m, runDir, bin)
	if got := getFake(ctx, t, m); got != "echo to world\n" {
		t.Fatalf("GET the echo fake = %q", got)
	}
	if _, err := events.Wait(ctx, runDir, events.Query{Kind: "front.close", Node: "world"}); err != nil {
		t.Fatal(err)
	}
	if got := get(ctx, t, m, m.Fronts[0].URL+"/"); got != "echo to home\n" {
		t.Fatalf("GET the echo fake through home's front = %q", got)
	}
	if _, err := events.Wait(ctx, runDir, events.Query{Kind: "front.close", Node: "home"}); err != nil {
		t.Fatal(err)
	}
	var fronts []string
	for _, l := range logged(t, runDir)[4:] {
		var d struct {
			Conn  int
			Cause string
		}
		f := strings.SplitN(l, " ", 4)
		if err := json.Unmarshal([]byte(f[3]), &d); err != nil {
			t.Fatal(err)
		}
		fronts = append(fronts, fmt.Sprintf("%s %s %d %s", f[1], f[2], d.Conn, d.Cause))
	}
	if want := []string{"front.open world 1 ", "front.close world 1 closed", "front.open home 2 ", "front.close home 2 closed"}; !slices.Equal(fronts, want) {
		t.Fatalf("front records = %q, want %q", fronts, want)
	}

	report, err := world.StopRun(ctx, runDir)
	if err != nil {
		t.Fatal(err)
	}
	want := world.StopReport{Nodes: []world.NodeStop{{Name: "home", ExitCode: 0}}}
	if !reflect.DeepEqual(report, want) {
		t.Fatalf("StopRun = %+v, want %+v", report, want)
	}
	<-run.Done()
	if again, err := run.Stop(worldtest.Context(t)); err != nil || !reflect.DeepEqual(again, want) {
		t.Fatalf("Stop after StopRun = %+v, %v; want %+v", again, err, want)
	}
	if again, err := world.StopRun(ctx, runDir); err != nil || !reflect.DeepEqual(again, want) {
		t.Fatalf("StopRun on a stopped run = %+v, %v; want %+v", again, err, want)
	}
	launched := logged(t, runDir)[:8]
	wantStopped := slices.Concat(launched, []string{
		`9 run.stopping world {}`,
		`10 node.error home {"level":"ERROR","msg":"stub node told to stop","detail":"logged verbatim"}`,
		fmt.Sprintf(`11 node.exited home {"pid":%d,"code":0}`, m.Nodes[0].PID),
		`12 run.stopped world {"clean":true}`,
	})
	if got := logged(t, runDir); !reflect.DeepEqual(got, wantStopped) {
		t.Fatalf("events.jsonl after stop holds\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(wantStopped, "\n"))
	}
	stopped, err := os.ReadFile(filepath.Join(runDir, "stopped.json"))
	if err != nil {
		t.Fatal(err)
	}
	if want := "{\n  \"nodes\": [\n    {\n      \"name\": \"home\",\n      \"exitCode\": 0,\n      \"exitedUnasked\": false\n    }\n  ]\n}\n"; string(stopped) != want {
		t.Fatalf("stopped.json =\n%s\nwant\n%s", stopped, want)
	}
	assertStopped(t, m)
}

func checkLaunched(t *testing.T, m world.Manifest, runDir, bin string) {
	t.Helper()
	written, err := world.ReadManifest(runDir)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(written, m) {
		t.Fatalf("manifest.json = %+v, want %+v", written, m)
	}
	validateManifest(t, filepath.Join(runDir, "manifest.json"))

	home := m.Nodes[0]
	scratch := filepath.Join(os.TempDir(), "ssfw", m.RunID)
	pinned := pinnedListeners(t, filepath.Join(home.DataDir, "config.yaml"))
	httpPort := netip.MustParseAddrPort(pinned["http"]).Port()
	wantHome := world.Node{
		Name: "home", Setup: scenario.SetupNone, Claim: world.ClaimUnclaimed,
		Address: "127.0.0.2", Host: "home.localhost",
		URL: fmt.Sprintf("http://127.0.0.2:%d", httpPort), BrowserURL: fmt.Sprintf("http://home.localhost:%d", httpPort),
		Listeners: map[string]string{"control": filepath.Join(scratch, "home", "run", "control.sock"), "http": pinned["http"], "link": pinned["link"], "torrent": pinned["torrent"]},
		DataDir:   filepath.Join(scratch, "home"), State: world.NodeRunning, PID: home.PID, Seed: "stub/home",
		Log: filepath.Join(runDir, "nodes", "home.log"), Overrides: []node.Key{}, NodeKey: "0000-0000-0000-0000",
	}
	if !reflect.DeepEqual(home, wantHome) {
		t.Fatalf("manifest node = %+v\nwant %+v", home, wantHome)
	}
	for name, addr := range pinned {
		if ap := netip.MustParseAddrPort(addr); ap.Addr() != netip.MustParseAddr("127.0.0.2") || ap.Port() == 0 {
			t.Fatalf("config.yaml pins %s.listen at %s, want the port it bound on 127.0.0.2", name, addr)
		}
	}
	wantFronts := []world.Front{
		{Target: "echo", Node: "home", Address: strings.TrimPrefix(m.Fronts[0].URL, "https://"), URL: m.Fronts[0].URL},
		{Target: "echo", Node: "world", Address: strings.TrimPrefix(m.Fakes[0].URL, "https://"), URL: m.Fakes[0].URL},
		{Target: "link/home", Node: "", Address: m.Links["home"]},
	}
	if !reflect.DeepEqual(m.Fronts, wantFronts) || m.Fronts[0].Address == m.Fronts[1].Address || !strings.HasPrefix(m.Fronts[0].Address, "127.0.0.1:") {
		t.Fatalf("fronts = %+v, want echo's for home and the world and home's link front on ports of 127.0.0.1", m.Fronts)
	}
	if advertised := advertisedLink(t, filepath.Join(home.DataDir, "config.yaml")); len(m.Links) != 1 || !strings.HasPrefix(m.Links["home"], "127.0.0.1:") || !slices.Equal(advertised, []string{m.Links["home"]}) {
		t.Fatalf("links = %v and home's config.yaml advertises %v, want home's link front advertised", m.Links, advertised)
	}
	if want := []world.Fake{{Name: "echo", URL: m.Fronts[1].URL, Captures: []fake.CaptureSpec{}, Cues: []fake.CueSpec{}}}; !reflect.DeepEqual(m.Fakes, want) || len(m.Releases) != 0 {
		t.Fatalf("fakes = %+v and releases %+v, want %+v and none", m.Fakes, m.Releases, want)
	}
	wantBrowser := world.Browser{
		HostResolverRules: "MAP home.localhost 127.0.0.2",
		Config:            map[string]string{"desktop": "browser/desktop/playwright-cli.json", "phone": "browser/phone/playwright-cli.json"},
		Presets:           web.Presets(),
	}
	if !reflect.DeepEqual(m.Browser, wantBrowser) {
		t.Fatalf("browser = %+v, want %+v", m.Browser, wantBrowser)
	}
	if want := []string{"loopback-listen", "seed", "timings", "test-ca"}; !reflect.DeepEqual(m.Divergences, want) {
		t.Fatalf("divergences = %v, want %v", m.Divergences, want)
	}
	if m.ScratchDir != scratch || m.Binary.Version != "stub" || m.Binary.Path != filepath.Join(runDir, "bin", "slipstream"+filepath.Ext(bin)) || m.Admin != nil || m.APIKey != "" {
		t.Fatalf("manifest = %+v", m)
	}
	if written, err := os.ReadFile(filepath.Join(runDir, "scenario.yaml")); err != nil || string(written) != "description: \"\"\nseed: stub\nnodes:\n  - name: home\n    setup: none\ntimings:\n  node:\n    stop_deadline: 5s\n" {
		t.Fatalf("scenario.yaml = %q, %v", written, err)
	}
	want := []string{
		fmt.Sprintf(`1 run.launched world {"runId":%q,"pid":%d,"control":%q,"scratchDir":%q,"scenario":"stub","origin":{"kind":"test","command":"go test ./fakes/world -run '^TestLaunchAndStopOnTheStub$' -count=1"}}`, m.RunID, os.Getpid(), m.Supervisor.Control, m.ScratchDir),
		fmt.Sprintf(`2 node.spawned home {"pid":%d,"dataDir":%q}`, home.PID, home.DataDir),
		fmt.Sprintf(`3 node.ready home {"pid":%d}`, home.PID),
		`4 run.ready world {"manifest":"manifest.json"}`,
	}
	if got := logged(t, runDir); !reflect.DeepEqual(got, want) {
		t.Fatalf("events.jsonl holds\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

func logged(t *testing.T, runDir string) []string {
	t.Helper()
	recs, err := events.Read(runDir)
	if err != nil {
		t.Fatal(err)
	}
	out := make([]string, len(recs))
	for i, r := range recs {
		out[i] = fmt.Sprintf("%d %s %s %s", r.Seq, r.Kind, r.Node, r.Data)
		if r.Files != nil {
			out[i] += fmt.Sprint(" ", r.Files)
		}
	}
	return out
}

// besideFronts is the run's records, but its fronts', as "kind node data".
func besideFronts(t *testing.T, runDir string) []string {
	t.Helper()
	var out []string
	for _, line := range logged(t, runDir) {
		_, rest, _ := strings.Cut(line, " ")
		if !strings.HasPrefix(rest, "front.") {
			out = append(out, rest)
		}
	}
	return out
}

func getFake(ctx context.Context, t *testing.T, m world.Manifest) string {
	t.Helper()
	if len(m.Fakes) != 1 || m.Fakes[0].Name != "echo" || !strings.HasPrefix(m.Fakes[0].URL, "https://127.0.0.1:") {
		t.Fatalf("fakes = %+v", m.Fakes)
	}
	return get(ctx, t, m, m.Fakes[0].URL+"/")
}

// get GETs url trusting the run's CA, and closes the connection.
func get(ctx context.Context, t *testing.T, m world.Manifest, url string) string {
	t.Helper()
	pem, err := os.ReadFile(m.CA.File)
	if err != nil {
		t.Fatal(err)
	}
	roots := x509.NewCertPool()
	roots.AppendCertsFromPEM(pem)
	client := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12}}}
	defer client.CloseIdleConnections()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	res, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

func assertStopped(t *testing.T, m world.Manifest) {
	t.Helper()
	for _, n := range m.Nodes {
		if n.PID != 0 && !worldtest.ProcessGone(n.PID) {
			t.Errorf("node %s (PID %d) still runs", n.Name, n.PID)
		}
	}
	for _, path := range []string{m.ScratchDir, m.Supervisor.Control, filepath.Join(m.RunDir, "manifest.json")} {
		if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("%s after stop: %v, want it gone", path, err)
		}
	}
	if _, err := os.Stat(filepath.Join(m.RunDir, "manifest.stopped.json")); err != nil {
		t.Error(err)
	}
}

func TestANodeThatExitsMidRunIsReported(t *testing.T) {
	t.Parallel()
	runDir := filepath.Join(t.TempDir(), "run")
	run, err := world.Launch(worldtest.Context(t), world.Options{Binary: stub{}.build(t), Scenario: stubScenario(), RunDir: runDir, Origin: worldtest.Origin(t)})
	if err != nil {
		t.Fatal(err)
	}
	home, err := os.FindProcess(run.Manifest().Nodes[0].PID)
	if err != nil {
		t.Fatal(err)
	}
	if err := home.Kill(); err != nil {
		t.Fatal(err)
	}
	worldtest.Until(t, "the manifest says home exited", func() (bool, error) {
		m, err := world.ReadManifest(runDir)
		if err != nil {
			return false, err
		}
		return m.Nodes[0].State == world.NodeExited && m.Nodes[0].PID == 0, nil
	})
	report, err := run.Stop(worldtest.Context(t))
	if len(report.Nodes) != 1 || !report.Nodes[0].ExitedUnasked || report.Nodes[0].ExitCode == 0 {
		t.Fatalf("Stop = %+v, want home exited unasked and non-zero", report)
	}
	if want := fmt.Sprintf("node home exited with code %d before it was asked to stop", report.Nodes[0].ExitCode); err == nil || err.Error() != want {
		t.Fatalf("Stop = %v, want %s", err, want)
	}
	assertStopped(t, run.Manifest())
}

func TestLaunchRefuses(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		stub    stub
		want    string
		spawned bool
	}{
		{
			name: "a census entry",
			stub: stub{resolved: node.Resolved{
				Keys:         append(productionKeys(), node.Key{Key: "endpoints.example.api", Value: "https://api.example.org/", Default: "https://api.example.org/", Source: "default"}),
				Census:       []string{"endpoints.example.api"},
				ContentTypes: []string{},
			}},
			want: "node home: endpoints still at real hosts: endpoints.example.api (https://api.example.org/); add its fake to fakes/world",
		},
		{
			name: "a divergence the list does not name",
			stub: stub{resolved: node.Resolved{
				Keys:         append(productionKeys(), node.Key{Key: "torrent.dht", Value: "true", Default: "false", Source: "config.yaml"}),
				Census:       []string{},
				ContentTypes: []string{},
			}},
			want: "node home sets torrent.dht, which the divergences list in fakes/world does not name",
		},
		{
			name: "a fake setting a key the binary does not list",
			stub: stub{resolved: node.Resolved{Keys: productionKeys(), Census: []string{}, ContentTypes: []string{"music"}}},
			want: "fake music-meta sets endpoints.content_types.music.metadata, which this binary does not list",
		},
		{
			name:    "a listener off the Node's address",
			stub:    stub{torrent: "0.0.0.0:6881"},
			want:    "node home listener torrent is on 0.0.0.0:6881, not on its address 127.0.0.2",
			spawned: true,
		},
		{
			name:    "a Node that exits before it is ready",
			stub:    stub{runExit: "3"},
			want:    "node home exited with code 3 before it was ready\nstub node starting, a line that is not JSON\nstub node told to exit with 3",
			spawned: true,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			runDir := filepath.Join(t.TempDir(), "run")
			_, err := world.Launch(worldtest.Context(t), world.Options{Binary: c.stub.build(t), Scenario: stubScenario(), RunDir: runDir, Origin: worldtest.Origin(t)})
			if err == nil || err.Error() != c.want {
				t.Fatalf("Launch = %v\nwant %s", err, c.want)
			}
			assertNothingLeft(t, runDir, c.spawned)
			stops, err := events.Select(runDir, events.Query{Kind: events.RunStopped})
			if wantData := data(t, struct {
				Clean bool   `json:"clean"`
				Error string `json:"error"`
			}{false, c.want}); err != nil || len(stops) != 1 || string(stops[0].Data) != wantData {
				t.Fatalf("run.stopped records %+v, %v; want one with %s", stops, err, wantData)
			}
		})
	}
}

func TestLaunchRefusesABinaryWithoutItsTools(t *testing.T) {
	t.Parallel()
	runDir := filepath.Join(t.TempDir(), "run")
	_, err := world.Launch(worldtest.Context(t), world.Options{Binary: stub{noTools: true}.build(t), Scenario: stubScenario(), RunDir: runDir, Origin: worldtest.Origin(t)})
	if err == nil || !errors.Is(err, fs.ErrNotExist) || !strings.HasPrefix(err.Error(), "ffmpeg beside the binary: ") {
		t.Fatalf("Launch = %v, want ffmpeg beside the binary not found", err)
	}
	assertNothingLeft(t, runDir, false)
}

func data(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func assertNothingLeft(t *testing.T, runDir string, spawned bool) {
	t.Helper()
	spawns, err := events.Select(runDir, events.Query{Kind: events.NodeSpawned})
	if err != nil {
		t.Fatal(err)
	}
	if spawned != (len(spawns) > 0) {
		t.Fatalf("node.spawned records = %+v, want spawned %t", spawns, spawned)
	}
	for _, s := range spawns {
		var d struct{ PID int }
		if err := json.Unmarshal(s.Data, &d); err != nil {
			t.Fatal(err)
		}
		if !worldtest.ProcessGone(d.PID) {
			t.Errorf("node %s (PID %d) still runs", s.Node, d.PID)
		}
	}
	launched, err := events.Select(runDir, events.Query{Kind: events.RunLaunched})
	if err != nil || len(launched) != 1 {
		t.Fatalf("run.launched records = %+v, %v; want one", launched, err)
	}
	var l struct{ Control string }
	if err := json.Unmarshal(launched[0].Data, &l); err != nil {
		t.Fatal(err)
	}
	scratch := strings.TrimSuffix(l.Control, ".sock")
	for _, path := range []string{filepath.Join(runDir, "manifest.json"), scratch, l.Control} {
		if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("%s after a failed launch: %v, want it absent", path, err)
		}
	}
}

func TestLaunchUnderAnExpiredContextLeavesNothing(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithDeadline(worldtest.Context(t), time.Now().Add(-time.Second))
	defer cancel()
	runDir := filepath.Join(t.TempDir(), "run")
	_, err := world.Launch(ctx, world.Options{Binary: "slipstream", Scenario: stubScenario(), RunDir: runDir, Origin: worldtest.Origin(t)})
	if err == nil || err.Error() != "context deadline exceeded" {
		t.Fatalf("Launch = %v, want context deadline exceeded", err)
	}
	if _, err := os.Stat(runDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("run dir after an expired launch: %v, want it absent", err)
	}
}

func TestLaunchRefusesAnExistingRunDir(t *testing.T) {
	t.Parallel()
	runDir := t.TempDir()
	_, err := world.Launch(worldtest.Context(t), world.Options{Binary: "slipstream", Scenario: stubScenario(), RunDir: runDir, Origin: worldtest.Origin(t)})
	if want := "run dir " + runDir + " already exists"; err == nil || err.Error() != want {
		t.Fatalf("Launch = %v, want %s", err, want)
	}
}

func TestLaunchRefusesAFakeWithoutAnOnlineStance(t *testing.T) {
	t.Parallel()
	bare := echo
	bare.Online = fake.Online{}
	runDir := filepath.Join(t.TempDir(), "run")
	_, err := world.LaunchWith(worldtest.Context(t), world.Options{Binary: "slipstream", Scenario: stubScenario(), RunDir: runDir, Origin: worldtest.Origin(t)}, []fake.Spec{bare})
	if want := "fake echo declares no Online stance; set fake.OnlineSuite(suite) or fake.NoOnlineRun(reason)"; err == nil || err.Error() != want {
		t.Fatalf("Launch = %v\nwant %s", err, want)
	}
	if _, err := os.Stat(runDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("run dir after a refused launch: %v, want it absent", err)
	}
}

func TestLaunchRefusesVolumeFigures(t *testing.T) {
	t.Parallel()
	s, err := scenario.Load("home")
	if err != nil {
		t.Fatal(err)
	}
	s.Library.Roots[0].Volume = &scenario.LibraryVolume{TotalGiB: 10, FreeGiB: 1}
	runDir := filepath.Join(t.TempDir(), "run")
	_, err = world.Launch(worldtest.Context(t), world.Options{Binary: "slipstream", Scenario: s, RunDir: runDir, Origin: worldtest.Origin(t)})
	if want := "scenario home: library root movies: volume figures play only in the simulation tier"; err == nil || err.Error() != want {
		t.Fatalf("Launch = %v\nwant %s", err, want)
	}
	if _, err := os.Stat(runDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("run dir after a refused launch: %v, want it absent", err)
	}
}

func TestLaunchRefusesATimingTheLoaderRefuses(t *testing.T) {
	t.Parallel()
	s := stubScenario()
	s.Timings.Node.StopDeadline = -time.Second
	runDir := filepath.Join(t.TempDir(), "run")
	_, err := world.Launch(worldtest.Context(t), world.Options{Binary: testbin.Slipstream(t), Scenario: s, RunDir: runDir, Origin: worldtest.Origin(t)})
	if want := "node home: config.yaml:14: timings.node.stop_deadline: duration -1s is not positive"; err == nil || err.Error() != want {
		t.Fatalf("Launch = %v\nwant %s", err, want)
	}
	assertNothingLeft(t, runDir, false)
}

func TestLaunchRefusesAScenarioNeedingAContentTypeTheBinaryLacks(t *testing.T) {
	t.Parallel()
	s, err := scenario.Load("music")
	if err != nil {
		t.Fatal(err)
	}
	runDir := filepath.Join(t.TempDir(), "run")
	_, err = world.Launch(worldtest.Context(t), world.Options{Binary: testbin.Slipstream(t), Scenario: s, RunDir: runDir, Origin: worldtest.Origin(t)})
	want := "scenario music needs Content Type music, which slipstream does not register; launch it with worldtest.LaunchE2E"
	if runtime.GOOS == "windows" {
		want = strings.Replace(want, "slipstream", "slipstream.exe", 1)
	}
	if err == nil || err.Error() != want {
		t.Fatalf("Launch = %v\nwant %s", err, want)
	}
	assertNothingLeft(t, runDir, false)
	wl, err := os.ReadFile(filepath.Join(runDir, "world.log"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(wl), "fake started") {
		t.Fatalf("world.log records a fake started:\n%s", wl)
	}
}

func TestTheE2EBinaryCensusListsItsEndpointsAtTheirDefaults(t *testing.T) {
	t.Parallel()
	dir, err := os.MkdirTemp("", "ss") //nolint:usetesting // t.TempDir's path is too long for the socket limit on macOS and Windows
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	var stderr strings.Builder
	cmd := exec.CommandContext(worldtest.Context(t), testbin.E2E(t), "config", "--data-dir", dir)
	cmd.Env = nodeenv.Environ("")
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("slipstream-e2e config: %v\n%s", err, stderr.String())
	}
	var resolved node.Resolved
	if err := json.Unmarshal(out, &resolved); err != nil {
		t.Fatal(err)
	}
	if got, want := fmt.Sprint(resolved.ContentTypes, resolved.Census), "[movie music] [endpoints.tmdb.api endpoints.content_types.music.metadata]"; got != want {
		t.Fatalf("contentTypes and census = %s, want %s", got, want)
	}
}

func TestAddressCheckRefusesTheThirtyThirdAddress(t *testing.T) {
	t.Parallel()
	_, err := world.CheckAddresses(worldtest.Context(t), 32)
	if want := `127.0.0.33 is not usable on this machine; on macOS run "sudo fakeworld setup-loopback" once`; err == nil || err.Error() != want {
		t.Fatalf("err = %v, want %s", err, want)
	}
}

func TestStopRunWithoutASupervisorNamesEachNode(t *testing.T) {
	t.Parallel()
	runDir := t.TempDir()
	log := `{"seq":1,"at":"2026-10-14T00:00:00Z","kind":"run.launched","node":"world","data":{"pid":4242,"control":"` + filepath.ToSlash(filepath.Join(runDir, "gone.sock")) + `"}}
{"seq":2,"at":"2026-10-14T00:00:01Z","kind":"node.spawned","node":"home","data":{"pid":4250,"dataDir":"/tmp/ssfw/k3v9x2qa/home"}}
`
	if err := os.WriteFile(filepath.Join(runDir, "events.jsonl"), []byte(log), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := world.StopRun(worldtest.Context(t), runDir)
	want := "the supervisor (PID 4242) is not running; stop each Node with:\n  slipstream stop --data-dir /tmp/ssfw/k3v9x2qa/home"
	if _, gone := errors.AsType[*world.SupervisorGoneError](err); !gone || err.Error() != want {
		t.Fatalf("StopRun = %v, want %s", err, want)
	}
}

// pinnedListeners reads the listen keys of a Node's config.yaml, by
// listener name.
func pinnedListeners(t *testing.T, path string) map[string]string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var config map[string]any
	if err := yaml.Unmarshal(data, &config); err != nil {
		t.Fatal(err)
	}
	out := map[string]string{}
	for _, name := range []string{"http", "link", "torrent"} {
		block, _ := config[name].(map[string]any)
		out[name], _ = block["listen"].(string)
	}
	return out
}

// advertisedLink reads link.advertise from a Node's config.yaml.
func advertisedLink(t *testing.T, path string) []string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var config struct {
		Link struct {
			Advertise []string `yaml:"advertise"`
		} `yaml:"link"`
	}
	if err := yaml.Unmarshal(data, &config); err != nil {
		t.Fatal(err)
	}
	return config.Link.Advertise
}

func TestLaunchSetsTheScenarioFaultsAndRefusesOneNoFakeServes(t *testing.T) {
	t.Parallel()
	bin := stub{}.build(t)
	ctx := worldtest.Context(t)
	s := stubScenario()
	s.Faults = []fault.Rule{{ID: "echo-503", Target: "echo", Route: "echo", Fault: fault.Status{Code: 503}, Times: 1}}
	runDir := filepath.Join(t.TempDir(), "run")
	run, err := world.LaunchWith(ctx, world.Options{Binary: bin, Scenario: s, RunDir: runDir, Origin: worldtest.Origin(t)}, []fake.Spec{echo})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = run.Stop(ctx) })
	if got := getFake(ctx, t, run.Manifest()); got != "Service Unavailable\n" {
		t.Fatalf("the first request answered %q, want the Scenario's 503", got)
	}
	if got := besideFronts(t, runDir)[1]; got != `fault.set world {"id":"echo-503","target":"echo","route":"echo","times":1,"fault":"status=503"}` {
		t.Fatalf("the record after run.launched is %s, want the Scenario rule's fault.set", got)
	}

	s.Faults[0].Route = "nope"
	runDir = filepath.Join(t.TempDir(), "run")
	_, err = world.LaunchWith(ctx, world.Options{Binary: bin, Scenario: s, RunDir: runDir, Origin: worldtest.Origin(t)}, []fake.Spec{echo})
	if want := "scenario stub fault echo-503: echo serves no route nope"; err == nil || err.Error() != want {
		t.Fatalf("Launch = %v, want %s", err, want)
	}
	assertNothingLeft(t, runDir, false)
}
