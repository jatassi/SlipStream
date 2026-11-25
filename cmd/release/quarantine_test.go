package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jatassi/slipstream/cmd/release/internal/changescope"
)

const quarantineHeader = "tier\ttest\ttargets\tissue\tadded\texpires\n"

func quarantineList(t *testing.T, rows ...string) []quarantined {
	t.Helper()
	list, err := parseQuarantine([]byte(quarantineHeader + strings.Join(rows, "")))
	if err != nil {
		t.Fatal(err)
	}
	return list
}

func TestParseQuarantine(t *testing.T) {
	row := func(fields ...string) string { return strings.Join(fields, "\t") + "\n" }
	ok := row("go", "internal/link TestDial", "all", "712", "2026-11-20", "2026-12-04")
	var eleven strings.Builder
	for i := range 11 {
		eleven.WriteString(row("go", "internal/link TestDial"+strings.Repeat("x", i), "all", "712", "2026-11-20", "2026-12-04"))
	}
	for _, c := range []struct {
		name, file string
		// want is part of the error, empty for a file that loads.
		want string
		rows int
	}{
		{name: "no rows", file: quarantineHeader},
		{name: "a row of each runner", rows: 4, file: quarantineHeader + ok +
			row("browser", "web/e2e/work.spec.ts between Nodes › a Move keeps the files", "linux-amd64,darwin-arm64", "713", "2026-11-25", "2026-12-09") +
			row("component", "web/src/routes/_app/works/$workId.test.tsx Work > shows the files", "linux-amd64", "714", "2026-11-25", "2026-11-26") +
			row("platform", "web/e2e/home.spec.ts Home loads", "darwin-arm64", "715", "2026-11-25", "2026-12-09")},
		{name: "an expired row still loads", rows: 1, file: quarantineHeader + row("e2e", "e2e TestMove", "windows-amd64", "9", "2020-01-01", "2020-01-15")},
		{name: "no header", file: ok, want: "the first line is"},
		{name: "five columns", file: quarantineHeader + row("go", "internal/link TestDial", "all", "712", "2026-11-20"), want: "has 5 columns"},
		{name: "an unknown tier", file: quarantineHeader + row("online", "conformance/tmdb TestTMDB", "all", "712", "2026-11-20", "2026-12-04"), want: `the tier "online" isn't one of`},
		{name: "a test without a name", file: quarantineHeader + row("go", "internal/link", "all", "712", "2026-11-20", "2026-12-04"), want: "needs its path and its name"},
		{name: "a spec in the go tier", file: quarantineHeader + row("go", "web/e2e/work.spec.ts Search", "all", "712", "2026-11-20", "2026-12-04"), want: "the go tier doesn't run web/e2e/work.spec.ts"},
		{name: "a Vitest file in the browser tier", file: quarantineHeader + row("browser", "web/src/a.test.ts A", "all", "712", "2026-11-20", "2026-12-04"), want: "the browser tier doesn't run"},
		{name: "a target twice", file: quarantineHeader + row("go", "internal/link TestDial", "linux-amd64,linux-amd64", "712", "2026-11-20", "2026-12-04"), want: "distinct targets"},
		{name: "an unknown target", file: quarantineHeader + row("go", "internal/link TestDial", "linux", "712", "2026-11-20", "2026-12-04"), want: "distinct targets"},
		{name: "no issue", file: quarantineHeader + row("go", "internal/link TestDial", "all", "#712", "2026-11-20", "2026-12-04"), want: "isn't an issue number"},
		{name: "a date with a time", file: quarantineHeader + row("go", "internal/link TestDial", "all", "712", "2026-11-20T00:00", "2026-12-04"), want: "the added date"},
		{name: "15 days", file: quarantineHeader + row("go", "internal/link TestDial", "all", "712", "2026-11-20", "2026-12-05"), want: "by 2026-12-04"},
		{name: "expiring as added", file: quarantineHeader + row("go", "internal/link TestDial", "all", "712", "2026-11-20", "2026-11-20"), want: "after it was added"},
		{name: "a test twice", file: quarantineHeader + ok + row("go", "internal/link TestDial", "darwin-arm64", "713", "2026-11-21", "2026-12-04"), want: ":3: the go tier's internal/link TestDial has a row already"},
		{name: "11 rows", file: quarantineHeader + eleven.String(), want: "has 11 rows. It may have 10 at most"},
	} {
		list, err := parseQuarantine([]byte(c.file))
		switch {
		case c.want == "" && err != nil:
			t.Errorf("%s: %v", c.name, err)
		case c.want == "" && len(list) != c.rows:
			t.Errorf("%s: %d rows, want %d", c.name, len(list), c.rows)
		case c.want != "" && (err == nil || !strings.Contains(err.Error(), c.want)):
			t.Errorf("%s: error %v, want one with %q", c.name, err, c.want)
		}
	}
}

