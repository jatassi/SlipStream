package main

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jatassi/slipstream/cmd/release/internal/native"
)

func init() {
	register("test", "run the test tiers: -tier go|e2e|component|diagram|browser|online|all (default all, which leaves out online); -tier go and e2e pass the rest to go test -race -shuffle=on, -tier online to go test -tags online ./conformance/...", runTest)
}

type testTier struct {
	name string
	// run returns the tests that failed, which CI reports.
	run func(ctx context.Context, root string, o testOptions) ([]failure, error)
	// goArgs says the tier hands the arguments after its flags to go test.
	goArgs bool
	// sharded tiers take -shard, -filter and -repeat.
	sharded bool
	// timed tiers split their shards by -timings and write -record.
	timed bool
	// online tiers reach real services, so all leaves them out.
	online bool
}

var testTiers = []testTier{
	{name: "go", run: testGo, goArgs: true, sharded: true, timed: true},
	{name: "e2e", run: testE2E, goArgs: true, sharded: true, timed: true},
	{name: "component", run: testComponent},
	{name: "diagram", run: testDiagram},
	{name: "browser", run: testBrowser, sharded: true, timed: true},
	{name: "online", run: testOnline, goArgs: true, online: true},
}

type testOptions struct {
	shard   shard
	filter  string
	repeat  int
	timings string
	record  string
	// deadline ends the browser tier's Playwright runs before CI's job
	// limit. A shard that outgrows it fails with a report naming the tests
	// it never finished and keeps the timings of those it did. A job cut off
	// at its limit records nothing, so its tests stay unseen and the next
	// split overruns again.
	deadline time.Time
	goArgs   []string
	// failures is where the failed tests go as JSON.
	failures string
}

// shard is the index-th of count, from 1. The zero shard is the whole tier.
type shard struct{ index, count int }

func (s *shard) String() string { return fmt.Sprintf("%d/%d", s.index, s.count) }

func (s *shard) Set(v string) error {
	i, n, ok := strings.Cut(v, "/")
	index, err1 := strconv.Atoi(i)
	count, err2 := strconv.Atoi(n)
	if !ok || err1 != nil || err2 != nil || index < 1 || index > count {
		return fmt.Errorf("shard %q is not i/n with 1 <= i <= n", v)
	}
	*s = shard{index, count}
	return nil
}

// runTest reads its own flags ahead of go test's, so `release test -tier
// go -run Foo ./e2e` hands `-run Foo ./e2e` to go test.
func runTest(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	tier := fs.String("tier", "all", "go, e2e, component, diagram, browser, online or all")
	var o testOptions
	fs.Var(&o.shard, "shard", "run the i-th of n shards of the tier, as i/n")
	fs.StringVar(&o.filter, "filter", "", "run only the tests that match, as go test -run or playwright --grep")
	fs.IntVar(&o.repeat, "repeat", 1, "run each test this many times, as go test -count or playwright --repeat-each")
	fs.StringVar(&o.timings, "timings", "", "directory of earlier runs' timings to split the shards by")
	fs.StringVar(&o.record, "record", "", "write this run's timings to this file")
	fs.StringVar(&o.failures, "failures", "", "write the failed tests to this file as JSON")
	fs.Func("deadline", "end the browser tier's runs at this Unix time, in seconds, so a shard that outgrows its job still reports", func(v string) error {
		sec, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return fmt.Errorf("-deadline %q is not a Unix time in seconds", v)
		}
		o.deadline = time.Unix(sec, 0)
		return nil
	})
	ownArgs, goArgs := splitOwnFlags(fs, args)
	if err := fs.Parse(ownArgs); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}
	o.goArgs = goArgs
	tiers := slices.DeleteFunc(slices.Clone(testTiers), func(t testTier) bool { return t.online })
	if *tier != "all" {
		i := slices.IndexFunc(testTiers, func(t testTier) bool { return t.name == *tier })
		if i < 0 {
			return fmt.Errorf("-tier %q is not one of go, e2e, component, diagram, browser, online, all", *tier)
		}
		tiers = testTiers[i : i+1]
	}
	if err := o.fits(tiers); err != nil {
		return err
	}
	root, err := repoRoot(ctx)
	if err != nil {
		return err
	}
	var failed []string
	var failures []failure
	for _, t := range tiers {
		tierFailures, err := t.run(ctx, root, o)
		for i := range tierFailures {
			tierFailures[i].Tier, tierFailures[i].Target = t.name, ciTarget()
		}
		failures = append(failures, tierFailures...)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s tier: %v\n", t.name, err)
			failed = append(failed, t.name)
		}
	}
	reportErr := reportFailures(failures, o.failures)
	if len(failed) > 0 {
		return errors.Join(reportErr, fmt.Errorf("failed: %s", strings.Join(failed, ", ")))
	}
	if reportErr != nil {
		return reportErr
	}
	return nil
}

