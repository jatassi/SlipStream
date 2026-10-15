package main_test

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/fakes/events"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/testbin"
	"github.com/jatassi/slipstream/fakes/world"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
)

func self(t *testing.T) string {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	return exe
}

func jsonText(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// journal is the run's step and fault records as seq-free lines.
func journal(t *testing.T, runDir string) []string {
	t.Helper()
	recs, err := events.Read(runDir)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, r := range recs {
		if !strings.HasPrefix(r.Kind, "step.") && r.Kind != events.FaultSet && r.Kind != events.FaultCleared {
			continue
		}
		line := fmt.Sprintf("%s %s", r.Kind, r.Data)
		if r.Files != nil {
			line += fmt.Sprint(" ", r.Files)
		}
		out = append(out, line)
	}
	return out
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestExecJournalsAndReplays(t *testing.T) {
	t.Parallel()
	runDir, m := startWorld(t)
	t.Cleanup(func() { fakeworld(t, "stop", runDir) })
	exe := self(t)
	kept := filepath.Join(t.TempDir(), "kept")
	if err := os.WriteFile(kept, []byte("kept\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	set := fakeworld(t, "fault", "set", runDir, "tmdb-503", "tmdb", "status=503", "--times", "1")
	var setRec events.Record
	if err := json.Unmarshal([]byte(set.stdout), &setRec); set.code != 0 || err != nil || setRec.Kind != events.FaultSet {
		t.Fatalf("fault set = %+v, want its fault.set record", set)
	}
	get := []string{"--proof", "search-and-grab/held", "--", exe, commandArg, "get", "{ca.file}", "{fakes.tmdb.url}/3/configuration"}
	if got, want := fakeworld(t, append([]string{"exec", runDir}, get...)...), (result{0, "503\n", ""}); got != want {
		t.Fatalf("exec get = %+v, want %+v", got, want)
	}
	wait := fakeworld(t, "fault", "wait", runDir, "--rule", "tmdb-503")
	var fired events.Record
	if err := json.Unmarshal([]byte(wait.stdout), &fired); wait.code != 0 || err != nil || fired.Node != "world" || string(fired.Data) != `{"rule":"tmdb-503","target":"tmdb","fault":"status=503","match":1}` {
		t.Fatalf("fault wait = %+v, want the tmdb-503 firing", wait)
	}
	read := []string{"--", exe, commandArg, "read", kept}
	if got, want := fakeworld(t, append([]string{"exec", runDir}, read...)...), (result{0, "kept\n", ""}); got != want {
		t.Fatalf("exec read = %+v, want %+v", got, want)
	}
	failing := []string{"--", exe, commandArg, "echo", "exit=4", "{nodes.home.url}", "{apiKey}"}
	if got, want := fakeworld(t, append([]string{"exec", runDir}, failing...)...), (result{4, m.Nodes[0].URL + "\n" + m.APIKey + "\n", "stderr of the step\n"}); got != want {
		t.Fatalf("exec of a failing command = %+v, want %+v", got, want)
	}

	want := []string{
		`step.start {"n":1,"verb":"fault set","args":["tmdb-503","tmdb","status=503","--times","1"]}`,
		`fault.set {"fault":"status=503","id":"tmdb-503","step":1,"target":"tmdb","times":1}`,
		`step.end {"n":1,"exit":0}`,
		`step.start {"n":2,"verb":"exec","args":` + jsonText(t, get) + `}`,
		`step.end {"n":2,"exit":0} map[stderr:proof/search-and-grab/held/2.err stdout:proof/search-and-grab/held/2.out]`,
		`step.start {"n":3,"verb":"fault wait","args":["--rule","tmdb-503"]}`,
		`step.end {"n":3,"exit":0}`,
		`step.start {"n":4,"verb":"exec","args":` + jsonText(t, read) + `}`,
		`step.end {"n":4,"exit":0} map[stderr:journal/4.err stdout:journal/4.out]`,
		`step.start {"n":5,"verb":"exec","args":` + jsonText(t, failing) + `}`,
		`step.end {"n":5,"exit":4} map[stderr:journal/5.err stdout:journal/5.out]`,
	}
	if got := journal(t, runDir); !reflect.DeepEqual(got, want) {
		t.Fatalf("the journal holds\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	for path, want := range map[string]string{
		"proof/search-and-grab/held/2.out": "503\n",
		"journal/4.out":                    "kept\n",
		"journal/5.out":                    m.Nodes[0].URL + "\n" + m.APIKey + "\n",
		"journal/5.err":                    "stderr of the step\n",
	} {
		if got := readFile(t, filepath.Join(runDir, filepath.FromSlash(path))); got != want {
			t.Errorf("%s = %q, want %q", path, got, want)
		}
	}

	list := fakeworld(t, "fault", "list", runDir)
	if want := (result{0, "", ""}); list != want {
		t.Fatalf("fault list after tmdb-503 played its one time = %+v, want %+v", list, want)
	}

	checkReplayTo2(t, runDir, m, want[:5])

	if err := os.Remove(kept); err != nil {
		t.Fatal(err)
	}
	changed := fakeworld(t, "replay", runDir, "--stop")
	if !strings.HasSuffix(changed.stderr, "fakeworld replay: step 4 exited 1 (recorded 0)\n") || changed.code != 1 {
		t.Fatalf("replay after a step's input changed = %+v, want exit 1 naming step 4", changed)
	}
	if _, err := os.Stat(filepath.Join(runDir+"-replay-2", "manifest.stopped.json")); err != nil {
		t.Fatalf("replay --stop left its world running: %v", err)
	}

	fakeworld(t, "stop", runDir)
	if stopped := fakeworld(t, "fault", "list", runDir); stopped != list {
		t.Fatalf("fault list on the stopped run = %+v, want %+v", stopped, list)
	}
}

// checkReplayTo2 replays the run's first two steps on a fresh world and
// checks it journals them as the run did, on ports of its own.
func checkReplayTo2(t *testing.T, runDir string, m world.Manifest, want []string) {
	t.Helper()
	replayed := runDir + "-replay-1"
	t.Cleanup(func() { fakeworld(t, "stop", replayed) })
	if got := fakeworld(t, "replay", runDir, "--to", "2"); got.code != 0 || got.stderr != "" {
		t.Fatalf("replay --to 2 = %+v, want exit 0", got)
	}
	again, err := world.ReadManifest(replayed)
	if err != nil {
		t.Fatal(err)
	}
	if tmdb, tmdbAgain := tmdbURL(t, m), tmdbURL(t, again); again.Nodes[0].URL == m.Nodes[0].URL || tmdbAgain == tmdb {
		t.Fatalf("the replay's home at %s and tmdb at %s, want ports other than the run's %s and %s", again.Nodes[0].URL, tmdbAgain, m.Nodes[0].URL, tmdb)
	}
	if got := journal(t, replayed); !reflect.DeepEqual(got, want) {
		t.Fatalf("the replay's journal holds\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	if got := readFile(t, filepath.Join(replayed, "proof", "search-and-grab", "held", "2.out")); got != "503\n" {
		t.Fatalf("the replayed step 2 printed %q, want the run's 503", got)
	}
}

func tmdbURL(t *testing.T, m world.Manifest) string {
	t.Helper()
	urls, err := world.Expand(m, []string{"{fakes.tmdb.url}"})
	if err != nil {
		t.Fatal(err)
	}
	return urls[0]
}

func TestReplayRefusesAnotherBinary(t *testing.T) {
	t.Parallel()
	runDir, m := startWorld(t)
	t.Cleanup(func() { fakeworld(t, "stop", runDir) })
	exe := self(t)
	sum := sha256.New()
	f, err := os.Open(exe)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	if _, err := io.Copy(sum, f); err != nil {
		t.Fatal(err)
	}
	want := result{1, "", fmt.Sprintf("fakeworld replay: %s hashes to %x, not the manifest's %s\n", exe, sum.Sum(nil), m.Binary.SHA256)}
	if got := fakeworld(t, "replay", runDir, "--binary", exe); got != want {
		t.Fatalf("replay with another binary = %+v, want %+v", got, want)
	}
}

func TestTokens(t *testing.T) {
	t.Parallel()
	runDir, m := startWorld(t)
	t.Cleanup(func() { fakeworld(t, "stop", runDir) })
	exe := self(t)
	touched := filepath.Join(t.TempDir(), "touched")

	got := fakeworld(t, "exec", runDir, "--", exe, commandArg, "touch", touched, "{nodes.nas.url}")
	if wantErr := "fakeworld: exec: {nodes.nas.url} names nothing in the manifest\nusage: fakeworld <command>\n"; got.code != 3 || got.stdout != "" || !strings.HasPrefix(got.stderr, wantErr) {
		t.Fatalf("exec with an unknown path = %+v, want exit 3 with %q", got, wantErr)
	}
	if _, err := os.Stat(touched); err == nil {
		t.Fatal("the command ran despite an unknown path")
	}
	if steps := journal(t, runDir); len(steps) != 0 {
		t.Fatalf("an unknown path journaled %q", steps)
	}

	literal := fakeworld(t, "exec", runDir, "--", exe, commandArg, "echo", "exit=0", "{{apiKey}", `{"name":"{nodes.home.name}"}`, "{{{apiKey}}")
	if want := (result{0, "{apiKey}\n" + `{"name":"home"}` + "\n{" + m.APIKey + "}\n", "stderr of the step\n"}); literal != want {
		t.Fatalf("exec with literal braces = %+v, want %+v", literal, want)
	}
}

// TestReplayTarget is the test whose command the bundle in
// TestReplayRunsTheRecordedTest records. Run by that replay, it writes
// the binaries and the runs dir it was handed to TEST_REPLAY_OUT.
func TestReplayTarget(t *testing.T) {
	t.Parallel()
	if out := os.Getenv("TEST_REPLAY_OUT"); out != "" {
		handed := strings.Join([]string{os.Getenv("TEST_SLIPSTREAM_BINARY"), os.Getenv("TEST_FAKEWORLD_BINARY"), os.Getenv("TEST_FAKEWORLD_RUNS")}, "\n")
		if err := os.WriteFile(out, []byte(handed), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

func TestReplayRunsTheRecordedTest(t *testing.T) {
	t.Parallel()
	runDir := worldtest.RunDir(t)
	s, err := scenario.Load("first-run")
	if err != nil {
		t.Fatal(err)
	}
	ctx := worldtest.Context(t)
	run, err := world.Launch(ctx, world.Options{Binary: testbin.Slipstream(t), Scenario: s, RunDir: runDir, Origin: world.Origin{Kind: world.FromTest, Command: "go test ./cmd/fakeworld -run '^TestReplayTarget$' -count=1"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := run.Stop(ctx); err != nil {
		t.Fatal(err)
	}

	out := filepath.Join(t.TempDir(), "handed")
	cmd := exec.CommandContext(ctx, testbin.Fakeworld(t), "replay", runDir)
	unset := func(kv string) bool { return strings.HasPrefix(kv, "TEST_FAKEWORLD_BINARY=") }
	cmd.Env = append(slices.DeleteFunc(os.Environ(), unset), "TEST_REPLAY_OUT="+out)
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("replay of a test bundle: %v\n%s", err, b)
	}
	handed := strings.Split(readFile(t, out), "\n")
	if len(handed) != 3 {
		t.Fatalf("the replayed test was handed %q, want three paths", handed)
	}
	// os.Executable resolves symlinks on Linux only.
	handed[1] = resolved(t, handed[1])
	want := []string{filepath.Join(runDir, "bin", filepath.Base(run.Manifest().Binary.Path)), resolved(t, testbin.Fakeworld(t)), runDir + "-replay-1"}
	if !slices.Equal(handed, want) {
		t.Fatalf("the replayed test was handed %q, want the bundle's binary, the replaying fakeworld and the replay dir %q", handed, want)
	}
}

func resolved(t *testing.T, path string) string {
	t.Helper()
	p, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatal(err)
	}
	return p
}