func TestTheQuarantineListLoads(t *testing.T) {
	root, err := repoRoot(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := loadQuarantine(root); err != nil {
		t.Fatal(err)
	}
}

// goRuns says how many of the runs run pkg's top-level test, as go test
// applies -run and -skip to each package it runs.
func goRuns(runs []goRun, pkg, test string) int {
	n := 0
	for _, r := range runs {
		if slices.Contains(r.packages, pkg) && (r.run == "" || regexp.MustCompile(r.run).MatchString(test)) && (r.skip == "" || !regexp.MustCompile(r.skip).MatchString(test)) {
			n++
		}
	}
	return n
}

// The blocking go tiers run every test once but the quarantined ones, which
// share their names with tests of other packages, and the quarantine tier
// runs those alone.
func TestGoTiersSkipTheQuarantinedTests(t *testing.T) {
	const a, b, c = modulePath + "/a", modulePath + "/b", modulePath + "/c"
	tests := map[string][]string{a: {"TestDial", "TestDialer", "TestMove"}, b: {"TestDial", "TestMove"}, c: {"TestDial", "TestHeavy", "TestMove"}}
	shard := []goRun{
		{packages: []string{a, b}},
		{packages: []string{c}, skip: testsPattern([]string{"TestHeavy"})},
	}
	ids := []string{"a TestDial", "c TestMove", "web/e2e/work.spec.ts Search"}
	quarantined := map[string]bool{a + " TestDial": true, c + " TestMove": true}
	for _, only := range []bool{false, true} {
		runs := skipQuarantined(shard, goQuarantine(ids))
		if only {
			runs = onlyQuarantinedRuns(goQuarantine(ids))
		}
		for pkg, names := range tests {
			for _, name := range names {
				want := 0
				switch {
				case only && quarantined[pkg+" "+name]:
					want = 1
				case !only && !quarantined[pkg+" "+name] && (pkg != c || name != "TestHeavy"):
					want = 1
				}
				if got := goRuns(runs, pkg, name); got != want {
					t.Errorf("only %t: %s %s runs %d times, want %d, in %+v", only, pkg, name, got, want, runs)
				}
			}
		}
	}
}

func TestBrowserTierFiltersTheQuarantinedTests(t *testing.T) {
	listed := []string{
		"[chromium-desktop] › work.spec.ts:7:1 › Search finds the release",
		"[firefox-desktop] › work.spec.ts:7:1 › Search finds the release",
		"[chromium-desktop] › work.spec.ts:12:3 › between Nodes › a Move keeps the files",
		"[chromium-desktop] › home.spec.ts:7:1 › Search finds the release",
		"[chromium-desktop] › work.spec.ts:20:1 › Search finds the release twice",
	}
	ids := []string{"web/e2e/work.spec.ts Search finds the release", "web/e2e/work.spec.ts between Nodes › a Move keeps the files", "internal/link TestDial"}
	if got, want := filterQuarantined(listed, ids, false), []string{listed[3], listed[4]}; !slices.Equal(got, want) {
		t.Errorf("the blocking run keeps %q, want %q", got, want)
	}
	if got, want := filterQuarantined(listed, ids, true), listed[:3]; !slices.Equal(got, want) {
		t.Errorf("the quarantine run keeps %q, want %q", got, want)
	}
}

func TestComponentTierFiltersTheQuarantinedTests(t *testing.T) {
	ids := []string{"web/src/queue.test.ts Queue > orders by age", "web/src/routes/_app/works/$workId.test.tsx Work > shows (the) files", "web/src/queue.test.ts drops a finished grab"}
	for _, c := range []struct {
		ids  []string
		only bool
		want [][]string
	}{
		{want: [][]string{{}}},
		{ids: ids, want: [][]string{
			{"--exclude", "src/queue.test.ts", "--exclude", `src/routes/_app/works/$workId.test.tsx`},
			{"src/queue.test.ts", "-t", "^(?!(Queue > orders by age|drops a finished grab)$)"},
			{`src/routes/_app/works/$workId.test.tsx`, "-t", `^(?!(Work > shows \(the\) files)$)`},
		}},
		{ids: ids, only: true, want: [][]string{
			{"src/queue.test.ts", "-t", "^(Queue > orders by age|drops a finished grab)$"},
			{`src/routes/_app/works/$workId.test.tsx`, "-t", `^(Work > shows \(the\) files)$`},
		}},
	} {
		got := vitestRuns(c.ids, c.only)
		if !slices.EqualFunc(got, c.want, slices.Equal) {
			t.Errorf("vitestRuns(%q, %t) = %q, want %q", c.ids, c.only, got, c.want)
		}
	}
}

// The quarantine tier reads each test's result from the runs it records:
// a test failed when it or its package did, and passed when it passed.
func TestQuarantineOutcomes(t *testing.T) {
	read := func(name string) []byte {
		b, err := os.ReadFile(filepath.Join("testdata", "failures", name))
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	for _, c := range []struct {
		name string
		r    results
		want map[string]string
	}{
		{"go", goResults(t, read("go-first.jsonl")), map[string]string{
			"widget TestPasses": "passed", "widget TestCounts": "failed", "widget TestShapes": "failed",
			"stall TestHangs": "failed", "widget TestGone": "not run",
		}},
		{"playwright", playwrightResults(t, read("playwright-first.json")), map[string]string{
			"web/e2e/work.spec.ts Refresh reads the source": "passed", "web/e2e/work.spec.ts between Nodes › a Move keeps the files": "failed",
		}},
		{"vitest", vitestResults(t, read("vitest-first.json")), map[string]string{
			"web/src/queue.test.ts Queue > lists the grabs": "passed", "web/src/queue.test.ts drops a finished grab": "failed",
			"web/src/broken.test.ts Broken > loads": "failed", "web/src/queue.test.ts Queue > lists": "not run",
		}},
	} {
		var ids []string
		for id := range c.want {
			ids = append(ids, id)
		}
		for _, o := range quarantineOutcomes("go", "linux-amd64", ids, c.r) {
			if o.Result != c.want[o.Test] {
				t.Errorf("%s: %s is %s, want %s", c.name, o.Test, o.Result, c.want[o.Test])
			}
		}
	}
}

func TestPlanRunsTheQuarantineLaneWhereARowReachesALane(t *testing.T) {
	row := func(tier, targets string) string {
		return tier + "\tinternal/link TestDial\t" + targets + "\t712\t2026-11-20\t2026-12-04\n"
	}
	for _, c := range []struct {
		name, kind string
		rows       []string
		keep       []string
		want       []string
	}{
		{name: "no rows", kind: "merge"},
		{name: "a go row on every target", kind: "merge", rows: []string{row("go", "all")}, want: []string{"linux-amd64", "linux-arm64", "windows-amd64", "darwin-arm64"}},
		{name: "a wave keeps its target", kind: "wave", keep: []string{"linux-amd64"}, rows: []string{row("go", "all")}, want: []string{"linux-amd64"}},
		{name: "a platform row reaches the platform targets", kind: "daily", rows: []string{row("platform", "all")}, want: []string{"linux-arm64", "windows-amd64", "darwin-arm64"}},
		{name: "a row for a target its lane skips", kind: "daily", rows: []string{row("platform", "linux-amd64")}},
		{name: "a dispatch runs no quarantine", kind: "dispatch", rows: []string{row("go", "all")}},
	} {
		p, err := plan(planInputs, runKinds[c.kind], changescope.PR, touched{}, c.keep, "all", testImage, quarantineList(t, c.rows...), nil)
		if err != nil {
			t.Fatal(err)
		}
		if got := laneTargets(p.Lanes["quarantine"]); !slices.Equal(got, c.want) {
			t.Errorf("%s: the quarantine lane runs on %v, want %v", c.name, got, c.want)
		}
		if got := p.PlannedJobs()["quarantine"]; got != (len(c.want) > 0) {
			t.Errorf("%s: planned quarantine = %t", c.name, got)
		}
	}
}

// The daily check reads the failures artifacts of a week of runs and the
// quarantine lane's results of 14 days of daily checks.
func TestDailyQuarantineDecisions(t *testing.T) {
	today := time.Date(2026, 11, 25, 0, 0, 0, 0, time.UTC)
	failures, err := readFailures(filepath.Join("testdata", "quarantine", "failures"))
	if err != nil {
		t.Fatal(err)
	}
	list := quarantineList(t,
		"go\tlisted TestListed\tlinux-amd64,windows-amd64\t700\t2026-11-20\t2026-12-04\n",
		"go\tnarrow TestNarrow\twindows-amd64\t701\t2026-11-20\t2026-12-04\n",
		"go\twidget TestSteady\tall\t702\t2026-11-10\t2026-11-24\n",
		"browser\tweb/e2e/work.spec.ts Refresh reads the source\tlinux-amd64\t703\t2026-11-11\t2026-11-25\n",
		"component\tweb/src/queue.test.ts Queue > lists the grabs\tlinux-amd64\t704\t2026-11-11\t2026-11-25\n",
		"go\twidget TestGone\tlinux-amd64\t705\t2026-11-12\t2026-11-26\n",
	)
	var proposed []string
	for _, p := range proposeQuarantine(list, failures, today) {
		p.row.issue = 750
		proposed = append(proposed, p.row.row())
	}
	if want := []string{
		"browser\tweb/e2e/work.spec.ts Search finds the release\tlinux-amd64\t750\t2026-11-25\t2026-12-09",
		"go\tnarrow TestNarrow\tlinux-amd64,windows-amd64\t750\t2026-11-25\t2026-12-09",
		"go\twidget TestCounts\tlinux-amd64,windows-amd64\t750\t2026-11-25\t2026-12-09",
	}; !slices.Equal(proposed, want) {
		t.Errorf("proposed rows:\n%s\nwant:\n%s", strings.Join(proposed, "\n"), strings.Join(want, "\n"))
	}
	if _, err := parseQuarantine([]byte(quarantineHeader + strings.Join(proposed, "\n") + "\n")); err != nil {
		t.Errorf("the proposed rows don't load: %v", err)
	}

	b, err := os.ReadFile(filepath.Join("testdata", "quarantine", "results-by-day.json"))
	if err != nil {
		t.Fatal(err)
	}
	var byDay map[string][]quarantineResult
	if err := json.Unmarshal(b, &byDay); err != nil {
		t.Fatal(err)
	}
	var ready []string
	for _, q := range readyToRemove(list, byDay, today) {
		ready = append(ready, q.test)
	}
	if want := []string{"widget TestSteady"}; !slices.Equal(ready, want) {
		t.Errorf("ready to remove: %q, want %q", ready, want)
	}

	var expired []string
	for _, q := range expiredRows(list, today) {
		expired = append(expired, q.test)
	}
	if want := []string{"widget TestSteady", "web/e2e/work.spec.ts Refresh reads the source", "web/src/queue.test.ts Queue > lists the grabs"}; !slices.Equal(expired, want) {
		t.Errorf("expired: %q, want %q", expired, want)
	}
	for _, s := range []string{"widget TestSteady", "#702", "2026-11-24"} {
		if is := expiredIssue(expiredRows(list, today), "run"); !strings.Contains(is.body, s) || !strings.Contains(expiredError(expiredRows(list, today)).Error(), s) {
			t.Errorf("the expired issue and error don't both name %q", s)
		}
	}
}

// The pull request, push and daily checks call quarantine.yml beside
// tests.yml, outside their gates, so a quarantined test's failure fails no
// check.
func TestTheQuarantineLaneRunsOutsideTheGates(t *testing.T) {
	for _, workflow := range []string{"pr.yml", "push.yml", "daily.yml", "release.yml", "run-tests.yml"} {
		jobs := readWorkflow(t, workflow)
		q, ok := jobs["quarantine"]
		if want := slices.Contains([]string{"pr.yml", "push.yml", "daily.yml"}, workflow); ok != want {
			t.Errorf("%s has a quarantine job: %t, want %t", workflow, ok, want)
			continue
		}
		if !ok {
			continue
		}
		if q.Uses != "./.github/workflows/quarantine.yml" || q.Needs != "plan" || q.With["plan"] != "${{ needs.plan.outputs.plan }}" {
			t.Errorf("%s: job quarantine uses %q, needs %v and takes plan %v, want quarantine.yml, plan and the plan output", workflow, q.Uses, q.Needs, q.With["plan"])
		}
		for id, j := range jobs {
			if needs, isList := j.Needs.([]any); isList && slices.Contains(needs, any("quarantine")) {
				t.Errorf("%s: job %s needs the quarantine job, which would fail it", workflow, id)
			}
		}
	}
}
