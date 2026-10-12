package main

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/jatassi/slipstream/cmd/release/internal/native"
)

var planTargets = []native.Target{
	{Name: "linux-amd64", GOOS: "linux", GOARCH: "amd64", Runner: "ubuntu-24.04", Gate: native.GatePR},
	{Name: "windows-amd64", GOOS: "windows", GOARCH: "amd64", Runner: "windows-2025", Gate: native.GateNative},
	{Name: "darwin-amd64", GOOS: "darwin", GOARCH: "amd64", Runner: "macos-15", Gate: native.GateNative},
}

func TestPlan(t *testing.T) {
	prOnly := []MatrixEntry{{"linux-amd64", "ubuntu-24.04", "linux", "amd64"}}
	all := []MatrixEntry{
		{"linux-amd64", "ubuntu-24.04", "linux", "amd64"},
		{"windows-amd64", "windows-2025", "windows", "amd64"},
		{"darwin-amd64", "macos-15", "darwin", "amd64"},
	}
	tests := []struct {
		event         string
		touchesNative bool
		want          []MatrixEntry
	}{
		{"pr", false, prOnly},
		{"pr", true, all},
		{"push", false, all},
		{"schedule", false, all},
		{"release", false, all},
	}
	for _, tt := range tests {
		got := plan(planTargets, tt.event, tt.touchesNative)
		if !reflect.DeepEqual(got.Include, tt.want) {
			t.Errorf("plan(%s, native=%v) = %v, want %v", tt.event, tt.touchesNative, got.Include, tt.want)
		}
	}
}

func TestPlanJSON(t *testing.T) {
	got, err := json.Marshal(plan(planTargets, "pr", false))
	if err != nil {
		t.Fatal(err)
	}
	want := `{"include":[{"target":"linux-amd64","runner":"ubuntu-24.04","goos":"linux","goarch":"amd64"}]}`
	if string(got) != want {
		t.Errorf("matrix JSON = %s, want %s", got, want)
	}
}
