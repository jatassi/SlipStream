package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"
)

func readActions[T any](t *testing.T, name string) T {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "actions", name))
	if err != nil {
		t.Fatal(err)
	}
	var v T
	if err := json.Unmarshal(b, &v); err != nil {
		t.Fatal(err)
	}
	return v
}

func TestCIStatsMeasuresRecordedRuns(t *testing.T) {
	runs := readActions[[]ghRun](t, "runs.json")
	jobs := map[int64][]ghJob{}
	for _, j := range readActions[[]ghJob](t, "jobs.json") {
		jobs[j.RunID] = append(jobs[j.RunID], j)
	}
	daily := readActions[[]ghRun](t, "daily.json")
	lanes := map[laneTarget]laneCount{
		{"Plan", ""}:                          {5, 0},
		{"Build", "linux-amd64"}:              {1, 0},
		{"Go tests", "linux-amd64"}:           {3, 1},
		{"End-to-end tests", "windows-amd64"}: {2, 1},
		{"Browser tests", "linux-amd64"}:      {1, 1},
		{"Platform tests", "darwin-arm64"}:    {1, 0},
	}
	queue := map[string][]float64{"Linux": slices.Repeat([]float64{0.5}, 13), "Windows": {2, 2}, "macOS": {5}}
	labeled := []failure{
		{Tier: "go", Target: "linux-amd64", Rerun: flaky},
		{Tier: "e2e", Target: "windows-amd64", Rerun: flaky},
		{Tier: "e2e", Target: "windows-amd64", Rerun: failedAgain},
	}
	for _, tt := range []struct {
		name      string
		failures  []failure
		artifacts int
		flaky     map[laneTarget]int
	}{
		{"no artifacts", nil, 0, map[laneTarget]int{{"Go tests", "linux-amd64"}: 1}},
		{"artifacts", labeled, 2, map[laneTarget]int{{"go", "linux-amd64"}: 1, {"e2e", "windows-amd64"}: 1}},
	} {
		got := ciStats(runs, jobs, daily, tt.failures, tt.artifacts)
		want := stats{
			waveGreen: []float64{15}, waveRed: []float64{21}, mergeGreen: 1, mergeRuns: 2,
			lanes: lanes, flaky: tt.flaky, artifacts: tt.artifacts, streak: 2, streakConclusion: "failure", queue: queue,
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s: ciStats =\n%+v\nwant\n%+v", tt.name, got, want)
		}
	}
}

func TestPercentileTakesTheNearestRank(t *testing.T) {
	xs := []float64{9, 1, 5, 3, 7, 2, 8, 4, 10, 6}
	for _, tt := range []struct{ p, want float64 }{{0.5, 6}, {0.9, 10}, {0, 1}, {1, 10}} {
		if got := percentile(xs, tt.p); got != tt.want {
			t.Errorf("percentile(p%v) = %v, want %v", tt.p, got, tt.want)
		}
	}
}