func (o testOptions) fits(tiers []testTier) error {
	one := len(tiers) == 1
	switch {
	case len(o.goArgs) > 0 && (!one || !tiers[0].goArgs):
		return fmt.Errorf("go test arguments %q need -tier go, e2e or online", o.goArgs)
	case (o.shard != shard{} || o.filter != "" || o.repeat != 1) && (!one || !tiers[0].sharded):
		return errors.New("-shard, -filter and -repeat need -tier go, e2e or browser")
	case (o.timings != "" || o.record != "") && (!one || !tiers[0].timed):
		return errors.New("-timings and -record need -tier go, e2e or browser")
	case len(o.goArgs) > 0 && (o.shard != shard{} || o.filter != "" || o.repeat != 1 || o.timings != "" || o.record != ""):
		return fmt.Errorf("go test arguments %q pick the tests themselves, so they take none of -shard, -filter, -repeat, -timings or -record", o.goArgs)
	case !o.deadline.IsZero() && (!one || tiers[0].name != "browser"):
		return errors.New("-deadline needs -tier browser")
	case o.repeat < 1:
		return fmt.Errorf("-repeat %d is not a positive count", o.repeat)
	}
	return nil
}

// splitOwnFlags splits the leading flags fs defines from the arguments
// that follow, since a FlagSet refuses go test's flags. Each of fs's flags
// takes a value.
func splitOwnFlags(fs *flag.FlagSet, args []string) (own, rest []string) {
	i := 0
	for i < len(args) {
		name, _, hasValue := strings.Cut(strings.TrimLeft(args[i], "-"), "=")
		if !strings.HasPrefix(args[i], "-") || fs.Lookup(name) == nil {
			break
		}
		i++
		if !hasValue {
			i++
		}
	}
	i = min(i, len(args))
	return args[:i], args[i:]
}

func testGo(ctx context.Context, root string, o testOptions) ([]failure, error) {
	return testGoPackages(ctx, root, o, "go")
}

// testE2E runs the packages whose tests drive the slipstream, slipstream-e2e
// and fakeworld binaries through fakes/testbin, which CI hands the build
// job's dist/ through TEST_SLIPSTREAM_BINARY, TEST_SLIPSTREAM_E2E_BINARY and
// TEST_FAKEWORLD_BINARY.
func testE2E(ctx context.Context, root string, o testOptions) ([]failure, error) {
	return testGoPackages(ctx, root, o, "e2e")
}

// testOnline runs every conformance suite against the real service it
// checks. TEST_ONLINE_TMDB_KEY carries TMDB's key, TEST_ONLINE_TVDB_KEY
// TVDB's and TEST_ONLINE_OMDB_KEY OMDb's; -update rewrites each suite's
// shape.json, and runs only the suites that record one, since a test binary
// that defines no -update fails on it.
func testOnline(ctx context.Context, root string, o testOptions) ([]failure, error) {
	pkgs := []string{"./conformance/..."}
	if slices.Contains(o.goArgs, "-update") {
		var err error
		if pkgs, err = shapeSuites(ctx, root); err != nil {
			return nil, err
		}
	}
	args := slices.Concat([]string{"test", "-tags", "online", "-count=1"}, pkgs, o.goArgs)
	gotest := exec.CommandContext(ctx, "go", args...)
	gotest.Dir = root
	gotest.Stdout, gotest.Stderr = os.Stdout, os.Stderr
	if err := gotest.Run(); err != nil {
		return nil, fmt.Errorf("go test: %w", err)
	}
	return nil, nil
}

