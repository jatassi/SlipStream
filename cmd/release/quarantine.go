package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"time"
)

// quarantinePath is the quarantine list: the flaky tests the blocking lanes
// leave out, which the quarantine lane runs alone until each is fixed.
const quarantinePath = "ci/quarantine.tsv"

const (
	maxQuarantined = 10
	// quarantineDays is the longest a row may quarantine its test.
	quarantineDays = 14
)

var quarantineColumns = []string{"tier", "test", "targets", "issue", "added", "expires"}

// quarantined is a row of the quarantine list.
type quarantined struct {
	tier string
	// test is the test's path and name, split by the first space: the Go
	// package's directory and a top-level test, a Playwright spec from the
	// repo root and its titles joined by " › ", or a Vitest file from the
	// repo root and its titles joined by " > ".
	test string
	// targets are the targets the row quarantines the test on, nil for all.
	targets []string
	issue   int
	// expires is the first day the row no longer holds.
	added, expires time.Time
}

func (q quarantined) on(target string) bool {
	return q.targets == nil || slices.Contains(q.targets, target)
}

// laneRunsOn says whether the lane of the row's tier runs on target, so a
// row for all targets reaches only those.
func (q quarantined) laneRunsOn(target string) bool {
	i := slices.IndexFunc(lanes, func(l lane) bool { return l.kind == q.tier })
	return q.on(target) && i >= 0 && lanes[i].runsOn(target)
}

func (q quarantined) row() string {
	targets := "all"
	if q.targets != nil {
		targets = strings.Join(q.targets, ",")
	}
	return strings.Join([]string{q.tier, q.test, targets, strconv.Itoa(q.issue), q.added.Format(time.DateOnly), q.expires.Format(time.DateOnly)}, "\t")
}

type testRunner int

const (
	goRunner testRunner = iota
	playwrightRunner
	vitestRunner
)

// runnerOf names the runner of the test at path.
func runnerOf(path string) testRunner {
	switch {
	case strings.HasPrefix(path, e2eDir+"/"):
		return playwrightRunner
	case strings.HasPrefix(path, "web/"):
		return vitestRunner
	default:
		return goRunner
	}
}

// tierRunners are the runners whose tests each quarantinable tier runs.
var tierRunners = map[string][]testRunner{
	"go":        {goRunner},
	"e2e":       {goRunner},
	"browser":   {playwrightRunner},
	"component": {vitestRunner},
	"platform":  {goRunner, playwrightRunner},
}

var listedTarget = regexp.MustCompile(`^(linux|windows|darwin)-(amd64|arm64)$`)

// loadQuarantine reads and checks the quarantine list under root. The
// check covers the file's shape alone, so an expired row still loads.
func loadQuarantine(root string) ([]quarantined, error) {
	b, err := os.ReadFile(filepath.Join(root, quarantinePath))
	if err != nil {
		return nil, fmt.Errorf("read the quarantine list: %w", err)
	}
	return parseQuarantine(b)
}

func parseQuarantine(b []byte) ([]quarantined, error) {
	lines := strings.Split(strings.TrimSuffix(string(b), "\n"), "\n")
	if header := strings.Join(quarantineColumns, "\t"); lines[0] != header {
		return nil, fmt.Errorf("%s: the first line is %q. It must be the column names, %q", quarantinePath, lines[0], header)
	}
	var rows []quarantined
	var errs []error
	for i, line := range lines[1:] {
		q, err := parseQuarantineRow(line)
		if err == nil && slices.ContainsFunc(rows, func(o quarantined) bool { return o.tier == q.tier && o.test == q.test }) {
			err = fmt.Errorf("the %s tier's %s has a row already. Give it one row with every target it needs", q.tier, q.test)
		}
		if err != nil {
			errs = append(errs, fmt.Errorf("%s:%d: %w", quarantinePath, i+2, err))
			continue
		}
		rows = append(rows, q)
	}
	if n := len(lines) - 1; n > maxQuarantined {
		errs = append(errs, fmt.Errorf("%s has %d rows. It may have %d at most. Fix a quarantined test before you add one", quarantinePath, n, maxQuarantined))
	}
	return rows, errors.Join(errs...)
}

