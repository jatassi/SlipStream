package world_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/world"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
)

func TestNodeActionsRewriteTheManifest(t *testing.T) {
	t.Parallel()
	runDir := filepath.Join(t.TempDir(), "run")
	ctx := worldtest.Context(t)
	run, err := world.LaunchWith(ctx, world.Options{Binary: stub{}.build(t), Scenario: stubScenario(), RunDir: runDir, Origin: worldtest.Origin(t)}, []fake.Spec{echo})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = run.Stop(ctx) })
	launched := run.Manifest().Nodes[0]
	if launched.State != world.NodeRunning || launched.PID == 0 {
		t.Fatalf("home launched %s with PID %d, want running with its PID", launched.State, launched.PID)
	}

	code, stdout, stderr := fakeworld(t, "node", "kill", runDir, "home")
	var killed world.Node
	if err := json.Unmarshal([]byte(stdout), &killed); code != 0 || err != nil || stderr != "" || killed.State != world.NodeKilled || killed.PID != 0 {
		t.Fatalf("fakeworld node kill = %d %q %q, want home's entry, killed with PID 0", code, stdout, stderr)
	}
	if !worldtest.ProcessGone(launched.PID) {
		t.Fatalf("home (PID %d) still runs after the kill", launched.PID)
	}
	restarted := actOnHome(t, run, world.StartNode, world.NodeRunning)
	if restarted.PID == launched.PID || !reflect.DeepEqual(restarted.Listeners, launched.Listeners) {
		t.Fatalf("home restarted with PID %d on %v, want a new PID on %v", restarted.PID, restarted.Listeners, launched.Listeners)
	}
	actOnHome(t, run, world.StopNode, world.NodeStopped)
	actOnHome(t, run, world.WipeNode, world.NodeStopped)
	if entries, err := os.ReadDir(launched.DataDir); err != nil || len(entries) != 1 || entries[0].Name() != "config.yaml" {
		t.Fatalf("home's wiped data dir holds %v, %v; want only config.yaml", entries, err)
	}
	actOnHome(t, run, world.StartNode, world.NodeRunning)
	checkNodeActionRefusals(t, run)

	want := []string{
		"node.spawned home", "node.ready home",
		"step.start world", "node.exited home", "node.killed home", "step.end world",
		"node.spawned home", "node.ready home",
		"node.exited home", "node.stopped home",
		"node.wiped home",
		"node.spawned home", "node.ready home",
	}
	if kinds := nodeAndStepKinds(t, runDir); !reflect.DeepEqual(kinds, want) {
		t.Fatalf("events.jsonl's Node and step records are\n%s\nwant\n%s", strings.Join(kinds, "\n"), strings.Join(want, "\n"))
	}
	if rec := fmt.Sprintf(`node.killed home {"pid":%d,"step":1}`, launched.PID); !slices.Contains(besideFronts(t, runDir), rec) {
		t.Errorf("events.jsonl has no %s", rec)
	}
}

// actOnHome runs a on the stub world's home and checks the entry it
// answers is the one manifest.json now holds.
func actOnHome(t *testing.T, run *world.Run, a world.NodeAction, state world.NodeState) world.Node {
	t.Helper()
	entry, err := run.Node(worldtest.Context(t), "home", a)
	if err != nil {
		t.Fatalf("%s: %v", a, err)
	}
	if entry.State != state || (state == world.NodeRunning) != (entry.PID != 0) {
		t.Fatalf("%s left home %s with PID %d, want %s", a, entry.State, entry.PID, state)
	}
	m, err := world.ReadManifest(run.Manifest().RunDir)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(m.Nodes[0], entry) {
		t.Fatalf("after %s manifest.json holds %+v, want %+v", a, m.Nodes[0], entry)
	}
	return entry
}

func checkNodeActionRefusals(t *testing.T, run *world.Run) {
	t.Helper()
	for _, c := range []struct {
		node string
		a    world.NodeAction
		want string
	}{
		{"home", world.StartNode, "node home is running; start needs it stopped or killed or exited"},
		{"home", world.WipeNode, "node home is running; wipe needs it stopped"},
		{"nope", world.StopNode, "the run has no node nope"},
	} {
		if _, err := run.Node(worldtest.Context(t), c.node, c.a); err == nil || err.Error() != c.want {
			t.Errorf("%s %s: %v, want %s", c.a, c.node, err, c.want)
		}
	}
	code, _, stderr := fakeworld(t, "node", "pause", run.Manifest().RunDir, "home")
	if want := `fakeworld: "pause" is not a Node action: stop, kill, start, restart or wipe` + "\n"; code != 3 || !strings.HasPrefix(stderr, want) {
		t.Errorf("fakeworld node pause = %d %q, want exit 3 starting %q", code, stderr, want)
	}
}