// shapeSuites lists the conformance packages whose tests import the shape
// package, which defines -update.
func shapeSuites(ctx context.Context, root string) ([]string, error) {
	const shapePkg = "github.com/jatassi/slipstream/conformance/internal/shape"
	list := exec.CommandContext(ctx, "go", "list", "-tags", "online", "-f", "{{.ImportPath}}{{range .TestImports}} {{.}}{{end}}{{range .XTestImports}} {{.}}{{end}}", "./conformance/...")
	list.Dir = root
	list.Stderr = os.Stderr
	out, err := list.Output()
	if err != nil {
		return nil, fmt.Errorf("go list: %w", err)
	}
	var pkgs []string
	for line := range strings.Lines(string(out)) {
		fields := strings.Fields(line)
		if len(fields) > 1 && slices.Contains(fields[1:], shapePkg) {
			pkgs = append(pkgs, fields[0])
		}
	}
	return pkgs, nil
}

func testComponent(ctx context.Context, root string, _ testOptions) ([]failure, error) {
	if err := installChromium(ctx, root); err != nil {
		return nil, err
	}
	first, err := runVitest(ctx, root, nil)
	failed := rerunFailed(first, func(f []failure) results {
		again, _ := runVitest(ctx, root, vitestRerunArgs(f))
		return again
	})
	return failed, err
}

// vitestRerunArgs picks the failed tests by their files and their full
// names, which a test's line can't do: the browser mode's transforms move
// it.
func vitestRerunArgs(failed []failure) []string {
	var files, names []string
	for _, f := range failed {
		if file := strings.TrimPrefix(f.File, "web/"); !slices.Contains(files, file) {
			files = append(files, file)
		}
		names = append(names, f.selector)
	}
	return append(files, "-t", testsPattern(names))
}

// runVitest runs the component tests the arguments pick, or every one.
func runVitest(ctx context.Context, root string, filters []string) (results, error) {
	dir, err := os.MkdirTemp("", "vitest-")
	if err != nil {
		return results{}, fmt.Errorf("make the Vitest report dir: %w", err)
	}
	report := filepath.Join(dir, "report.json")
	args := slices.Concat([]string{"run", "test", "--reporter=default", "--reporter=json", "--outputFile.json=" + report, "--includeTaskLocation"}, filters)
	runErr := run(ctx, filepath.Join(root, "web"), nil, "bun", args...)
	r, err := readVitestReport(report, root)
	return r, errors.Join(runErr, err)
}

// vitestReport is the part of Vitest's JSON report the failures read.
type vitestReport struct {
	TestResults []struct {
		// Name is the test file's absolute path.
		Name string `json:"name"`
		// Message says why a file failed outside its tests.
		Message          string `json:"message"`
		Status           string `json:"status"`
		AssertionResults []struct {
			AncestorTitles  []string `json:"ancestorTitles"`
			Title           string   `json:"title"`
			FullName        string   `json:"fullName"`
			Status          string   `json:"status"`
			FailureMessages []string `json:"failureMessages"`
			Location        *struct {
				Line int `json:"line"`
			} `json:"location"`
		} `json:"assertionResults"`
	} `json:"testResults"`
}

