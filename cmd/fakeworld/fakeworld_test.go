package main_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"

	"github.com/jatassi/slipstream/fakes/events"
	"github.com/jatassi/slipstream/fakes/testbin"
	"github.com/jatassi/slipstream/fakes/world"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
)

type result struct {
	code           int
	stdout, stderr string
}

func fakeworld(t *testing.T, args ...string) result {
	t.Helper()
	cmd := exec.CommandContext(worldtest.Context(t), testbin.Fakeworld(t), args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		if _, exited := errors.AsType[*exec.ExitError](err); !exited {
			t.Fatal(err)
		}
	}
	return result{cmd.ProcessState.ExitCode(), stdout.String(), stderr.String()}
}

func start(t *testing.T, runDir string, flags ...string) result {
	t.Helper()
	args := append([]string{"start", "--binary", testbin.Slipstream(t), "--scenario", "single-machine", "--run-dir", runDir}, flags...)
	return fakeworld(t, args...)
}

func startWorld(t *testing.T) (string, world.Manifest) {
	t.Helper()
	runDir := worldtest.RunDir(t)
	if got, want := start(t, runDir), (result{0, filepath.Join(runDir, "manifest.json") + "\n", ""}); got != want {
		t.Fatalf("start = %+v, want %+v", got, want)
	}
	m, err := world.ReadManifest(runDir)
	if err != nil {
		t.Fatal(err)
	}
	return runDir, m
}

// untilGone waits for pid to vanish. A supervisor that exited is a zombie
// until init reaps it, since start, its parent, has already exited.
func untilGone(t *testing.T, what string, pid int) {
	t.Helper()
	worldtest.Until(t, fmt.Sprintf("%s (PID %d) is gone", what, pid), func() (bool, error) {
		return worldtest.ProcessGone(pid), nil
	})
}

func assertGone(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("%s: %v, want it gone", path, err)
	}
}

func names(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, e := range entries {
		out = append(out, e.Name())
	}
	return out
}

const cleanReport = `{
  "nodes": [
    {
      "name": "home",
      "exitCode": 0,
      "exitedUnasked": false
    }
  ]
}
`

func TestStartStop(t *testing.T) {
	t.Parallel()
	runDir, m := startWorld(t)
	home := m.Nodes[0]
	for what, pid := range map[string]int{"the supervisor": m.Supervisor.PID, "node home": home.PID} {
		if worldtest.ProcessGone(pid) {
			t.Fatalf("%s (PID %d) is not running after start", what, pid)
		}
	}

	if got, want := fakeworld(t, "stop", runDir), (result{0, cleanReport, ""}); got != want {
		t.Fatalf("stop = %+v, want %+v", got, want)
	}
	if !worldtest.ProcessGone(home.PID) {
		t.Errorf("node home (PID %d) is still running after stop", home.PID)
	}
	untilGone(t, "the supervisor", m.Supervisor.PID)
	assertGone(t, m.ScratchDir)
	assertGone(t, m.Supervisor.Control)
	stopped, err := os.ReadFile(filepath.Join(runDir, "stopped.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(stopped) != cleanReport {
		t.Errorf("stopped.json =\n%s\nwant\n%s", stopped, cleanReport)
	}
	want := []string{"bin", "ca.pem", "events.jsonl", "manifest.stopped.json", "nodes", "scenario.yaml", "stopped.json", "world.log"}
	if got := names(t, runDir); !slices.Equal(got, want) {
		t.Errorf("run dir holds %v, want %v", got, want)
	}
	if got := names(t, filepath.Join(runDir, "nodes")); !slices.Equal(got, []string{"home.log"}) {
		t.Errorf("nodes/ holds %v, want [home.log]", got)
	}
}

func TestStartRecordsTheTestCommand(t *testing.T) {
	t.Parallel()
	runDir := worldtest.RunDir(t)
	command := "bun run --cwd web e2e skeleton.spec.ts:6 --project chromium-desk"
	if got := start(t, runDir, "--test-command", command); got.code != 0 {
		t.Fatalf("start = %+v, want exit 0", got)
	}
	t.Cleanup(func() { fakeworld(t, "stop", runDir) })
	launched, err := events.Select(runDir, events.Query{Kind: events.RunLaunched})
	if err != nil || len(launched) != 1 {
		t.Fatalf("run.launched records = %+v, %v; want one", launched, err)
	}
	var data struct {
		Origin world.Origin `json:"origin"`
	}
	if err := json.Unmarshal(launched[0].Data, &data); err != nil {
		t.Fatal(err)
	}
	if want := (world.Origin{Kind: world.FromTest, Command: command}); data.Origin != want {
		t.Fatalf("run.launched origin = %+v, want %+v", data.Origin, want)
	}
}

func TestStartTimeout(t *testing.T) {
	t.Parallel()
	runDir := worldtest.RunDir(t)
	if got, want := start(t, runDir, "--timeout", "1ms"), (result{1, "", "fakeworld start: launch did not finish within 1ms\n"}); got != want {
		t.Fatalf("start = %+v, want %+v", got, want)
	}
	assertGone(t, filepath.Join(runDir, "manifest.json"))
	for _, pid := range loggedPIDs(t, runDir) {
		if !worldtest.ProcessGone(pid) {
			t.Errorf("PID %d from the event log is still running after a failed start", pid)
		}
	}
}

func loggedPIDs(t *testing.T, runDir string) []int {
	t.Helper()
	recs, err := events.Read(runDir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		t.Fatal(err)
	}
	var pids []int
	for _, r := range recs {
		var d struct {
			PID int `json:"pid"`
		}
		if json.Unmarshal(r.Data, &d) == nil && d.PID != 0 {
			pids = append(pids, d.PID)
		}
	}
	return pids
}

func TestStopWithoutSupervisor(t *testing.T) {
	t.Parallel()
	runDir, m := startWorld(t)
	home := m.Nodes[0]
	t.Cleanup(func() { stopByHand(t, m) })
	supervisor, err := os.FindProcess(m.Supervisor.PID)
	if err != nil {
		t.Fatal(err)
	}
	if err := supervisor.Kill(); err != nil {
		t.Fatal(err)
	}
	untilGone(t, "the supervisor", m.Supervisor.PID)

	want := result{1, "", fmt.Sprintf("fakeworld stop: the supervisor (PID %d) is not running; stop each Node with:\n  slipstream stop --data-dir %s\n", m.Supervisor.PID, home.DataDir)}
	if got := fakeworld(t, "stop", runDir); got != want {
		t.Fatalf("stop = %+v, want %+v", got, want)
	}
}

func stopByHand(t *testing.T, m world.Manifest) {
	t.Helper()
	for _, n := range m.Nodes {
		_ = exec.CommandContext(worldtest.Context(t), m.Binary.Path, "stop", "--data-dir", n.DataDir).Run()
		untilGone(t, "node "+n.Name, n.PID)
	}
	for _, path := range []string{m.ScratchDir, m.Supervisor.Control} {
		if err := os.RemoveAll(path); err != nil {
			t.Error(err)
		}
	}
}
