package main

import (
	"encoding/json"
	"reflect"
	"testing"

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

func TestPlan(t *testing.T) {
	prOnly := []MatrixEntry{{"linux-amd64", "ubuntu-24.04", "linux", "amd64", ""}}
	all := []MatrixEntry{
		{"linux-amd64", "ubuntu-24.04", "linux", "amd64", ""},
		{"windows-amd64", "windows-2025", "windows", "amd64", ""},
		{"darwin-arm64", "macos-15", "darwin", "arm64", "16.4"},
	}
	tests := []struct {
		event string
		scope changescope.Scope
		want  []MatrixEntry
	}{
		{"pr", changescope.Docs, []MatrixEntry{}},
		{"pr", changescope.PR, prOnly},
		{"pr", changescope.Native, all},
		{"push", changescope.Docs, all},
		{"push", changescope.PR, all},
		{"schedule", changescope.Docs, all},
		{"schedule", changescope.PR, all},
		{"tag", changescope.Docs, all},
		{"tag", changescope.PR, all},
	}
	for _, tt := range tests {
		got, err := plan(planInputs, tt.event, tt.scope, false)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got.Matrix.Include, tt.want) {
			t.Errorf("plan(%s, %s) = %v, want %v", tt.event, tt.scope, got.Matrix.Include, tt.want)
		}
	}
}

func TestPlanPackages(t *testing.T) {
	for _, tt := range []struct {
		event   string
		scope   changescope.Scope
		touched bool
		want    bool
		targets int
	}{
		{"tag", changescope.Docs, false, true, 3},
		{"schedule", changescope.Docs, false, true, 3},
		{"push", changescope.PR, false, false, 3},
		{"pr", changescope.PR, false, false, 1},
		{"pr", changescope.Docs, false, false, 0},
		{"pr", changescope.PR, true, true, 3},
	} {
		got, err := plan(planInputs, tt.event, tt.scope, tt.touched)
		if err != nil {
			t.Fatal(err)
		}
		if got.Package != tt.want || len(got.Matrix.Include) != tt.targets {
			t.Errorf("plan(%s, %s, touched %t) packages %t on %d targets, want %t on %d", tt.event, tt.scope, tt.touched, got.Package, len(got.Matrix.Include), tt.want, tt.targets)
		}
	}
}

func TestPlanJSON(t *testing.T) {
	p, err := plan(planInputs, "push", changescope.PR, false)
	if err != nil {
		t.Fatal(err)
	}
	got, err := json.Marshal(p.Matrix)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"include":[` +
		`{"target":"linux-amd64","runner":"ubuntu-24.04","goos":"linux","goarch":"amd64"},` +
		`{"target":"windows-amd64","runner":"windows-2025","goos":"windows","goarch":"amd64"},` +
		`{"target":"darwin-arm64","runner":"macos-15","goos":"darwin","goarch":"arm64","xcode":"16.4"}]}`
	if string(got) != want {
		t.Errorf("matrix JSON = %s, want %s", got, want)
	}
}