func parseQuarantineRow(line string) (quarantined, error) {
	fields := strings.Split(line, "\t")
	if len(fields) != len(quarantineColumns) {
		return quarantined{}, fmt.Errorf("the row has %d columns. It needs %d, split by tabs: %s", len(fields), len(quarantineColumns), strings.Join(quarantineColumns, ", "))
	}
	q := quarantined{tier: fields[0], test: fields[1]}
	runners, ok := tierRunners[q.tier]
	if !ok {
		return quarantined{}, fmt.Errorf("the tier %q isn't one of %s", q.tier, strings.Join(slices.Sorted(maps.Keys(tierRunners)), ", "))
	}
	path, name, _ := strings.Cut(q.test, " ")
	if path == "" || name == "" {
		return quarantined{}, fmt.Errorf("the test %q needs its path and its name, split by a space", q.test)
	}
	if !slices.Contains(runners, runnerOf(path)) {
		return quarantined{}, fmt.Errorf("the %s tier doesn't run %s", q.tier, path)
	}
	if fields[2] != "all" {
		q.targets = strings.Split(fields[2], ",")
		for i, t := range q.targets {
			if !listedTarget.MatchString(t) || slices.Contains(q.targets[:i], t) {
				return quarantined{}, fmt.Errorf("the targets %q aren't all or a list of distinct targets such as linux-amd64,darwin-arm64", fields[2])
			}
		}
	}
	issue, err := strconv.Atoi(fields[3])
	if err != nil || issue < 1 {
		return quarantined{}, fmt.Errorf("the issue %q isn't an issue number such as 712", fields[3])
	}
	q.issue = issue
	for i, d := range []*time.Time{&q.added, &q.expires} {
		if *d, err = time.Parse(time.DateOnly, fields[4+i]); err != nil {
			return quarantined{}, fmt.Errorf("the %s date %q isn't a date such as 2026-11-25", quarantineColumns[4+i], fields[4+i])
		}
	}
	if latest := q.added.AddDate(0, 0, quarantineDays); !q.expires.After(q.added) || q.expires.After(latest) {
		return quarantined{}, fmt.Errorf("the row expires on %s. It must expire after it was added and by %s, %d days later", fields[5], latest.Format(time.DateOnly), quarantineDays)
	}
	return q, nil
}

// quarantinedIDs are the tests the list quarantines in tier on target.
func quarantinedIDs(list []quarantined, tier, target string) []string {
	var ids []string
	for _, q := range list {
		if q.tier == tier && q.on(target) {
			ids = append(ids, q.test)
		}
	}
	return ids
}

// testID names a test of a tier's results as the quarantine list does: a
// Go test by its package's directory and top-level test, a Playwright test
// without its project, and a Vitest test as its file and titles. A failure
// outside any test has no ID.
func testID(pkg, test string) string {
	if test == "" {
		return ""
	}
	pkg = strings.TrimPrefix(pkg, modulePath+"/")
	switch runnerOf(pkg) {
	case playwrightRunner:
		_, test, _ = strings.Cut(test, " › ")
	case goRunner:
		test, _, _ = strings.Cut(test, "/")
	case vitestRunner:
	}
	return pkg + " " + test
}

// goQuarantine maps each Go package's import path to the top-level tests
// the IDs name in it.
func goQuarantine(ids []string) map[string][]string {
	tests := map[string][]string{}
	for _, id := range ids {
		if dir, name, _ := strings.Cut(id, " "); runnerOf(dir) == goRunner {
			pkg := modulePath + "/" + dir
			tests[pkg] = append(tests[pkg], name)
		}
	}
	return tests
}

// skipQuarantined gives each package with quarantined tests a go test of
// its own that skips them, since go test -skip leaves a test out of every
// package the go test runs.
func skipQuarantined(runs []goRun, skip map[string][]string) []goRun {
	var out []goRun
	for _, r := range runs {
		rest := goRun{run: r.run, skip: r.skip}
		var alone []goRun
		for _, pkg := range r.packages {
			names, ok := skip[pkg]
			if !ok {
				rest.packages = append(rest.packages, pkg)
				continue
			}
			pattern := testsPattern(names)
			if r.skip != "" {
				pattern = r.skip + "|" + pattern
			}
			alone = append(alone, goRun{packages: []string{pkg}, run: r.run, skip: pattern})
		}
		if len(rest.packages) > 0 {
			out = append(out, rest)
		}
		out = append(out, alone...)
	}
	return out
}

// onlyQuarantinedRuns is a go test per package of its quarantined tests.
func onlyQuarantinedRuns(tests map[string][]string) []goRun {
	var runs []goRun
	for _, pkg := range slices.Sorted(maps.Keys(tests)) {
		runs = append(runs, goRun{packages: []string{pkg}, run: testsPattern(tests[pkg])})
	}
	return runs
}

// browserTestID is a playwright --list line's test as the quarantine list
// names it.
func browserTestID(listed string) string {
	_, rest, _ := strings.Cut(browserTestKey(listed), " › ")
	file, titles, _ := strings.Cut(rest, " › ")
	return e2eDir + "/" + file + " " + titles
}

// filterQuarantined drops the listed browser tests the IDs name, or with
// only keeps those alone. Each test of every project goes or stays alike.
func filterQuarantined(listed, ids []string, only bool) []string {
	return slices.DeleteFunc(slices.Clone(listed), func(l string) bool { return slices.Contains(ids, browserTestID(l)) != only })
}