// readVitestReport reads the failures of the report at path, which a run
// that never started leaves missing. A failed test runs again by the name
// Vitest's -t matches, its suites' and its titles joined by " > ".
func readVitestReport(path, root string) (results, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return results{}, nil
	}
	if err != nil {
		return results{}, fmt.Errorf("read the Vitest report: %w", err)
	}
	var report vitestReport
	if err := json.Unmarshal(b, &report); err != nil {
		return results{}, fmt.Errorf("%s: %w", path, err)
	}
	var r results
	for _, file := range report.TestResults {
		rel, err := filepath.Rel(root, file.Name)
		if err != nil {
			return results{}, fmt.Errorf("read the Vitest report: %w", err)
		}
		rel = filepath.ToSlash(rel)
		testFailed := false
		for _, a := range file.AssertionResults {
			f := failure{Package: rel, File: rel, Test: a.FullName}
			switch a.Status {
			case "passed":
				r.pass(f.key())
				continue
			case "failed":
			default:
				continue
			}
			testFailed = true
			if len(a.FailureMessages) > 0 {
				f.Message = firstLine(a.FailureMessages[0])
			}
			if a.Location != nil {
				f.Line = a.Location.Line
			}
			f.selector = strings.Join(slices.Concat(a.AncestorTitles, []string{a.Title}), " > ")
			r.failed = append(r.failed, f)
		}
		if file.Status == "failed" && !testFailed {
			r.failed = append(r.failed, failure{Package: rel, File: rel, Message: firstLine(file.Message)})
		}
	}
	return r, nil
}

func testDiagram(ctx context.Context, root string, _ testOptions) ([]failure, error) {
	if err := installChromium(ctx, root); err != nil {
		return nil, err
	}
	return nil, run(ctx, filepath.Join(root, "web"), nil, "bun", "run", "diagram:check")
}

// installChromium installs the web app's packages and Chromium, which
// Vitest's browser mode and the Diagram check launch through Playwright.
func installChromium(ctx context.Context, root string) error {
	if err := installWeb(ctx, root); err != nil {
		return err
	}
	return run(ctx, filepath.Join(root, "web"), nil, "bun", "x", "playwright", "install", "chromium")
}

// testBrowser needs Go only to find or build a binary the variables leave
// out, so CI's browser shards run it from the build job's dist/ alone.
func testBrowser(ctx context.Context, root string, o testOptions) ([]failure, error) {
	slipstream, haveSlipstream := testBinary(root, "TEST_SLIPSTREAM_BINARY")
	fakeworld, haveFakeworld := testBinary(root, "TEST_FAKEWORLD_BINARY")
	if !haveSlipstream || !haveFakeworld {
		_, target, bundle, err := goTargetBundle(ctx)
		if err != nil {
			return nil, err
		}
		dist := filepath.Join(root, "dist")
		if !haveSlipstream {
			slipstream = executable(target, dist, "slipstream")
		}
		if !haveFakeworld {
			fakeworld = executable(target, dist, "fakeworld")
			if err := run(ctx, root, bundleEnv(bundle), "go", "build", "-o", fakeworld, "./cmd/fakeworld"); err != nil {
				return nil, err
			}
		}
	}
	if _, err := os.Stat(slipstream); errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("%s: run release build first: the browser tier needs the web app embedded", slipstream)
	} else if err != nil {
		return nil, fmt.Errorf("release binary: %w", err)
	}
	if err := installWeb(ctx, root); err != nil {
		return nil, err
	}
	web := filepath.Join(root, "web")
	env := append(os.Environ(), "TEST_SLIPSTREAM_BINARY="+slipstream, "TEST_FAKEWORLD_BINARY="+fakeworld)
	if err := run(ctx, web, env, "bun", "run", "e2e:install"); err != nil {
		return nil, err
	}
	selected, err := selectBrowserTests(ctx, web, env, o)
	if err != nil {
		return nil, err
	}
	first, err := runPlaywright(ctx, web, env, o, selected)
	if !o.deadline.IsZero() && time.Now().After(o.deadline) {
		return first.failed, err
	}
	failed := rerunFailed(first, func(f []failure) results {
		again, _ := rerunPlaywright(ctx, web, env, o, f)
		return again
	})
	return failed, err
}

// rerunPlaywright runs the failed tests once, each by its project, file and
// titles in a test list.
func rerunPlaywright(ctx context.Context, web string, env []string, o testOptions, failed []failure) (results, error) {
	dir, err := os.MkdirTemp("", "browser-rerun-")
	if err != nil {
		return results{}, fmt.Errorf("write the rerun's test list: %w", err)
	}
	list := filepath.Join(dir, "tests.txt")
	var lines []string
	for _, f := range failed {
		lines = append(lines, f.selector)
	}
	if err := os.WriteFile(list, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		return results{}, fmt.Errorf("write the rerun's test list: %w", err)
	}
	return runPlaywright(ctx, web, env, testOptions{repeat: 1, deadline: o.deadline}, list)
}

