package main_test

import (
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/testbin"
	"github.com/jatassi/slipstream/fakes/world"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
	"github.com/jatassi/slipstream/internal/buildinfo"
	"github.com/jatassi/slipstream/internal/worktree"
)

func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.CommandContext(worldtest.Context(t), "git", append([]string{"-C", dir, "-c", "user.name=test", "-c", "user.email=test@example.com", "-c", "commit.gpgsign=false"}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// stampedBuild builds cmd/slipstream stamped as a build of the worktree
// dir, with the tool programs beside it.
func stampedBuild(t *testing.T, dir string) string {
	t.Helper()
	stamp, err := worktree.Stamp(worldtest.Context(t), dir)
	if err != nil {
		t.Fatal(err)
	}
	ldflag, err := buildinfo.Encode(stamp)
	if err != nil {
		t.Fatal(err)
	}
	bin := t.TempDir()
	binary := testbin.Build(t, bin, "./cmd/slipstream", "-ldflags", "-X=github.com/jatassi/slipstream/internal/buildinfo.stamp="+ldflag)
	testbin.PlaceTools(t, bin)
	return binary
}

func TestDoctorPassesThenFailsOnAnEditedWorktree(t *testing.T) {
	t.Parallel()
	repo := t.TempDir()
	git(t, repo, "init", "-q")
	readme := filepath.Join(repo, "README.md")
	if err := os.WriteFile(readme, []byte("readme\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	git(t, repo, "add", "-A")
	git(t, repo, "commit", "-q", "-m", "initial")
	binary := stampedBuild(t, repo)

	runDir := worldtest.RunDir(t)
	if got := fakeworld(t, "start", "--binary", binary, "--scenario", "home", "--run-dir", runDir); got.code != 0 {
		t.Fatalf("start = %+v", got)
	}
	t.Cleanup(func() { fakeworld(t, "stop", runDir) })
	passed := fakeworld(t, "doctor", runDir, "--worktree", repo)
	want := "ok   1 the run is ours\nok   2 the build matches the worktree\nok   3 the cluster matches the Scenario\nok   4 every fake answers\n" +
		"ok   5 Admin-entered addresses point at fakes\nok   6 the API key works\n"
	if passed != (result{0, want, ""}) {
		t.Fatalf("doctor on a fresh run = %+v, want exit 0 with\n%s", passed, want)
	}

	if err := os.WriteFile(readme, []byte("edited\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	edited, err := worktree.Fingerprint(worldtest.Context(t), repo)
	if err != nil {
		t.Fatal(err)
	}
	failing := fakeworld(t, "doctor", runDir, "--worktree", repo)
	want = strings.Replace(want, "ok   2 the build matches the worktree\n", "FAIL 2 the build matches the worktree\n"+
		"       worktree changed since the build (fingerprint clean vs "+world.Abbrev(edited)+"); run go run ./cmd/release build and relaunch\n", 1)
	if failing != (result{1, want, ""}) {
		t.Fatalf("doctor after an edit = %+v, want exit 1 with\n%s", failing, want)
	}
	if kept := readFile(t, filepath.Join(runDir, "journal", "2.out")); kept != want {
		t.Fatalf("journal/2.out = %q, want the second report", kept)
	}
}

// TestDoctorCatchesARealIndexerURL plants an Indexer at a real address on
// a stand-in Controller, since creating one through the API fails its caps
// call.
func TestDoctorCatchesARealIndexerURL(t *testing.T) {
	t.Parallel()
	controller := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/indexers" || r.Header.Get("X-Api-Key") != "ssk_test" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`{"indexers":[{"name":"Planted","feedHost":"example.org","homeNodeId":"n1"}]}`))
	}))
	t.Cleanup(controller.Close)

	runDir := t.TempDir()
	ca := filepath.Join(runDir, "ca.pem")
	if err := os.WriteFile(ca, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: controller.Certificate().Raw}), 0o600); err != nil {
		t.Fatal(err)
	}
	m := world.Manifest{
		RunDir: runDir, CA: world.CAFile{File: ca}, APIKey: "ssk_test",
		Nodes:  []world.Node{{Name: "home", Setup: scenario.SetupController, URL: controller.URL, NodeID: "n1", State: world.NodeRunning, PID: os.Getpid()}},
		Fakes:  []world.Fake{{Name: "fixture-site"}},
		Fronts: []world.Front{{Target: "fixture-site", Node: "home", Address: "127.0.0.1:1"}},
	}
	data, err := json.Marshal(m) //nolint:gosec // a stand-in run's manifest
	if err != nil {
		t.Fatal(err)
	}
	for name, content := range map[string][]byte{"manifest.json": data, "events.jsonl": nil} {
		if err := os.WriteFile(filepath.Join(runDir, name), content, 0o600); err != nil {
			t.Fatal(err)
		}
	}

	got := fakeworld(t, "doctor", runDir, "--json")
	var report world.Report
	if err := json.Unmarshal([]byte(got.stdout), &report); got.code != 1 || err != nil || len(report.Checks) != 6 {
		t.Fatalf("doctor = %+v, want exit 1 with a report", got)
	}
	want := world.CheckResult{N: 5, Name: "Admin-entered addresses point at fakes", Failures: []string{"Indexer Planted https://example.org is not fixture-site's front for home"}}
	if !reflect.DeepEqual(report.Checks[4], want) {
		t.Fatalf("check 5 = %+v, want %+v", report.Checks[4], want)
	}
	if !strings.Contains(got.stdout, `,"activeFaults":[]}`) {
		t.Fatalf("doctor --json = %s, want an empty activeFaults list", got.stdout)
	}
}
