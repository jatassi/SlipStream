// Package worldtest is the end-to-end tier's helpers: launch a world for a
// test, wait on conditions under the test's deadline, and read what a
// person would read from a Node's log.
package worldtest

import (
	"bytes"
	"encoding/json"
	"errors"
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

// stopGrace is how long slipstream stop waits past a Node's stop deadline.
const stopGrace = 5 * time.Second

const logLinesOnFailure = 50

// Launch launches the catalog Scenario name. See LaunchScenario.
func Launch(t *testing.T, name string) *world.Run {
	t.Helper()
	s, err := scenario.Load(name)
	if err != nil {
		t.Fatal(err)
	}
	return LaunchScenario(t, s)
}

// LaunchScenario launches s in-process against the binary under test. The
// test's cleanup stops the run and fails the test on any unclean Node exit,
// a Node process still running, a scratch dir left behind or a manifest not
// renamed. On failure it logs the end of every Node log.
func LaunchScenario(t *testing.T, s scenario.Scenario) *world.Run {
	t.Helper()
	binary := testbin.Slipstream(t)
	runDir := newRunDir(t)
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
	run, err := world.Launch(Context(t), world.Options{Binary: binary, Scenario: s, RunDir: runDir})
	if err != nil {
		t.Fatalf("launch %s: %v", s.Name, err)
	}
	return run
}

// newRunDir is a path under TEST_FAKEWORLD_RUNS, removed when the test
// passes, or else under the test's temp dir.
func newRunDir(t *testing.T) string {
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
			if err := os.RemoveAll(dir); err != nil {
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
	if _, err := run.Stop(); err != nil {
		t.Errorf("the run did not stop cleanly:\n%v", err)
	}
	for _, n := range m.Nodes {
		if !ProcessGone(n.PID) {
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
	manifest, err := os.ReadFile(filepath.Join(m.RunDir, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(manifest, []byte(ready)) {
		t.Fatalf("manifest.json holds node %s's Setup code", name)
	}
	return ready
}

// ProcessGone reports whether no process with pid exists.
func ProcessGone(pid int) bool { return !proc.Alive(pid) }
