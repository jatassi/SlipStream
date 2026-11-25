package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"go.yaml.in/yaml/v3"

	"github.com/jatassi/slipstream/cmd/release/internal/changescope"
	"github.com/jatassi/slipstream/cmd/release/internal/native"
)

var planInputs = native.Inputs{
	Targets: []native.Target{
		{Name: "linux-amd64", GOOS: "linux", GOARCH: "amd64", Runner: "ubuntu-24.04", Toolchain: "musl-cross", Gate: native.GatePR},
		{Name: "windows-amd64", GOOS: "windows", GOARCH: "amd64", Runner: "windows-2025", Toolchain: "llvm-mingw-ucrt", Gate: native.GateNative},
		{Name: "darwin-arm64", GOOS: "darwin", GOARCH: "arm64", Runner: "macos-15", Toolchain: "xcode", Gate: native.GateNative},
	},
	Toolchains: []native.Toolchain{{Name: "musl-cross"}, {Name: "llvm-mingw-ucrt"}, {Name: "xcode", Xcode: "16.4"}},
}

const testImage = "mcr.microsoft.com/playwright:v1.64.1-noble"

func laneTargets(m Matrix) []string {
	var targets []string
	for _, e := range m.Include {
		if !slices.Contains(targets, e.Target) {
			targets = append(targets, e.Target)
		}
	}
	return targets
}

func TestPlanTargets(t *testing.T) {
	prOnly := []string{"linux-amd64"}
	all := []string{"linux-amd64", "windows-amd64", "darwin-arm64"}
	tests := []struct {
		kind    string
		scope   changescope.Scope
		touched bool
		want    []string
		pack    bool
	}{
		{"merge", changescope.Docs, false, nil, false},
		{"merge", changescope.PR, false, prOnly, false},
		{"merge", changescope.Native, false, all, false},
		{"merge", changescope.PR, true, all, true},
		{"wave", changescope.Docs, false, nil, false},
		{"wave", changescope.PR, false, prOnly, false},
		{"wave", changescope.Native, false, prOnly, false},
		{"wave", changescope.PR, true, prOnly, true},
		{"train", changescope.PR, false, prOnly, false},
		{"train", changescope.Native, false, prOnly, false},
		{"push", changescope.Docs, false, all, false},
		{"dispatch", changescope.Docs, false, all, false},
		{"dispatch", changescope.Native, false, all, false},
		{"schedule", changescope.Docs, false, all, true},
		{"tag", changescope.Docs, false, all, true},
	}
	for _, tt := range tests {
		got, err := plan(planInputs, runKinds[tt.kind], tt.scope, touched{packages: tt.touched, diagram: true}, nil, "all", testImage, nil)
		if err != nil {
			t.Fatal(err)
		}
		for _, l := range lanes {
			if !slices.Contains(runKinds[tt.kind].lanes, l.kind) {
				continue
			}
			want := slices.DeleteFunc(slices.Clone(tt.want), func(target string) bool { return !l.runsOn(target) })
			if targets := laneTargets(got.Lanes[l.kind]); !slices.Equal(targets, want) {
				t.Errorf("plan(%s, %s, touched %t) runs %s on %v, want %v", tt.kind, tt.scope, tt.touched, l.kind, targets, want)
			}
		}
		if got.Package != tt.pack {
			t.Errorf("plan(%s, %s, touched %t) packages %t, want %t", tt.kind, tt.scope, tt.touched, got.Package, tt.pack)
		}
	}
}

func TestEachRunKindRunsItsLanes(t *testing.T) {
	for kind, want := range map[string][]string{
		"wave":     {"build", "lint", "lint-repo", "go", "component", "diagram"},
		"train":    {"build", "e2e", "browser", "go-cache"},
		"merge":    {"build", "lint", "lint-repo", "go", "e2e", "browser", "component", "diagram"},
		"push":     {"lint", "go-cache"},
		"dispatch": {"build", "lint", "lint-repo", "go", "e2e", "browser", "component", "diagram"},
	} {
		got, err := plan(planInputs, runKinds[kind], changescope.Native, touched{diagram: true}, nil, "all", testImage, nil)
		if err != nil {
			t.Fatal(err)
		}
		var planned []string
		for _, l := range lanes {
			if len(got.Lanes[l.kind].Include) > 0 {
				planned = append(planned, l.kind)
			}
		}
		if !slices.Equal(planned, want) {
			t.Errorf("a %s plans %v, want %v", kind, planned, want)
		}
	}
}