// runPlaywright runs the tests in the list selected and records their
// timings when o asks for them.
func runPlaywright(ctx context.Context, web string, env []string, o testOptions, selected string) (results, error) {
	reporter := "list"
	if inCI() {
		reporter = "dot"
	}
	var reports []string
	var err error
	for i, args := range playwrightRuns(o, selected) {
		if !o.deadline.IsZero() {
			args = append(args, "--global-timeout="+strconv.FormatInt(max(time.Until(o.deadline).Milliseconds(), 1), 10))
		}
		report := filepath.Join(filepath.Dir(selected), "report-"+strconv.Itoa(i)+".json")
		reports = append(reports, report)
		runEnv := append(slices.Clone(env), "PLAYWRIGHT_JSON_OUTPUT_FILE="+report)
		args = append([]string{"--reporter=" + reporter + ",json"}, args...)
		if err = run(ctx, web, runEnv, "bun", append([]string{"run", "e2e"}, args...)...); err != nil {
			break
		}
	}
	parsed, readErr := readPlaywrightReports(reports)
	var r results
	for _, p := range parsed {
		r.add(p.results())
	}
	var recordErr error
	if o.record != "" {
		recordErr = recordBrowserTimings(o.record, parsed)
	}
	return r, errors.Join(err, readErr, recordErr)
}

// browserTimings names the browser tier's entry in the timings, beside the
// go and e2e tiers' packages.
const browserTimings = "browser"

func browserTarget() string { return runtime.GOOS + "-" + runtime.GOARCH }

// selectBrowserTests writes this shard's tests to a file for playwright
// --test-list and returns its path. Each shard lists the tests itself, so a
// test the timings have not seen, or a filter, reaches every shard alike.
func selectBrowserTests(ctx context.Context, web string, env []string, o testOptions) (string, error) {
	var filter []string
	if o.filter != "" {
		filter = []string{"--grep=" + o.filter}
	}
	all, err := listBrowserTests(ctx, web, env, filter)
	if err != nil {
		return "", err
	}
	past, err := loadTimings(o.timings, browserTarget())
	if err != nil {
		return "", err
	}
	seconds := past.Packages[browserTimings].Tests
	s := cmp.Or(o.shard, shard{1, 1})
	tests := shardBrowserTests(all, seconds, s)
	timed := 0
	for _, t := range all {
		if seconds[browserTestKey(t)] > 0 {
			timed++
		}
	}
	fmt.Printf("browser tier, shard %s on %s: %d of %d tests, %d of them with timings\n", &s, browserTarget(), len(tests), len(all), timed)
	dir, err := os.MkdirTemp("", "browser-shard-")
	if err != nil {
		return "", fmt.Errorf("write the shard's test list: %w", err)
	}
	selected := filepath.Join(dir, "tests.txt")
	if err := os.WriteFile(selected, []byte(strings.Join(tests, "\n")+"\n"), 0o644); err != nil {
		return "", fmt.Errorf("write the shard's test list: %w", err)
	}
	// Playwright reads a test list by splitting each line on ›, so a title
	// holding one would silently match nothing.
	matched, err := listBrowserTests(ctx, web, env, append(filter, "--test-list="+selected))
	if err != nil {
		return "", err
	}
	if len(matched) != len(tests) {
		return "", fmt.Errorf("the test list of shard %s selects %d of its %d tests", &s, len(matched), len(tests))
	}
	return selected, nil
}

// listBrowserTests is playwright --list's lines, one per test on this
// target in playwright's order, such as "[webkit-phone] › work.spec.ts:14:1 ›
// Refresh reads the source".
func listBrowserTests(ctx context.Context, web string, env, args []string) ([]string, error) {
	cmd := exec.CommandContext(ctx, "bun", slices.Concat([]string{"run", "e2e", "--list", "--pass-with-no-tests"}, args)...)
	cmd.Dir = web
	cmd.Env = env
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("list the browser tests: %w", err)
	}
	var tests []string
	for line := range strings.Lines(string(out)) {
		if line = strings.TrimSpace(line); strings.HasPrefix(line, "[") {
			tests = append(tests, line)
		}
	}
	return tests, nil
}

