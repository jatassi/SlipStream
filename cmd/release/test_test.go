package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"maps"
	"math"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestSplitOwnFlags(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.String("tier", "", "")
	fs.String("shard", "", "")
	for _, c := range []struct {
		args, own, rest []string
	}{
		{nil, []string{}, []string{}},
		{[]string{"-tier", "go", "-run", "Foo", "./e2e"}, []string{"-tier", "go"}, []string{"-run", "Foo", "./e2e"}},
		{[]string{"--tier=browser", "-shard", "2/4"}, []string{"--tier=browser", "-shard", "2/4"}, []string{}},
		{[]string{"-run", "Foo", "-tier", "go"}, []string{}, []string{"-run", "Foo", "-tier", "go"}},
		{[]string{"-tier"}, []string{"-tier"}, []string{}},
	} {
		own, rest := splitOwnFlags(fs, c.args)
		if !slices.Equal(own, c.own) || !slices.Equal(rest, c.rest) {
			t.Errorf("splitOwnFlags(%q) = %q, %q, want %q, %q", c.args, own, rest, c.own, c.rest)
		}
	}
}

func TestTestRefusesFlagsTheTierDoesNotTake(t *testing.T) {
	for _, c := range []struct {
		args []string
		want string
	}{
		{[]string{"-run", "Foo"}, `go test arguments ["-run" "Foo"] need -tier go, e2e or online`},
		{[]string{"-tier", "component", "./..."}, `go test arguments ["./..."] need -tier go, e2e or online`},
		{[]string{"-tier", "lint"}, `-tier "lint" is not one of go, e2e, component, diagram, browser, online, all`},
		{[]string{"-shard", "1/2"}, "-shard, -filter and -repeat need -tier go, e2e or browser"},
		{[]string{"-tier", "component", "-filter", "Foo"}, "-shard, -filter and -repeat need -tier go, e2e or browser"},
		{[]string{"-tier", "component", "-timings", "t"}, "-timings and -record need -tier go, e2e or browser"},
		{[]string{"-tier", "go", "-shard", "1/2", "./internal/..."}, `go test arguments ["./internal/..."] pick the tests themselves, so they take none of -shard, -filter, -repeat, -timings or -record`},
		{[]string{"-tier", "go", "-repeat", "0"}, "-repeat 0 is not a positive count"},
		{[]string{"-tier", "go", "-shard", "3/2"}, `parse flags: invalid value "3/2" for flag -shard: shard "3/2" is not i/n with 1 <= i <= n`},
	} {
		if err := runTest(t.Context(), c.args); err == nil || err.Error() != c.want {
			t.Errorf("runTest(%q) = %v, want %s", c.args, err, c.want)
		}
	}
}

func TestPlaywrightRuns(t *testing.T) {
	for _, c := range []struct {
		o    testOptions
		want [][]string
	}{
		{testOptions{shard: shard{2, 3}, filter: "@phone", repeat: 5}, [][]string{
			{"--grep-invert=@alone", "--pass-with-no-tests", "--test-list=shard.txt", "--repeat-each=5", "--grep=@phone"},
			{"--workers=1", "--grep=(?=.*@alone)(?=.*(?:@phone))", "--pass-with-no-tests", "--test-list=shard.txt", "--repeat-each=5"},
		}},
		{testOptions{repeat: 1}, [][]string{
			{"--grep-invert=@alone", "--pass-with-no-tests", "--test-list=shard.txt"},
			{"--workers=1", "--grep=@alone", "--pass-with-no-tests", "--test-list=shard.txt"},
		}},
	} {
		got := playwrightRuns(c.o, "shard.txt")
		if !slices.EqualFunc(got, c.want, slices.Equal) {
			t.Errorf("playwrightRuns(%+v) = %q, want %q", c.o, got, c.want)
		}
	}
}

func browserTests(n int) []string {
	tests := make([]string, n)
	for i := range tests {
		tests[i] = fmt.Sprintf("[chromium-desktop] › a.spec.ts:%d:1 › test %d", i+1, i)
	}
	return tests
}

func shardsOf(tests []string, seconds map[string]float64, n int) [][]string {
	shards := make([][]string, n)
	for i := range shards {
		shards[i] = shardBrowserTests(tests, seconds, shard{i + 1, n})
	}
	return shards
}

