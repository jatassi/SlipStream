// Package worldtest is the end-to-end tier's helpers: launch a world for a
// test, wait on conditions under the test's deadline, and read what a
// person would read from a Node's log.
package worldtest

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/testbin"
	"github.com/jatassi/slipstream/fakes/world"
	"github.com/jatassi/slipstream/fakes/world/internal/proc"
	"github.com/jatassi/slipstream/internal/node"
)

const stopGrace = 5 * time.Second

const logLinesOnFailure = 50

// Launch launches the catalog Scenario name. See LaunchScenario.
func Launch(t *testing.T, name string) *world.Run {
	t.Helper()
	return LaunchScenario(t, load(t, name))
}

// LaunchScenario launches s in-process against the binary under test. The
// test's cleanup stops the run and fails the test on any unclean Node exit,
// a Node process still running, a scratch dir left behind or a manifest not
// renamed. On failure it logs the end of every Node log.
func LaunchScenario(t *testing.T, s scenario.Scenario) *world.Run {
	t.Helper()
	return launch(t, s, testbin.Slipstream(t))
}

// LaunchE2E launches the catalog Scenario name on the e2e binary, which
// also registers the test Content Types. See LaunchScenario.
func LaunchE2E(t *testing.T, name string) *world.Run {
	t.Helper()
	return LaunchE2EScenario(t, load(t, name))
}

// LaunchE2EScenario launches s on the e2e binary. See LaunchScenario.
func LaunchE2EScenario(t *testing.T, s scenario.Scenario) *world.Run {
	t.Helper()
	return launch(t, s, testbin.E2E(t))
}

func load(t *testing.T, name string) scenario.Scenario {
	t.Helper()
	s, err := scenario.Load(name)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func launch(t *testing.T, s scenario.Scenario, binary string) *world.Run {
	t.Helper()
	runDir := RunDir(t)
	addBudget(t, stopBudget(s))
	var run *world.Run
	t.Cleanup(func() {
		if run != nil {
			checkStop(t, run)
		}
		if t.Failed() {
			logNodeLogs(t, runDir)
		}
	})
	run, err := world.Launch(Context(t), world.Options{Binary: binary, Scenario: s, RunDir: runDir, Origin: Origin(t)})
	if err != nil {
		t.Fatalf("launch %s: %v", s.Name, err)
	}
	journalInto(t, run)
	return run
}

// Origin is the origin of a run t launches: the command that runs t
// again, from the module root.
func Origin(t *testing.T) world.Origin {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	pkg, err := filepath.Rel(testbin.ModuleRoot(t), wd)
	if err != nil {
		t.Fatal(err)
	}
	names := strings.Split(t.Name(), "/")
	for i, n := range names {
		names[i] = "^" + regexp.QuoteMeta(n) + "$"
	}
	return world.Origin{Kind: world.FromTest, Command: fmt.Sprintf("go test ./%s -run '%s' -count=1", filepath.ToSlash(pkg), strings.Join(names, "/"))}
}

// RunDir is a run dir that does not exist yet: under TEST_FAKEWORLD_RUNS,
// removed when the test passes, or else under the test's temp dir.
func RunDir(t *testing.T) string {
	t.Helper()
	runs, ok := os.LookupEnv("TEST_FAKEWORLD_RUNS")
	if !ok {
		return filepath.Join(t.TempDir(), "run")
	}
	if err := os.MkdirAll(runs, 0o755); err != nil {
		t.Fatal(err)
	}
	dir, err := os.MkdirTemp(runs, //nolint:usetesting // a run dir under TEST_FAKEWORLD_RUNS outlives a failed test for CI to upload
		strings.NewReplacer("/", "-", "\\", "-").Replace(t.Name())+"-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if !t.Failed() {
			if err := removeAll(Context(t), dir); err != nil {
				t.Error(err)
			}
		}
	})
	return filepath.Join(dir, "run")
}

