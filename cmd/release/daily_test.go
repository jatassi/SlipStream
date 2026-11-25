package main

import (
	"maps"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestDecideDailyRunsTheFullMatrixUnlessThisTreePassedItThisWeek(t *testing.T) {
	now := time.Date(2026, 11, 25, 7, 30, 0, 0, time.UTC)
	green := func(id int64, tree string, age time.Duration) ghRun {
		r := ghRun{ID: id, HeadSHA: "sha" + tree, CreatedAt: now.Add(-age)}
		r.HeadCommit.TreeID = tree
		return r
	}
	built := []ghJob{{Name: "Tests / linux-amd64 / Build (linux-amd64)", Conclusion: "success"}, {Name: "Tests / Online conformance tests", Conclusion: "success"}}
	onlineOnly := []ghJob{{Name: "Plan", Conclusion: "success"}, {Name: "Tests / Online conformance tests", Conclusion: "success"}}
	day := 24 * time.Hour
	for _, tt := range []struct {
		name  string
		green []ghRun
		jobs  map[int64][]ghJob
		kind  string
		since string
	}{
		{"no green run", nil, nil, "schedule", ""},
		{"green runs of the online tests alone", []ghRun{green(1, "t", day)}, map[int64][]ghJob{1: onlineOnly}, "schedule", ""},
		{"this tree a day ago", []ghRun{green(1, "t", day)}, map[int64][]ghJob{1: built}, "daily", "shat"},
		{"this tree just under a week ago", []ghRun{green(1, "t", 7*day-time.Minute)}, map[int64][]ghJob{1: built}, "daily", "shat"},
		{"this tree a week ago", []ghRun{green(1, "t", 7*day)}, map[int64][]ghJob{1: built}, "schedule", "shat"},
		{"an older tree a day ago", []ghRun{green(1, "old", day)}, map[int64][]ghJob{1: built}, "schedule", "shaold"},
		{
			"newer online runs over this tree's full run",
			[]ghRun{green(3, "t", day), green(2, "t", 2*day), green(1, "old", 3*day)},
			map[int64][]ghJob{3: onlineOnly, 2: built, 1: built}, "daily", "shat",
		},
	} {
		ranFull := func(r ghRun) (bool, error) { return ranFullMatrix(tt.jobs[r.ID]), nil }
		got, err := decideDaily(tt.green, ranFull, "t", now)
		if err != nil {
			t.Fatal(err)
		}
		if got.kind != tt.kind || got.since != tt.since {
			t.Errorf("%s: decideDaily = kind %s since %q, want kind %s since %q", tt.name, got.kind, got.since, tt.kind, tt.since)
		}
	}
}

func TestDailyIssues(t *testing.T) {
	jobs := []ghJob{
		{Name: "Plan", Conclusion: "success"},
		{Name: "Tests / windows-amd64 / End-to-end tests (windows-amd64, shard 2 of 4)", Conclusion: "failure"},
		{Name: "Tests / Online conformance tests", Conclusion: "failure"},
		{Name: "Link throughput test", Conclusion: "failure"},
	}
	online := []failure{
		{Tier: "online", Package: "github.com/jatassi/slipstream/conformance/tmdb", Test: "TestTMDB/search", Message: "shape changed"},
		{Tier: "online", Package: "github.com/jatassi/slipstream/conformance/tvdb", Test: "TestTVDB", Message: "401"},
		{Tier: "e2e", Package: "github.com/jatassi/slipstream/e2e", Test: "TestX"},
	}
	commits := []string{"abc1234 feat(a): one (#701)", "def5678 fix(b): two (#702)"}
	daily := "Daily check failed"
	for _, tt := range []struct {
		name     string
		jobs     []ghJob
		failures []failure
		since    string
		want     map[string][]string
		// without are the jobs no issue names, since they file their own.
		without []string
	}{
		{"every kind", jobs, online, "0123456789abcdef", map[string][]string{
			daily: {"End-to-end tests (windows-amd64, shard 2 of 4)", "0123456789ab", commits[0], commits[1]},
			"Online conformance tests failed for tmdb": {"TestTMDB/search shape changed"},
			"Online conformance tests failed for tvdb": {"TestTVDB 401"},
		}, []string{"- Online conformance tests", "- Link throughput test"}},
		{"an online job that named no test", jobs[2:], nil, "", map[string][]string{daily: {"Online conformance tests", "No daily check on main has passed the full matrix yet."}}, []string{"- Link throughput test"}},
		{"the link alone", jobs[3:], nil, "", map[string][]string{}, nil},
	} {
		got := dailyIssues(tt.jobs, tt.failures, commits, tt.since, "https://github.com/o/r/actions/runs/1")
		var titles []string
		for _, is := range got {
			titles = append(titles, is.title)
			for _, s := range tt.without {
				if strings.Contains(is.body, s) {
					t.Errorf("%s: issue %q names %q:\n%s", tt.name, is.title, s, is.body)
				}
			}
			for _, s := range tt.want[is.title] {
				if !strings.Contains(is.body, s) {
					t.Errorf("%s: issue %q doesn't name %q:\n%s", tt.name, is.title, s, is.body)
				}
			}
		}
		if want := slices.Sorted(maps.Keys(tt.want)); !slices.Equal(slices.Sorted(slices.Values(titles)), want) {
			t.Errorf("%s: dailyIssues titles %v, want %v", tt.name, titles, want)
		}
	}
}