var browserTestLocation = regexp.MustCompile(`:\d+:\d+ › `)

// browserTestKey drops a listed test's line and column, which move as its
// file changes, to name it in the timings.
func browserTestKey(listed string) string {
	if loc := browserTestLocation.FindStringIndex(listed); loc != nil {
		return listed[:loc[0]] + " › " + listed[loc[1]:]
	}
	return listed
}

// shardBrowserTests is the s-th of the runs that cut tests into shards of
// about equal seconds, keeping their order, so a shard holds one or two of
// the target's projects and starts few workers. A test the timings have not
// seen weighs as the mean of those they have, or 1 without timings, which cuts
// the tests evenly. A test falls to the shard its midpoint lands in, so each
// test lands in exactly one.
func shardBrowserTests(tests []string, seconds map[string]float64, s shard) []string {
	var known float64
	var seen int
	for _, t := range tests {
		if w := seconds[browserTestKey(t)]; w > 0 {
			known += w
			seen++
		}
	}
	mean := 1.0
	if known > 0 {
		mean = known / float64(seen)
	}
	weight := func(t string) float64 {
		if w := seconds[browserTestKey(t)]; w > 0 {
			return w
		}
		return mean
	}
	var total float64
	for _, t := range tests {
		total += weight(t)
	}
	var kept []string
	var before float64
	for _, t := range tests {
		w := weight(t)
		if min(int((before+w/2)*float64(s.count)/total), s.count-1) == s.index-1 {
			kept = append(kept, t)
		}
		before += w
	}
	return kept
}

// playwrightReport is the part of playwright's JSON report the timings read.
type playwrightReport struct {
	Stats struct {
		// Duration is the run's milliseconds.
		Duration float64 `json:"duration"`
	} `json:"stats"`
	Suites []playwrightSuite `json:"suites"`
}

type playwrightSuite struct {
	Title  string            `json:"title"`
	Suites []playwrightSuite `json:"suites"`
	Specs  []struct {
		Title string           `json:"title"`
		File  string           `json:"file"`
		Line  int              `json:"line"`
		Tests []playwrightTest `json:"tests"`
	} `json:"specs"`
}

type playwrightTest struct {
	ProjectName string `json:"projectName"`
	// Status is expected, unexpected, flaky or skipped.
	Status  string `json:"status"`
	Results []struct {
		Duration float64 `json:"duration"`
		Status   string  `json:"status"`
		Errors   []struct {
			Message string `json:"message"`
		} `json:"errors"`
	} `json:"results"`
}

// e2eDir is where Playwright's config sits, which the report's files are
// relative to.
const e2eDir = "web/e2e"

// eachTest calls fn with each test, its spec's file and line, and the
// titles under its project: describes holds the suites between the file
// and s.
func (s playwrightSuite) eachTest(describes []string, fn func(test playwrightTest, file string, line int, titles []string)) {
	for _, spec := range s.Specs {
		for _, test := range spec.Tests {
			fn(test, spec.File, spec.Line, slices.Concat(describes, []string{spec.Title}))
		}
	}
	for _, sub := range s.Suites {
		sub.eachTest(append(slices.Clip(describes), sub.Title), fn)
	}
}

// addMilliseconds adds each test's milliseconds, by the name browserTestKey
// gives it.
func (s playwrightSuite) addMilliseconds(ms map[string]float64) {
	s.eachTest(nil, func(test playwrightTest, file string, _ int, titles []string) {
		path := slices.Concat([]string{"[" + test.ProjectName + "]", file}, titles)
		for _, r := range test.Results {
			ms[strings.Join(path, " › ")] += r.Duration
		}
	})
}