func stopBudget(s scenario.Scenario) time.Duration {
	deadline := s.Timings.Node.StopDeadline
	if deadline == 0 {
		deadline = node.DefaultTimings().StopDeadline
	}
	return time.Duration(len(s.Nodes)) * (deadline + stopGrace)
}

func checkStop(t *testing.T, run *world.Run) {
	t.Helper()
	m := run.Manifest()
	nodes := make([]Process, len(m.Nodes))
	for i, n := range m.Nodes {
		if n.PID != 0 {
			nodes[i] = Watch(t, n.PID)
		}
	}
	if _, err := run.Stop(Context(t)); err != nil {
		t.Errorf("the run did not stop cleanly:\n%v", err)
	}
	for i, n := range m.Nodes {
		if n.PID != 0 && !nodes[i].Gone() {
			t.Errorf("node %s (PID %d) is still running after stop", n.Name, n.PID)
		}
	}
	if _, err := os.Stat(m.ScratchDir); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("scratch dir %s after stop: %v, want it gone", m.ScratchDir, err)
	}
	if _, err := os.Stat(filepath.Join(m.RunDir, "manifest.stopped.json")); err != nil {
		t.Errorf("manifest.stopped.json after stop: %v", err)
	}
	if _, err := os.Stat(filepath.Join(m.RunDir, "manifest.json")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("manifest.json after stop: %v, want it renamed", err)
	}
}

func logNodeLogs(t *testing.T, runDir string) {
	t.Helper()
	logs, _ := filepath.Glob(filepath.Join(runDir, "nodes", "*.log"))
	for _, path := range logs {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Log(err)
			continue
		}
		lines := strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
		lines = lines[max(0, len(lines)-logLinesOnFailure):]
		t.Logf("last %d lines of %s:\n%s", len(lines), path, strings.Join(lines, "\n"))
	}
}

var setupCodeShape = regexp.MustCompile(`^[0-9A-HJKMNP-TV-Z]{4}-[0-9A-HJKMNP-TV-Z]{4}-[0-9A-HJKMNP-TV-Z]{4}$`)

// SetupCode reads the Setup code of the Node named name from its log, the
// way a person reads it. The readiness record and the setup code record
// must agree, the code must have its shape, and the manifest must not hold
// it.
func SetupCode(t *testing.T, run *world.Run, name string) string {
	t.Helper()
	m := run.Manifest()
	var logPath string
	for _, n := range m.Nodes {
		if n.Name == name {
			logPath = n.Log
		}
	}
	if logPath == "" {
		t.Fatalf("the manifest has no node %s", name)
	}
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	var ready, logged string
	for line := range bytes.Lines(data) {
		if r, err := node.ParseReady(line); err == nil {
			ready = r.SetupCode
		}
		var rec struct {
			Msg  string `json:"msg"`
			Code string `json:"code"`
		}
		if json.Unmarshal(line, &rec) == nil && rec.Msg == "setup code" {
			logged = rec.Code
		}
	}
	if ready != logged || !setupCodeShape.MatchString(ready) {
		t.Fatalf("node %s: the readiness record's setup_code is %q and the setup code record's code is %q, want one code matching %s", name, ready, logged, setupCodeShape)
	}
	// os.Root, as world.ReadManifest says, lets the supervisor replace the
	// manifest while it is read.
	root, err := os.OpenRoot(m.RunDir)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = root.Close() }()
	manifest, err := root.ReadFile("manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(manifest, []byte(ready)) {
		t.Fatalf("manifest.json holds node %s's Setup code", name)
	}
	return ready
}

// ProcessGone reports whether no process with pid exists. Once a process
// is reaped, Windows can give its PID to another: Watch it while it runs.
func ProcessGone(pid int) bool { return !proc.Alive(pid) }

// Process is a process watched until a test ends.
type Process struct {
	PID int
	*proc.Watched
}

// Watch watches the process pid until t ends. Take it while pid still
// names that process.
func Watch(t *testing.T, pid int) Process {
	t.Helper()
	w, err := proc.Watch(pid)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.Close)
	return Process{pid, w}
}
