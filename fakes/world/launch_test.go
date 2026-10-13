package world_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/testbin"
	"github.com/jatassi/slipstream/fakes/world"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
	"github.com/jatassi/slipstream/internal/node"
)

func TestMain(m *testing.M) {
	os.Exit(testbin.Run(m))
}

type stub struct {
	resolved node.Resolved
	torrent  string
	runExit  string
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
		r = node.Resolved{Keys: productionKeys(), Census: []string{}}
	}
	resolved, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	flags := "-X=main.resolved=" + string(resolved) + " -X=main.torrent=" + s.torrent + " -X=main.runExit=" + s.runExit
	return testbin.Build(t, t.TempDir(), "./fakes/world/testdata/stubnode", "-ldflags", flags)
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
	TLS:  true,
	New: func(scenario.Scenario, fake.Env) (fake.Server, error) {
		return fake.HTTP(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, "echo\n") })), nil
	},
	Endpoints: func(*node.EndpointsBlock, fake.Address) {},
}

func TestLaunchAndStopOnTheStub(t *testing.T) {
	t.Parallel()
	bin := stub{}.build(t)
	runDir := filepath.Join(t.TempDir(), "run")
	ctx := worldtest.Context(t)
	run, err := world.LaunchWith(ctx, world.Options{Binary: bin, Scenario: stubScenario(), RunDir: runDir}, []fake.Spec{echo})
	if err != nil {
		t.Fatal(err)
	}
	m := run.Manifest()
	checkLaunched(t, m, runDir, bin)
	if got := getFake(ctx, t, m); got != "echo\n" {
		t.Fatalf("GET the echo fake = %q", got)
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
	if again, err := run.Stop(); err != nil || !reflect.DeepEqual(again, want) {
		t.Fatalf("Stop after StopRun = %+v, %v; want %+v", again, err, want)
	}
	if again, err := world.StopRun(ctx, runDir); err != nil || !reflect.DeepEqual(again, want) {
		t.Fatalf("StopRun on a stopped run = %+v, %v; want %+v", again, err, want)
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
	wantHome := world.Node{
		Name: "home", Setup: scenario.SetupNone, Claim: world.ClaimUnclaimed,
		Address: "127.0.0.2", Host: "home.localhost",
		URL: "http://127.0.0.2:0", BrowserURL: "http://home.localhost:0",
		Listeners: map[string]string{"control": filepath.Join(scratch, "home", "run", "control.sock"), "http": "127.0.0.2:0", "link": "127.0.0.2:0", "torrent": "127.0.0.2:0"},
		DataDir:   filepath.Join(scratch, "home"), PID: home.PID, Seed: "stub/home",
		Log: filepath.Join(runDir, "nodes", "home.log"), Overrides: []node.Key{}, NodeKey: "0000-0000-0000-0000",
	}
	if !reflect.DeepEqual(home, wantHome) {
		t.Fatalf("manifest node = %+v\nwant %+v", home, wantHome)
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
	wl, err := world.ReadWorldLog(runDir)
	if err != nil {
		t.Fatal(err)
	}
	if want := []world.Spawn{{Node: "home", PID: home.PID, DataDir: home.DataDir}}; !reflect.DeepEqual(wl.Spawns, want) || wl.Supervisor != m.Supervisor {
		t.Fatalf("world.log records %+v, want spawns %+v and supervisor %+v", wl, want, m.Supervisor)
	}
}

func getFake(ctx context.Context, t *testing.T, m world.Manifest) string {
	t.Helper()
	pem, err := os.ReadFile(m.CA.File)
	if err != nil {
		t.Fatal(err)
	}
	roots := x509.NewCertPool()
	roots.AppendCertsFromPEM(pem)
	client := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12}}}
	defer client.CloseIdleConnections()
	if len(m.Fakes) != 1 || m.Fakes[0].Name != "echo" || !strings.HasPrefix(m.Fakes[0].URL, "https://127.0.0.1:") {
		t.Fatalf("fakes = %+v", m.Fakes)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, m.Fakes[0].URL+"/", http.NoBody)
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
		if !worldtest.ProcessGone(n.PID) {
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
	run, err := world.Launch(worldtest.Context(t), world.Options{Binary: stub{}.build(t), Scenario: stubScenario(), RunDir: runDir})
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
	worldtest.Until(t, "world.log records home's exit", func() (bool, error) {
		data, err := os.ReadFile(filepath.Join(runDir, "world.log"))
		return strings.Contains(string(data), `"msg":"node exited"`), err
	})
	report, err := run.Stop()
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
				Keys:   append(productionKeys(), node.Key{Key: "endpoints.example.api", Value: "https://api.example.org/", Default: "https://api.example.org/", Source: "default"}),
				Census: []string{"endpoints.example.api"},
			}},
			want: "node home: endpoints still at real hosts: endpoints.example.api (https://api.example.org/); add its fake to fakes/world",
		},
		{
			name: "a divergence the list does not name",
			stub: stub{resolved: node.Resolved{
				Keys:   append(productionKeys(), node.Key{Key: "torrent.dht", Value: "true", Default: "false", Source: "config.yaml"}),
				Census: []string{},
			}},
			want: "node home sets torrent.dht, which the divergences list in fakes/world does not name",
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
			_, err := world.Launch(worldtest.Context(t), world.Options{Binary: c.stub.build(t), Scenario: stubScenario(), RunDir: runDir})
			if err == nil || err.Error() != c.want {
				t.Fatalf("Launch = %v\nwant %s", err, c.want)
			}
			assertNothingLeft(t, runDir, c.spawned)
		})
	}
}