// vitestRuns are the Vitest runs that leave out the quarantined tests, or
// with only run them alone. -t matches a test's titles in every file it
// runs, so each file with quarantined tests runs on its own, and the
// first run excludes those files.
func vitestRuns(ids []string, only bool) [][]string {
	byFile := map[string][]string{}
	for _, id := range ids {
		path, name, _ := strings.Cut(id, " ")
		file := strings.TrimPrefix(path, "web/")
		byFile[file] = append(byFile[file], name)
	}
	var runs [][]string
	if !only {
		rest := []string{}
		for _, file := range slices.Sorted(maps.Keys(byFile)) {
			rest = append(rest, "--exclude", globEscapes.Replace(file))
		}
		runs = append(runs, rest)
	}
	for _, file := range slices.Sorted(maps.Keys(byFile)) {
		pattern := testsPattern(byFile[file])
		if !only {
			pattern = "^(?!" + strings.TrimPrefix(pattern, "^") + ")"
		}
		runs = append(runs, []string{file, "-t", pattern})
	}
	return runs
}

var globEscapes = strings.NewReplacer(`\`, `\\`, "*", `\*`, "?", `\?`, "[", `\[`, "]", `\]`, "{", `\{`, "}", `\}`, "(", `\(`, ")", `\)`, "!", `\!`, "+", `\+`, "@", `\@`)

func init() {
	// Appended here because testQuarantined runs the other tiers of the
	// list, which can't name it.
	testTiers = append(testTiers, testTier{name: "quarantine", run: testQuarantined, subset: true})
}

// quarantineResult is how a quarantined test did in the quarantine lane.
type quarantineResult struct {
	Tier   string `json:"tier"`
	Test   string `json:"test"`
	Target string `json:"target"`
	// Result is passed, failed, or not run when the tier ran no test of
	// that name.
	Result string `json:"result"`
}

// testQuarantined runs each tier's quarantined tests on this target alone,
// and writes each one's result to o.results.
func testQuarantined(ctx context.Context, root string, o testOptions) (results, error) {
	list, err := loadQuarantine(root)
	if err != nil {
		return results{}, err
	}
	target := ciTarget()
	var all results
	var outcomes []quarantineResult
	var errs []error
	for _, t := range testTiers {
		ids := quarantinedIDs(list, t.name, target)
		if !t.quarantine || len(ids) == 0 {
			continue
		}
		fmt.Printf("The quarantine tier runs the tests %s lists for the %s tier on %s:\n  %s\n", quarantinePath, t.name, target, strings.Join(ids, "\n  "))
		// A Linux runner outside Playwright's container lacks the browsers'
		// system libraries, which the browsers' own install adds.
		browsers := inCI() && runtime.GOOS == "linux" && slices.ContainsFunc(tierRunners[t.name], func(r testRunner) bool { return r != goRunner })
		r, err := t.run(ctx, root, testOptions{repeat: 1, quarantined: ids, onlyQuarantined: true, systemDeps: browsers})
		for i := range r.failed {
			r.failed[i].Tier = t.name
		}
		all.add(r)
		outcomes = append(outcomes, quarantineOutcomes(t.name, target, ids, r)...)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s tier: %w", t.name, err))
		}
	}
	if o.results != "" {
		if outcomes == nil {
			outcomes = []quarantineResult{}
		}
		if err := writeJSON(o.results, outcomes); err != nil {
			errs = append(errs, fmt.Errorf("write the quarantined tests' results: %w", err))
		}
	}
	return all, errors.Join(errs...)
}

// quarantineOutcomes is each quarantined test's result in r: failed when
// it or its package failed, passed when it passed, and not run otherwise.
func quarantineOutcomes(tier, target string, ids []string, r results) []quarantineResult {
	passed := map[string]bool{}
	for k := range r.passed {
		pkg, test, _ := strings.Cut(k, "\x00")
		passed[testID(pkg, test)] = true
	}
	var out []quarantineResult
	for _, id := range ids {
		path, _, _ := strings.Cut(id, " ")
		res := quarantineResult{Tier: tier, Test: id, Target: target, Result: "not run"}
		switch {
		case slices.ContainsFunc(r.failed, func(f failure) bool {
			return testID(f.Package, f.Test) == id || f.Test == "" && strings.TrimPrefix(f.Package, modulePath+"/") == path
		}):
			res.Result = "failed"
		case passed[id]:
			res.Result = "passed"
		}
		out = append(out, res)
	}
	return out
}

func writeJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "\t")
	if err != nil {
		return fmt.Errorf("encode %s: %w", path, err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	if err := os.WriteFile(path, append(b, '\n'), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}
