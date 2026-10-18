package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
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
		event   string
		scope   changescope.Scope
		touched bool
		want    []string
		pack    bool
	}{
		{"pr", changescope.Docs, false, nil, false},
		{"pr", changescope.PR, false, prOnly, false},
		{"pr", changescope.Native, false, all, false},
		{"pr", changescope.PR, true, all, true},
		{"stacked", changescope.Docs, false, nil, false},
		{"stacked", changescope.PR, false, prOnly, false},
		{"stacked", changescope.Native, false, prOnly, false},
		{"stacked", changescope.PR, true, prOnly, true},
		{"push", changescope.Docs, false, all, false},
		{"push", changescope.PR, false, all, false},
		{"dispatch", changescope.Native, false, all, false},
		{"schedule", changescope.Docs, false, all, true},
		{"tag", changescope.Docs, false, all, true},
	}
	for _, tt := range tests {
		got, err := plan(planInputs, events[tt.event], tt.scope, tt.touched, nil, "all")
		if err != nil {
			t.Fatal(err)
		}
		for kind, m := range got.Lanes {
			if kind == "lint-repo" || kind == "component" {
				continue
			}
			if targets := laneTargets(m); !slices.Equal(targets, tt.want) {
				t.Errorf("plan(%s, %s, touched %t) runs %s on %v, want %v", tt.event, tt.scope, tt.touched, kind, targets, tt.want)
			}
		}
		if got.Package != tt.pack {
			t.Errorf("plan(%s, %s, touched %t) packages %t, want %t", tt.event, tt.scope, tt.touched, got.Package, tt.pack)
		}
	}
}

func TestPlanShardsEachLanePerTarget(t *testing.T) {
	got, err := plan(planInputs, events["push"], changescope.PR, false, nil, "all")
	if err != nil {
		t.Fatal(err)
	}
	for _, l := range lanes {
		var want []string
		for _, target := range planInputs.Targets {
			n := l.shards[target.Name]
			for i := 1; i <= n; i++ {
				want = append(want, fmt.Sprintf("%s %d/%d", target.Name, i, n))
			}
		}
		var shards []string
		for _, e := range got.Lanes[l.kind].Include {
			shards = append(shards, fmt.Sprintf("%s %d/%d", e.Target, e.Shard, e.Shards))
		}
		if !slices.Equal(shards, want) {
			t.Errorf("%s shards = %v, want %v", l.kind, shards, want)
		}
	}
}

func TestPlanDispatchKeepsTargetsAndTier(t *testing.T) {
	for _, tt := range []struct {
		keep []string
		tier string
		want map[string]bool
	}{
		{[]string{"windows-amd64"}, "go", map[string]bool{"go": true}},
		{nil, "browser", map[string]bool{"build": true, "browser": true}},
		{nil, "lint", map[string]bool{"lint": true, "lint-repo": true}},
		{[]string{"darwin-arm64"}, "lint", map[string]bool{"lint": true}},
		{[]string{"windows-amd64"}, "all", map[string]bool{"build": true, "lint": true, "go": true, "e2e": true, "browser": true}},
	} {
		got, err := plan(planInputs, events["dispatch"], changescope.Native, false, tt.keep, tt.tier)
		if err != nil {
			t.Fatal(err)
		}
		for kind, planned := range got.Planned() {
			if planned != tt.want[kind] {
				t.Errorf("dispatch of %v on %s: %s planned %t, want %t", tt.keep, tt.tier, kind, planned, tt.want[kind])
			}
			if targets := laneTargets(got.Lanes[kind]); tt.keep != nil && planned && !slices.Equal(targets, tt.keep) {
				t.Errorf("dispatch of %v on %s runs %s on %v", tt.keep, tt.tier, kind, targets)
			}
		}
	}
	if _, err := plan(planInputs, events["dispatch"], changescope.Native, false, []string{"plan9-amd64"}, "all"); err == nil {
		t.Error("a dispatch of an unknown target planned")
	}
	if _, err := plan(planInputs, events["dispatch"], changescope.Native, false, nil, "build"); err == nil {
		t.Error("a dispatch of an unknown tier planned")
	}
}