func assertNothingLeft(t *testing.T, runDir string, spawned bool) {
	t.Helper()
	wl, err := world.ReadWorldLog(runDir)
	if err != nil {
		t.Fatal(err)
	}
	if spawned != (len(wl.Spawns) > 0) {
		t.Fatalf("world.log spawns = %+v, want spawned %t", wl.Spawns, spawned)
	}
	for _, s := range wl.Spawns {
		if !worldtest.ProcessGone(s.PID) {
			t.Errorf("node %s (PID %d) still runs", s.Node, s.PID)
		}
	}
	scratch := strings.TrimSuffix(wl.Supervisor.Control, ".sock")
	for _, path := range []string{filepath.Join(runDir, "manifest.json"), scratch, wl.Supervisor.Control} {
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
	_, err := world.Launch(ctx, world.Options{Binary: "slipstream", Scenario: stubScenario(), RunDir: runDir})
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
	_, err := world.Launch(worldtest.Context(t), world.Options{Binary: "slipstream", Scenario: stubScenario(), RunDir: runDir})
	if want := "run dir " + runDir + " already exists"; err == nil || err.Error() != want {
		t.Fatalf("Launch = %v, want %s", err, want)
	}
}

func TestLaunchRefusesATimingTheLoaderRefuses(t *testing.T) {
	t.Parallel()
	s := stubScenario()
	s.Timings.Node.StopDeadline = -time.Second
	runDir := filepath.Join(t.TempDir(), "run")
	_, err := world.Launch(worldtest.Context(t), world.Options{Binary: testbin.Slipstream(t), Scenario: s, RunDir: runDir})
	if want := "node home: config.yaml:11: timings.node.stop_deadline: duration -1s is not positive"; err == nil || err.Error() != want {
		t.Fatalf("Launch = %v\nwant %s", err, want)
	}
	assertNothingLeft(t, runDir, false)
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
	log := `{"msg":"launch","pid":4242,"control":"` + filepath.ToSlash(filepath.Join(runDir, "gone.sock")) + `"}
panic: a line that is not JSON
{"msg":"node spawned","node":"home","pid":4250,"data_dir":"/tmp/ssfw/k3v9x2qa/home"}
`
	if err := os.WriteFile(filepath.Join(runDir, "world.log"), []byte(log), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := world.StopRun(worldtest.Context(t), runDir)
	want := "the supervisor (PID 4242) is not running; stop each Node with:\n  slipstream stop --data-dir /tmp/ssfw/k3v9x2qa/home"
	if _, gone := errors.AsType[*world.SupervisorGoneError](err); !gone || err.Error() != want {
		t.Fatalf("StopRun = %v, want %s", err, want)
	}
}