func testRecords(t *testing.T) map[laneTarget]recorded {
	t.Helper()
	rec, err := loadRecords(filepath.Join("testdata", "timings"))
	if err != nil {
		t.Fatal(err)
	}
	if len(rec) == 0 {
		t.Fatal("testdata/timings holds no recorded timings")
	}
	return rec
}

// shardNames is each job of m as its target, shard and timings run.
func shardNames(m Matrix) []string {
	var names []string
	for _, e := range m.Include {
		names = append(names, fmt.Sprintf("%s %d/%d timings %q", e.Target, e.Shard, e.Shards, e.Timings))
	}
	return names
}

func TestPlanShardsEachTimedLaneByItsRecordedTime(t *testing.T) {
	missing, err := loadRecords(filepath.Join(t.TempDir(), "missing"))
	if err != nil {
		t.Fatal(err)
	}
	for name, rec := range map[string]map[laneTarget]recorded{"testdata": testRecords(t), "no timings": missing} {
		got, err := plan(planInputs, runKinds["merge"], changescope.Native, touched{}, nil, "all", testImage, rec)
		if err != nil {
			t.Fatal(err)
		}
		for _, l := range slices.DeleteFunc(slices.Clone(lanes), func(l lane) bool { return l.shardTime == 0 }) {
			want := Matrix{}
			for _, target := range planInputs.Targets {
				r := rec[laneTarget{l.kind, target.Name}]
				n := l.defaultShards
				if r.run != "" {
					n = int(math.Ceil(r.seconds / l.shardTime.Seconds()))
				}
				for i := 1; i <= n; i++ {
					want.Include = append(want.Include, MatrixEntry{Target: target.Name, Shard: i, Shards: n, Timings: r.run})
				}
			}
			if shards, want := shardNames(got.Lanes[l.kind]), shardNames(want); !slices.Equal(shards, want) {
				t.Errorf("%s: %s shards = %v, want %v", name, l.kind, shards, want)
			}
		}
	}
}

func TestPlanFitsEachRunKindsBudget(t *testing.T) {
	in := testInputs(t)
	jobs := func(k runKind, rec map[laneTarget]recorded, tch touched) int {
		p, err := plan(in, k, changescope.Native, tch, nil, "all", testImage, rec)
		if err != nil {
			t.Fatal(err)
		}
		return p.Jobs()
	}
	for kind, k := range runKinds {
		uncapped := k
		uncapped.budget = 0
		for _, rec := range []map[laneTarget]recorded{testRecords(t), nil} {
			for _, tch := range []touched{{}, {packages: true, diagram: true}} {
				if got, want := jobs(k, rec, tch), min(k.budget, jobs(uncapped, rec, tch)); k.budget > 0 && got != want {
					t.Errorf("a %s touching %+v with %d recorded lanes plans %d jobs, want %d of its budget of %d", kind, tch, len(rec), got, want, k.budget)
				}
			}
		}
	}
}

func TestPlanDiagramWhenTouched(t *testing.T) {
	for _, tt := range []struct {
		kind    string
		scope   changescope.Scope
		touched touched
		want    bool
	}{
		{"merge", changescope.PR, touched{}, false},
		{"merge", changescope.PR, touched{diagram: true}, true},
		{"merge", changescope.Native, touched{packages: true}, false},
		{"merge", changescope.Docs, touched{diagram: true}, false},
		{"wave", changescope.PR, touched{}, false},
		{"wave", changescope.PR, touched{diagram: true}, true},
		{"dispatch", changescope.Native, touched{}, true},
		{"schedule", changescope.Docs, touched{}, true},
		{"tag", changescope.Docs, touched{}, true},
	} {
		got, err := plan(planInputs, runKinds[tt.kind], tt.scope, tt.touched, nil, "all", testImage, nil)
		if err != nil {
			t.Fatal(err)
		}
		if planned := got.PlannedJobs()["diagram"]; planned != tt.want {
			t.Errorf("plan(%s, %s, %+v) plans diagram %t, want %t", tt.kind, tt.scope, tt.touched, planned, tt.want)
		}
	}
}