// TestBrowserShardsRunEveryTestOnce covers a run with no timings, one with
// timings for only some of its tests, a filter that leaves fewer tests than
// shards, and timings of nothing but zeros.
func TestBrowserShardsRunEveryTestOnce(t *testing.T) {
	tests := browserTests(100)
	some := map[string]float64{"gone": 500}
	zeros := map[string]float64{}
	for i, name := range tests {
		zeros[browserTestKey(name)] = 0
		if i%3 == 0 {
			some[browserTestKey(name)] = float64(i * i)
		}
	}
	for name, c := range map[string]struct {
		tests   []string
		seconds map[string]float64
	}{
		"no timings": {tests, nil},
		"partial":    {tests, some},
		"few tests":  {tests[:2], some},
		"no tests":   {nil, some},
		"zeros":      {tests, zeros},
	} {
		for n := 1; n <= 4; n++ {
			count := map[string]int{}
			for _, s := range shardsOf(c.tests, c.seconds, n) {
				for _, test := range s {
					count[test]++
				}
			}
			if len(count) != len(c.tests) || slices.ContainsFunc(c.tests, func(test string) bool { return count[test] != 1 }) {
				t.Errorf("%s: %d shards run %v, want each of %d tests once", name, n, count, len(c.tests))
			}
		}
	}
}

func TestBrowserShardsWithoutTimingsCutEvenly(t *testing.T) {
	for _, s := range shardsOf(browserTests(100), nil, 4) {
		if len(s) != 25 {
			t.Errorf("a shard has %d tests, want 25", len(s))
		}
	}
}

func TestBrowserShardsCutBySecondsInOrder(t *testing.T) {
	tests := browserTests(5)
	seconds := map[string]float64{}
	for i, s := range []float64{90, 30, 30, 30, 0} {
		seconds[browserTestKey(tests[i])] = s
	}
	got := shardsOf(tests, seconds, 2)
	if want := tests[:2]; !slices.Equal(got[0], want) {
		t.Errorf("shard 1 = %q, want %q", got[0], want)
	}
	if want := tests[2:]; !slices.Equal(got[1], want) {
		t.Errorf("shard 2 = %q, want %q, with the test the timings have not seen", got[1], want)
	}
}

func TestBrowserTimingsNameTestsAsTheListDoes(t *testing.T) {
	const report = `{"stats": {"duration": 30000}, "suites": [{
		"title": "a.spec.ts",
		"specs": [{"title": "top", "file": "a.spec.ts", "tests": [
			{"projectName": "chromium-desktop", "results": [{"duration": 2000}]},
			{"projectName": "webkit-phone", "results": [{"duration": 3000}]}
		]}],
		"suites": [{"title": "between Nodes", "specs": [
			{"title": "nested", "file": "a.spec.ts", "tests": [{"projectName": "chromium-desktop", "results": [{"duration": 5000}]}]}
		]}]
	}]}`
	var r playwrightReport
	if err := json.Unmarshal([]byte(report), &r); err != nil {
		t.Fatal(err)
	}
	got := r.seconds()
	listed := map[string]float64{
		"[chromium-desktop] › a.spec.ts:3:1 › top":                    6,
		"[webkit-phone] › a.spec.ts:3:1 › top":                        9,
		"[chromium-desktop] › a.spec.ts:9:3 › between Nodes › nested": 15,
	}
	want := map[string]float64{}
	for name, s := range listed {
		want[browserTestKey(name)] = s
	}
	if !maps.EqualFunc(got, want, func(a, b float64) bool { return math.Abs(a-b) < 1e-9 }) {
		t.Errorf("seconds = %v, want %v, the 30s run shared by duration", got, want)
	}
}

func TestRecordBrowserTimingsReadsBackForTheNextRun(t *testing.T) {
	dir := t.TempDir()
	report := filepath.Join(dir, "report.json")
	const body = `{"stats": {"duration": 4000}, "suites": [{"title": "a.spec.ts", "specs": [
		{"title": "one", "file": "a.spec.ts", "tests": [{"projectName": "p", "results": [{"duration": 1000}]}]}
	]}]}`
	if err := os.WriteFile(report, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	// A run that failed before the second report leaves that one missing.
	if err := recordBrowserTimings(filepath.Join(dir, "timings", "t.json"), []string{report, report + ".missing"}); err != nil {
		t.Fatal(err)
	}
	past, err := loadTimings(filepath.Join(dir, "timings"), browserTarget())
	if err != nil {
		t.Fatal(err)
	}
	if got := past.Packages[browserTimings].Tests[browserTestKey("[p] › a.spec.ts:1:1 › one")]; got != 4 {
		t.Errorf("the test's seconds = %v, want 4", got)
	}
}