// nodeAndStepKinds is the kind and Node of each record in the run's event
// log but node.error that is about a Node or a step.
func nodeAndStepKinds(t *testing.T, runDir string) []string {
	t.Helper()
	var kinds []string
	for _, line := range besideFronts(t, runDir) {
		kind, rest, _ := strings.Cut(line, " ")
		if strings.HasPrefix(kind, "node.") && kind != "node.error" || strings.HasPrefix(kind, "step.") {
			node, _, _ := strings.Cut(rest, " ")
			kinds = append(kinds, kind+" "+node)
		}
	}
	return kinds
}

func TestTheWorldFollowsARestartedNode(t *testing.T) {
	t.Parallel()
	successor := map[string]bool{"as a successor": true}
	if runtime.GOOS != "windows" {
		successor["in place"] = false
	}
	for name, successor := range successor {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			followRestarts(t, successor)
		})
	}
}

// followRestarts restarts the stub's home, kills it, starts it and
// restarts it again, then stops the run, which must find it clean.
func followRestarts(t *testing.T, successor bool) {
	runDir := filepath.Join(t.TempDir(), "run")
	ctx := worldtest.Context(t)
	run, err := world.LaunchWith(ctx, world.Options{Binary: stub{successor: successor}.build(t), Scenario: stubScenario(), RunDir: runDir, Origin: worldtest.Origin(t)}, []fake.Spec{echo})
	if err != nil {
		t.Fatal(err)
	}
	launched := run.Manifest().Nodes[0]
	restarted := actOnHome(t, run, world.RestartNode, world.NodeRunning)
	if successor == (restarted.PID == launched.PID) || !reflect.DeepEqual(restarted.Listeners, launched.Listeners) {
		t.Fatalf("home restarted as PID %d on %v after PID %d on %v; want a new PID %t on the same listeners", restarted.PID, restarted.Listeners, launched.PID, launched.Listeners, successor)
	}
	if successor {
		gone(t, launched.PID)
	}
	if killed := actOnHome(t, run, world.KillNode, world.NodeKilled); killed.PID != 0 {
		t.Fatalf("after the kill home's entry has PID %d, want 0", killed.PID)
	}
	gone(t, restarted.PID)
	started := actOnHome(t, run, world.StartNode, world.NodeRunning)
	again := actOnHome(t, run, world.RestartNode, world.NodeRunning)
	report, err := run.Stop(ctx)
	if want := []world.NodeStop{{Name: "home"}}; err != nil || !reflect.DeepEqual(report.Nodes, want) {
		t.Fatalf("stop after the restart = %+v, %v; want %+v", report.Nodes, err, want)
	}
	gone(t, again.PID)

	want := []string{
		"node.spawned home", "node.ready home",
		"node.restarted home", "node.ready home",
		"node.exited home", "node.killed home",
		"node.spawned home", "node.ready home",
		"node.restarted home", "node.ready home",
		"node.exited home",
	}
	if kinds := nodeAndStepKinds(t, runDir); !reflect.DeepEqual(kinds, want) {
		t.Fatalf("events.jsonl's Node and step records are\n%s\nwant\n%s", strings.Join(kinds, "\n"), strings.Join(want, "\n"))
	}
	logged := besideFronts(t, runDir)
	for _, rec := range []string{
		fmt.Sprintf(`node.restarted home {"oldPid":%d,"pid":%d}`, launched.PID, restarted.PID),
		fmt.Sprintf(`node.restarted home {"oldPid":%d,"pid":%d}`, started.PID, again.PID),
	} {
		if !slices.Contains(logged, rec) {
			t.Errorf("events.jsonl has no %s", rec)
		}
	}
}

// gone waits for pid to be reaped. A successor's parent has exited, so
// init reaps it some time after its stderr closes.
func gone(t *testing.T, pid int) {
	t.Helper()
	worldtest.Until(t, fmt.Sprintf("PID %d is gone", pid), func() (bool, error) { return worldtest.ProcessGone(pid), nil })
}