func TestLanesCoverEveryTarget(t *testing.T) {
	root, err := repoRoot(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	in, err := native.LoadInputs(filepath.Join(root, native.InputsPath))
	if err != nil {
		t.Fatal(err)
	}
	everywhere := []string{"build", "lint", "go", "e2e", "browser"}
	for _, l := range lanes {
		for name, n := range l.shards {
			if _, err := in.Target(name); err != nil || n < 1 {
				t.Errorf("lane %s runs %d shards on %s: %v", l.kind, n, name, err)
			}
		}
		if !slices.Contains(everywhere, l.kind) {
			if len(l.shards) != 1 {
				t.Errorf("lane %s checks what no OS changes, so runs on one target, not %v", l.kind, l.shards)
			}
			continue
		}
		for _, target := range in.Targets {
			if l.shards[target.Name] == 0 {
				t.Errorf("lane %s does not run on %s", l.kind, target.Name)
			}
		}
	}
}

func TestPlanJSON(t *testing.T) {
	p, err := plan(planInputs, events["push"], changescope.PR, false, nil, "all")
	if err != nil {
		t.Fatal(err)
	}
	got, err := json.Marshal(p.Lanes["e2e"])
	if err != nil {
		t.Fatal(err)
	}
	want := `{"include":[` +
		`{"target":"linux-amd64","runner":"ubuntu-24.04","goos":"linux","goarch":"amd64","shard":1,"shards":1},` +
		`{"target":"windows-amd64","runner":"windows-2025","goos":"windows","goarch":"amd64","shard":1,"shards":1},` +
		`{"target":"darwin-arm64","runner":"macos-15","goos":"darwin","goarch":"arm64","xcode":"16.4","shard":1,"shards":1}]}`
	if string(got) != want {
		t.Errorf("e2e matrix JSON = %s, want %s", got, want)
	}
}

// TestCIRunsEveryLane holds ci.yml to the lane table: a job per lane that
// runs the plan's matrix when the plan says so, and a gate over them all.
func TestCIRunsEveryLane(t *testing.T) {
	root, err := repoRoot(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(root, ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatal(err)
	}
	type job struct {
		If       string            `yaml:"if"`
		Needs    any               `yaml:"needs"`
		Outputs  map[string]string `yaml:"outputs"`
		Strategy struct {
			Matrix any `yaml:"matrix"`
		} `yaml:"strategy"`
	}
	var ci struct {
		Jobs map[string]job `yaml:"jobs"`
	}
	if err := yaml.Unmarshal(b, &ci); err != nil {
		t.Fatal(err)
	}
	gateNeeds := []string{"plan"}
	for _, l := range lanes {
		gateNeeds = append(gateNeeds, l.kind)
		j, ok := ci.Jobs[l.kind]
		if !ok {
			t.Errorf("ci.yml has no %s job", l.kind)
			continue
		}
		if want := fmt.Sprintf("${{ fromJSON(needs.plan.outputs.%s) }}", l.kind); j.Strategy.Matrix != want {
			t.Errorf("job %s matrix = %v, want %q", l.kind, j.Strategy.Matrix, want)
		}
		if want := fmt.Sprintf("fromJSON(needs.plan.outputs.planned)['%s']", l.kind); j.If != want {
			t.Errorf("job %s if = %q, want %q", l.kind, j.If, want)
		}
		if needs, _ := j.Needs.([]any); slices.Contains(lanesOnBuild, l.kind) && !slices.Contains(needs, any("build")) {
			t.Errorf("job %s needs %v, want build among them for its dist/", l.kind, j.Needs)
		}
		if want := fmt.Sprintf("${{ steps.plan.outputs.%s }}", l.kind); ci.Jobs["plan"].Outputs[l.kind] != want {
			t.Errorf("plan output %s = %q, want %q", l.kind, ci.Jobs["plan"].Outputs[l.kind], want)
		}
	}
	gate, ok := ci.Jobs["gate"].Needs.([]any)
	if !ok {
		t.Fatalf("gate needs %v, want a list", ci.Jobs["gate"].Needs)
	}
	var needs []string
	for _, n := range gate {
		needs = append(needs, fmt.Sprint(n))
	}
	if !slices.Equal(needs, gateNeeds) {
		t.Errorf("gate needs %s, want %s", strings.Join(needs, ", "), strings.Join(gateNeeds, ", "))
	}
}