// results names each failed test by its project and titles, and selects it
// for a rerun by the test list line that names it.
func (r playwrightReport) results() results {
	var out results
	for _, s := range r.Suites {
		s.eachTest(nil, func(test playwrightTest, file string, line int, titles []string) {
			project := "[" + test.ProjectName + "]"
			f := failure{
				Package:  e2eDir + "/" + file,
				File:     e2eDir + "/" + file,
				Line:     line,
				Test:     strings.Join(slices.Concat([]string{project}, titles), " › "),
				selector: strings.Join(slices.Concat([]string{project, file}, titles), " › "),
			}
			switch test.Status {
			case "expected", "flaky":
				out.pass(f.key())
			case "unexpected":
				if n := len(test.Results); n > 0 {
					last := test.Results[n-1]
					f.Message = fmt.Sprintf("The test ended with the result %s and no error message.", last.Status)
					if len(last.Errors) > 0 {
						f.Message = firstLine(last.Errors[0].Message)
					}
				}
				out.failed = append(out.failed, f)
			}
		})
	}
	return out
}

// seconds shares the run's wall time among its tests by their durations, so
// the seconds of a shard's tests add up to the time it takes. A duration alone
// falls short of that: it leaves out a test's world starting and stopping,
// most of the Library's scale measure, and two workers spend theirs at once.
func (r playwrightReport) seconds() map[string]float64 {
	ms := map[string]float64{}
	for _, s := range r.Suites {
		s.addMilliseconds(ms)
	}
	var sum float64
	for _, d := range ms {
		sum += d
	}
	if sum == 0 {
		return nil
	}
	seconds := make(map[string]float64, len(ms))
	for name, d := range ms {
		seconds[name] = d / sum * r.Stats.Duration / 1000
	}
	return seconds
}

// readPlaywrightReports reads each report a run wrote. A run that failed
// before another started leaves that one's report missing.
func readPlaywrightReports(paths []string) ([]playwrightReport, error) {
	var reports []playwrightReport
	for _, f := range paths {
		b, err := os.ReadFile(f)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("read the browser report: %w", err)
		}
		var r playwrightReport
		if err := json.Unmarshal(b, &r); err != nil {
			return nil, fmt.Errorf("%s: %w", f, err)
		}
		reports = append(reports, r)
	}
	return reports, nil
}

func recordBrowserTimings(path string, reports []playwrightReport) error {
	recorded := packageTimings{Tests: map[string]float64{}}
	for _, r := range reports {
		for name, s := range r.seconds() {
			recorded.Tests[name] += s
		}
		recorded.Seconds += r.Stats.Duration / 1000
	}
	t := timings{Target: browserTarget(), CPUs: runtime.NumCPU(), Packages: map[string]packageTimings{browserTimings: recorded}}
	return t.write(path)
}

// alone tags a browser test that loads the whole runner, as the Library's
// 20,000-Work measure does. A test beside it on a Windows runner stalls past
// its timeout, and the measure would count that test's work as its own.
const alone = "@alone"

// playwrightRuns runs the shard's tests in the file selected: those tagged
// alone after the rest, on one worker. A filter can match tests of one run
// only, so neither fails for finding none, as go test -run doesn't.
func playwrightRuns(o testOptions, selected string) [][]string {
	shared := []string{"--pass-with-no-tests", "--test-list=" + selected}
	if o.repeat > 1 {
		shared = append(shared, "--repeat-each="+strconv.Itoa(o.repeat))
	}
	rest := append([]string{"--grep-invert=" + alone}, shared...)
	tagged := alone
	if o.filter != "" {
		rest = append(rest, "--grep="+o.filter)
		tagged = "(?=.*" + alone + ")(?=.*(?:" + o.filter + "))"
	}
	return [][]string{rest, append([]string{"--workers=1", "--grep=" + tagged}, shared...)}
}

// testBinary is the variable's path, read from the repo root as testbin
// reads it, and whether the variable is set.
func testBinary(root, variable string) (string, bool) {
	p, ok := os.LookupEnv(variable)
	if !ok {
		return "", false
	}
	if !filepath.IsAbs(p) {
		p = filepath.Join(root, p)
	}
	return p, true
}

func executable(target native.Target, dir, name string) string {
	if target.GOOS == "windows" {
		name += ".exe"
	}
	return filepath.Join(dir, name)
}