func testInputs(t *testing.T) native.Inputs {
	t.Helper()
	root, err := repoRoot(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	in, err := native.LoadInputs(filepath.Join(root, native.InputsPath))
	if err != nil {
		t.Fatal(err)
	}
	return in
}

func TestLanesCoverEveryTarget(t *testing.T) {
	in := testInputs(t)
	everywhere := []string{"build", "go", "e2e", "browser", "go-cache"}
	for _, l := range lanes {
		for _, name := range l.targets {
			if _, err := in.Target(name); err != nil {
				t.Errorf("lane %s runs on %s: %v", l.kind, name, err)
			}
		}
		if (l.shardTime > 0) != (l.defaultShards > 0) {
			t.Errorf("lane %s has shard time %s and %d default shards, want both or neither", l.kind, l.shardTime, l.defaultShards)
		}
		switch {
		case l.kind == "lint":
		case slices.Contains(everywhere, l.kind) && l.targets != nil:
			t.Errorf("lane %s runs on %v, want every target", l.kind, l.targets)
		case !slices.Contains(everywhere, l.kind) && len(l.targets) != 1:
			t.Errorf("lane %s checks what no OS changes, so runs on one target, not %v", l.kind, l.targets)
		}
	}
}

func TestLintRunsOncePerOS(t *testing.T) {
	in := testInputs(t)
	i := slices.IndexFunc(lanes, func(l lane) bool { return l.kind == "lint" })
	jobs := map[string]int{}
	for _, target := range in.Targets {
		if lanes[i].runsOn(target.Name) {
			jobs[target.GOOS]++
		}
	}
	for _, target := range in.Targets {
		if jobs[target.GOOS] != 1 {
			t.Errorf("lint runs %d jobs on %s, want one per OS", jobs[target.GOOS], target.GOOS)
		}
	}
}

func TestPlanTargetsSplitsEachLaneByTarget(t *testing.T) {
	got, err := plan(planInputs, runKinds["dispatch"], changescope.Native, touched{}, nil, "browser", testImage, nil)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	browser := 0
	for _, c := range got.Targets().Include {
		names = append(names, c.Target)
		browser += len(c.Browser.Include)
		for kind, m := range map[string]Matrix{"build": c.Build, "browser": c.Browser} {
			if targets := laneTargets(m); !slices.Equal(targets, []string{c.Target}) {
				t.Errorf("call for %s runs %s on %v", c.Target, kind, targets)
			}
		}
		if len(c.E2E.Include) != 0 {
			t.Errorf("call for %s runs e2e on a browser dispatch: %v", c.Target, c.E2E)
		}
	}
	if want := laneTargets(got.Lanes["build"]); !slices.Equal(names, want) {
		t.Errorf("calls on %v, want %v", names, want)
	}
	if want := len(got.Lanes["browser"].Include); browser != want {
		t.Errorf("calls hold %d browser jobs, want %d", browser, want)
	}
}

func TestPlanTargetsEmptyWithoutABuild(t *testing.T) {
	got, err := plan(planInputs, runKinds["dispatch"], changescope.Native, touched{}, nil, "go", testImage, nil)
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(got.Targets())
	if err != nil {
		t.Fatal(err)
	}
	if want := `{"include":[]}`; string(b) != want {
		t.Errorf("target matrix JSON = %s, want %s", b, want)
	}
}

type ciJob struct {
	Name      string `yaml:"name"`
	If        string `yaml:"if"`
	Container struct {
		Image string `yaml:"image"`
	} `yaml:"container"`
	Needs    any               `yaml:"needs"`
	Uses     string            `yaml:"uses"`
	With     map[string]any    `yaml:"with"`
	Outputs  map[string]string `yaml:"outputs"`
	Strategy struct {
		Matrix any `yaml:"matrix"`
	} `yaml:"strategy"`
}

func readWorkflow(t *testing.T, name string) map[string]ciJob {
	t.Helper()
	root, err := repoRoot(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(root, ".github", "workflows", name))
	if err != nil {
		t.Fatal(err)
	}
	var w struct {
		Jobs map[string]ciJob `yaml:"jobs"`
	}
	if err := yaml.Unmarshal(b, &w); err != nil {
		t.Fatal(err)
	}
	return w.Jobs
}

// checkContainer checks that a job runs in the container its plan entries
// name, and only then.
func checkContainer(t *testing.T, l lane, j ciJob) {
	t.Helper()
	want := ""
	if l.playwright {
		want = "${{ matrix.container }}"
	}
	if j.Container.Image != want {
		t.Errorf("job %s container image = %q, want %q", l.kind, j.Container.Image, want)
	}
}

func TestTestsWorkflowCallsTargetWorkflowPerTarget(t *testing.T) {
	call := readWorkflow(t, "tests.yml")["target"]
	if want := "./.github/workflows/target.yml"; call.Uses != want {
		t.Errorf("job target uses %q, want %q", call.Uses, want)
	}
	if want := "fromJSON(inputs.plan).planned['target']"; call.If != want {
		t.Errorf("job target if = %q, want %q", call.If, want)
	}
	if want := "${{ fromJSON(inputs.plan)['target'] }}"; call.Strategy.Matrix != want {
		t.Errorf("job target matrix = %v, want %q", call.Strategy.Matrix, want)
	}
}

func TestTargetWorkflowRunsItsLanesOnItsBuild(t *testing.T) {
	jobs := readWorkflow(t, "target.yml")
	for _, l := range lanes {
		if !l.inTargetWorkflow() {
			continue
		}
		j, ok := jobs[l.kind]
		if !ok {
			t.Errorf("target.yml has no %s job", l.kind)
			continue
		}
		if want := fmt.Sprintf("${{ fromJSON(inputs.%s) }}", l.kind); j.Strategy.Matrix != want {
			t.Errorf("job %s matrix = %v, want %q", l.kind, j.Strategy.Matrix, want)
		}
		checkContainer(t, l, j)
		if l.kind == "build" {
			continue
		}
		if want := fmt.Sprintf("fromJSON(inputs.%s).include[0]", l.kind); j.If != want {
			t.Errorf("job %s if = %q, want %q", l.kind, j.If, want)
		}
		if j.Needs != "build" {
			t.Errorf("job %s needs %v, want build for its dist/", l.kind, j.Needs)
		}
	}
}

func TestTestsWorkflowRunsEveryLane(t *testing.T) {
	jobs := readWorkflow(t, "tests.yml")
	got, err := plan(planInputs, runKinds["dispatch"], changescope.Native, touched{}, nil, "all", testImage, nil)
	if err != nil {
		t.Fatal(err)
	}
	output := got.output()
	for _, key := range []string{"planned", "package", "target"} {
		if _, ok := output[key]; !ok {
			t.Errorf("the plan output has no %s", key)
		}
	}
	for _, l := range lanes {
		if l.inTargetWorkflow() {
			continue
		}
		if _, ok := output[l.kind]; !ok {
			t.Errorf("the plan output has no %s matrix", l.kind)
		}
		j, ok := jobs[l.kind]
		if !ok {
			t.Errorf("tests.yml has no %s job", l.kind)
			continue
		}
		if want := fmt.Sprintf("${{ fromJSON(inputs.plan)['%s'] }}", l.kind); j.Strategy.Matrix != want {
			t.Errorf("job %s matrix = %v, want %q", l.kind, j.Strategy.Matrix, want)
		}
		checkContainer(t, l, j)
		if want := fmt.Sprintf("fromJSON(inputs.plan).planned['%s']", l.kind); j.If != want {
			t.Errorf("job %s if = %q, want %q", l.kind, j.If, want)
		}
	}
}

// The ruleset requires the pull request run's gate by name, and a job in a
// called workflow is named after its caller too, so each gate is a job of
// the workflow an event starts. It fails when the plan or the tests call
// fails or is canceled, and a failed lane fails the call.
func TestEachRunGatesOnItsTests(t *testing.T) {
	for workflow, gate := range map[string]string{"ci.yml": "All required checks passed", "push.yml": "All push checks passed"} {
		jobs := readWorkflow(t, workflow)
		if want := "${{ steps.plan.outputs.plan }}"; jobs["plan"].Outputs["plan"] != want {
			t.Errorf("%s: plan output plan = %q, want %q", workflow, jobs["plan"].Outputs["plan"], want)
		}
		tests := jobs["tests"]
		if want := "./.github/workflows/tests.yml"; tests.Uses != want || tests.Needs != "plan" {
			t.Errorf("%s: job tests uses %q and needs %v, want %q and plan", workflow, tests.Uses, tests.Needs, want)
		}
		if want := "${{ needs.plan.outputs.plan }}"; tests.With["plan"] != want {
			t.Errorf("%s: job tests takes plan %v, want %q", workflow, tests.With["plan"], want)
		}
		if named := jobsNamed(jobs, gate); !slices.Equal(named, []string{"gate"}) {
			t.Fatalf("%s: jobs %v are named %q, want the gate alone", workflow, named, gate)
		}
		j := jobs["gate"]
		if j.Uses != "" {
			t.Errorf("%s: the gate calls %s, which would name it after its caller", workflow, j.Uses)
		}
		if want := "${{ always() && (contains(needs.*.result, 'failure') || contains(needs.*.result, 'cancelled')) }}"; j.If != want {
			t.Errorf("%s: gate if = %q, want %q", workflow, j.If, want)
		}
		needs, ok := j.Needs.([]any)
		if !ok || !slices.Equal(needs, []any{"plan", "tests"}) {
			t.Errorf("%s: gate needs %v, want plan and tests", workflow, j.Needs)
		}
	}
}

func jobsNamed(jobs map[string]ciJob, name string) []string {
	var named []string
	for id, j := range jobs {
		if j.Name == name {
			named = append(named, id)
		}
	}
	return named
}
