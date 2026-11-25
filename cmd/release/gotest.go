package main

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"maps"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
)

const modulePath = "github.com/jatassi/slipstream"

// testbinPath is the package that hands a test the binaries under test, so
// a package whose tests depend on it belongs to the e2e tier.
const testbinPath = modulePath + "/fakes/testbin"

// testGoPackages runs the go or e2e tier's packages, or this shard's part
// of them, through go test -json, printing what go test prints without -v.
func testGoPackages(ctx context.Context, root string, o testOptions, tier string) ([]failure, error) {
	_, target, bundle, err := goTargetBundle(ctx)
	if err != nil {
		return nil, err
	}
	env := bundleEnv(bundle)
	// A simulation package, worksapi's 20,000-Work and 32,767-Item cases,
	// and the e2e package when no timings split it across shards run for
	// most of twenty minutes under -race on a busy runner; the e2e job's
	// own limit is thirty-five.
	base := []string{"-race", "-shuffle=on", "-timeout", "25m"}
	if len(o.goArgs) > 0 {
		return nil, goTest(ctx, root, env, append(base, o.goArgs...), os.Stdout)
	}
	flags := slices.Clone(base)
	if o.filter != "" {
		flags = append(flags, "-run", o.filter)
	}
	if o.repeat > 1 {
		flags = append(flags, "-count", strconv.Itoa(o.repeat))
	}
	pkgs, err := listGoPackages(ctx, root, env)
	if err != nil {
		return nil, err
	}
	pkgs = slices.DeleteFunc(pkgs, func(p goPackage) bool { return p.e2e != (tier == "e2e") })
	past, err := loadTimings(o.timings, target.Name)
	if err != nil {
		return nil, err
	}
	s := cmp.Or(o.shard, shard{1, 1})
	runs := partition(pkgs, past, s.count, o.filter == "")[s.index-1]
	if tier == "e2e" {
		// An e2e part runs beside the whole packages, and its budget tests
		// miss under the two heaviest packages started first, so the whole
		// packages start by name.
		for _, r := range runs {
			slices.Sort(r.packages)
		}
	}
	fmt.Printf("%s tier, shard %s on %s:\n", tier, &s, target.Name)
	for _, r := range runs {
		fmt.Print(r.listing())
	}
	recorded, first, err := runShard(ctx, root, env, flags, runs)
	if o.record != "" {
		recorded.Target = target.Name
		if err := recorded.write(o.record); err != nil {
			return nil, err
		}
	}
	failed := rerunFailed(first, func(f []failure) results {
		_, again, _ := runShard(ctx, root, env, append(slices.Clone(base), "-count=1"), rerunRuns(f))
		return again
	})
	return failed, err
}

// rerunRuns is a go test per package of the failed tests' top-level tests,
// which run their subtests again with them.
func rerunRuns(failed []failure) []goRun {
	tests := map[string][]string{}
	for _, f := range failed {
		if !slices.Contains(tests[f.Package], f.selector) {
			tests[f.Package] = append(tests[f.Package], f.selector)
		}
	}
	var runs []goRun
	for _, pkg := range slices.Sorted(maps.Keys(tests)) {
		runs = append(runs, goRun{packages: []string{pkg}, run: testsPattern(tests[pkg])})
	}
	return runs
}

// runShard runs a shard's go tests at once, as go test runs packages, so
// a package's part runs beside the whole packages rather than after them.
func runShard(ctx context.Context, root string, env, flags []string, runs []goRun) (timings, results, error) {
	out := &lockedWriter{w: os.Stdout}
	errs := make([]error, len(runs))
	logs := make([]*goTestLog, len(runs))
	var wg sync.WaitGroup
	for i, r := range runs {
		logs[i] = newGoTestLog(out)
		wg.Go(func() {
			errs[i] = errors.Join(goTest(ctx, root, env, slices.Concat([]string{"-json"}, flags, r.args()), logs[i]), logs[i].flush())
		})
	}
	wg.Wait()
	recorded := timings{CPUs: runtime.NumCPU(), Packages: map[string]packageTimings{}}
	var r results
	for _, l := range logs {
		recorded.add(l.timings)
		r.add(l.results())
	}
	return recorded, r, errors.Join(errs...)
}

type lockedWriter struct {
	mu sync.Mutex
	w  io.Writer
}

func (w *lockedWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	n, err := w.w.Write(p)
	if err != nil {
		return n, fmt.Errorf("print the go test output: %w", err)
	}
	return n, nil
}

func goTest(ctx context.Context, root string, env, args []string, stdout io.Writer) error {
	gotest := exec.CommandContext(ctx, "go", append([]string{"test"}, args...)...)
	gotest.Dir = root
	gotest.Stdout, gotest.Stderr = stdout, os.Stderr
	gotest.Env = env
	if err := gotest.Run(); err != nil {
		return fmt.Errorf("go test: %w", err)
	}
	return nil
}

type goPackage struct {
	path string
	// tests says the package has test files.
	tests bool
	// testBytes is the size of the package's test files.
	testBytes int64
	// testNames is the top-level tests its test files declare.
	testNames []string
	// e2e says the package's tests depend on fakes/testbin.
	e2e bool
}

// listGoPackages lists every package of the module, with the test
// packages go list -test adds folded into the package they test.
func listGoPackages(ctx context.Context, root string, env []string) ([]goPackage, error) {
	const format = "{{.ImportPath}}\t{{.Dir}}\t{{join .TestGoFiles \",\"}}\t{{join .XTestGoFiles \",\"}}\t" +
		`{{range .Deps}}{{if eq . "` + testbinPath + `"}}testbin{{end}}{{end}}`
	cmd := exec.CommandContext(ctx, "go", "list", "-test", "-f", format, "./...")
	cmd.Dir = root
	cmd.Env = env
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("go list: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	var pkgs []goPackage
	testbin := map[string]bool{}
	for line := range strings.Lines(string(out)) {
		fields := strings.Split(strings.TrimSuffix(line, "\n"), "\t")
		if len(fields) != 5 {
			return nil, fmt.Errorf("go list: unexpected line %q", line)
		}
		path := fields[0]
		switch {
		case strings.Contains(path, " "):
			// A package recompiled for a test, such as "p [p.test]".
		case strings.HasSuffix(path, ".test"):
			testbin[strings.TrimSuffix(path, ".test")] = fields[4] != ""
		default:
			names := strings.FieldsFunc(fields[2]+","+fields[3], func(r rune) bool { return r == ',' })
			pkg := goPackage{path: path, tests: len(names) > 0}
			for _, n := range names {
				file := filepath.Join(fields[1], n)
				info, err := os.Stat(file)
				if err != nil {
					return nil, fmt.Errorf("go list: %w", err)
				}
				pkg.testBytes += info.Size()
				tests, err := topLevelTests(file)
				if err != nil {
					return nil, err
				}
				pkg.testNames = append(pkg.testNames, tests...)
			}
			pkgs = append(pkgs, pkg)
		}
	}
	for i := range pkgs {
		pkgs[i].e2e = testbin[pkgs[i].path]
	}
	return pkgs, nil
}

// topLevelTests is the Test functions a test file declares, the names
// go test -run matches at the top level.
func topLevelTests(file string) ([]string, error) {
	f, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.SkipObjectResolution)
	if err != nil {
		return nil, fmt.Errorf("go list: %w", err)
	}
	var tests []string
	for _, d := range f.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if ok && fn.Recv == nil && strings.HasPrefix(fn.Name.Name, "Test") && fn.Name.Name != "TestMain" {
			tests = append(tests, fn.Name.Name)
		}
	}
	return tests, nil
}

// goRun is one go test of a shard: whole packages, or one package's
// top-level tests that run matches or skip leaves out.
type goRun struct {
	packages  []string
	run, skip string
}

func (r goRun) args() []string {
	var args []string
	if r.run != "" {
		args = append(args, "-run", r.run)
	}
	if r.skip != "" {
		args = append(args, "-skip", r.skip)
	}
	return append(args, r.packages...)
}

// listing is the run as the shard's header lists it: a whole package to
// a line, or a part with its pattern.
func (r goRun) listing() string {
	if r.run == "" && r.skip == "" {
		return "  " + strings.Join(r.packages, "\n  ") + "\n"
	}
	return "  " + strings.Join(r.args(), " ") + "\n"
}

// weighed is a package, or a part of one, as partition packs it.
type weighed struct {
	pkg     string
	seconds float64
	// tests is the part's top-level tests, nil for a whole package.
	tests []string
	// rest marks the part that runs every test the other parts leave, so
	// a test the timings have not seen yet still runs once.
	rest bool
}

// partition packs pkgs into shards, longest first by the seconds the past
// timings give each, so the shards finish close together. A package the
// timings have not seen weighs as the mean of those they have; when they
// have seen none, every package weighs the bytes of its test files. A
// package with no test files weighs nothing. A shard runs its packages on
// every CPU at once, so it can finish no sooner than its longest
// package: with splitTests, a package longer than a shard's share spread
// over the timed runner's CPUs splits by its top-level tests, into a part
// per such span, on as many shards. The timings may come from other targets,
// as loadTimings says, which works because only their ratios matter.
func partition(pkgs []goPackage, past timings, shards int, splitTests bool) [][]goRun {
	seconds, total := weigh(pkgs, past)
	span := total / float64(shards*max(past.CPUs, 1))
	var items []weighed
	for _, p := range pkgs {
		w := seconds[p.path]
		tests := past.Packages[p.path].Tests
		if !splitTests || w <= span || len(tests) < 2 || shards < 2 {
			items = append(items, weighed{pkg: p.path, seconds: w})
			continue
		}
		items = append(items, splitPackage(p.path, w, tests, p.testNames, min(shards, len(tests), int(math.Ceil(w/span))))...)
	}
	return packShards(items, shards)
}

// weigh gives each package its seconds, or its test-file bytes when the
// timings have none of the packages, and sums them. Each shard
// partitions on its own, so every sum runs in package order: float
// addition in map order could tip a comparison one way in one shard and
// the other way in the next.
func weigh(pkgs []goPackage, past timings) (seconds map[string]float64, total float64) {
	seconds = map[string]float64{}
	var known float64
	var seen int
	for _, p := range pkgs {
		if t, ok := past.Packages[p.path]; ok && p.tests {
			known += t.Seconds
			seen++
		}
	}
	var mean float64
	if known > 0 {
		mean = known / float64(seen)
	}
	for _, p := range pkgs {
		t, ok := past.Packages[p.path]
		switch {
		case !p.tests:
			seconds[p.path] = 0
		case known == 0:
			seconds[p.path] = float64(p.testBytes)
		case ok:
			seconds[p.path] = t.Seconds
		default:
			seconds[p.path] = mean
		}
		total += seconds[p.path]
	}
	return seconds, total
}

// packShards gives each item, longest first, to the shard with the least so
// far, keeping a package's parts on different shards. A shard lists its
// whole packages longest first too, since go test starts them in that
// order: listed by name, a long package could start last and end the shard
// long after the rest.
func packShards(items []weighed, shards int) [][]goRun {
	slices.SortStableFunc(items, func(a, b weighed) int {
		return cmp.Or(cmp.Compare(b.seconds, a.seconds), cmp.Compare(a.pkg, b.pkg))
	})
	load := make([]float64, shards)
	whole := make([][]string, shards)
	parts := make([][]weighed, shards)
	for _, it := range items {
		best := -1
		for i := range shards {
			taken := it.tests != nil && slices.ContainsFunc(parts[i], func(w weighed) bool { return w.pkg == it.pkg })
			if !taken && (best < 0 || load[i] < load[best]) {
				best = i
			}
		}
		load[best] += it.seconds
		if it.tests == nil {
			whole[best] = append(whole[best], it.pkg)
		} else {
			parts[best] = append(parts[best], it)
		}
	}
	out := make([][]goRun, shards)
	for i := range shards {
		if len(whole[i]) > 0 {
			out[i] = append(out[i], goRun{packages: whole[i]})
		}
		for _, part := range parts[i] {
			out[i] = append(out[i], part.run(items))
		}
	}
	return out
}

// splitPackage packs a package's top-level tests into n parts, longest
// first, scaled so the timed tests weigh the package's seconds. A declared
// test the timings have not seen weighs as their mean, so a batch of new
// tests spreads over the parts instead of piling into the rest part.
func splitPackage(pkg string, seconds float64, timed map[string]float64, declared []string, n int) []weighed {
	var sum float64
	for _, name := range slices.Sorted(maps.Keys(timed)) {
		sum += timed[name]
	}
	tests := maps.Clone(timed)
	for _, name := range declared {
		if _, ok := tests[name]; !ok {
			tests[name] = sum / float64(len(timed))
		}
	}
	names := slices.SortedFunc(maps.Keys(tests), func(a, b string) int { return cmp.Or(cmp.Compare(tests[b], tests[a]), cmp.Compare(a, b)) })
	scale := 1.0
	if sum > 0 {
		scale = seconds / sum
	}
	parts := make([]weighed, n)
	for i := range parts {
		parts[i] = weighed{pkg: pkg, tests: []string{}, rest: i == n-1}
	}
	for _, name := range names {
		lightest := 0
		for i := range parts {
			if parts[i].seconds < parts[lightest].seconds {
				lightest = i
			}
		}
		parts[lightest].tests = append(parts[lightest].tests, name)
		parts[lightest].seconds += tests[name] * scale
	}
	return parts
}

// run is the go test of a part: the tests it lists, or for the rest part
// every test the package's other parts do not list.
func (w weighed) run(items []weighed) goRun {
	r := goRun{packages: []string{w.pkg}}
	if !w.rest {
		r.run = testsPattern(w.tests)
		return r
	}
	var others []string
	for _, it := range items {
		if it.pkg == w.pkg && !it.rest {
			others = append(others, it.tests...)
		}
	}
	r.skip = testsPattern(others)
	return r
}

func testsPattern(tests []string) string {
	quoted := make([]string, len(tests))
	for i, t := range tests {
		quoted[i] = regexp.QuoteMeta(t)
	}
	slices.Sort(quoted)
	return "^(" + strings.Join(quoted, "|") + ")$"
}

// timings are the seconds each package's tests took on one target, as
// go test -json reports them.
type timings struct {
	Target string `json:"target"`
	// CPUs is how many CPUs the runner that took the timings had.
	CPUs     int                       `json:"cpus"`
	Packages map[string]packageTimings `json:"packages"`
}

type packageTimings struct {
	Seconds float64 `json:"seconds"`
	// Tests holds the seconds of each top-level test.
	Tests map[string]float64 `json:"tests,omitempty"`
}

// loadTimings merges the target's timings files in dir; a missing dir
// holds none. A package split across shards sums its parts. A package
// the target has no timings for takes them from the first other target,
// in name order, that has it, and a target that recorded no CPUs takes
// the CPUs of the first target that filled a package. Partition uses only
// how the seconds compare, so another runner's need no scaling. Within a
// package the target has, a test it never recorded takes another target's
// seconds as fillTests scales them.
func loadTimings(dir, target string) (timings, error) {
	byTarget := map[string]*timings{target: {Target: target, Packages: map[string]packageTimings{}}}
	if dir == "" {
		return *byTarget[target], nil
	}
	files, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return timings{}, fmt.Errorf("list the timings: %w", err)
	}
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			return timings{}, fmt.Errorf("read the timings: %w", err)
		}
		var t timings
		if err := json.Unmarshal(b, &t); err != nil {
			return timings{}, fmt.Errorf("%s: %w", f, err)
		}
		if byTarget[t.Target] == nil {
			byTarget[t.Target] = &timings{Target: t.Target, Packages: map[string]packageTimings{}}
		}
		byTarget[t.Target].add(t)
	}
	merged := byTarget[target]
	fillUnseenTests(byTarget, target)
	for _, name := range slices.Sorted(maps.Keys(byTarget)) {
		other := byTarget[name]
		supplied := false
		for pkg, pt := range other.Packages {
			if _, ok := merged.Packages[pkg]; !ok {
				merged.Packages[pkg] = pt
				supplied = true
			}
		}
		if supplied && merged.CPUs == 0 {
			merged.CPUs = other.CPUs
		}
	}
	return *merged, nil
}

// fillUnseenTests fills each package the target recorded with the tests
// it never ran, from every other target in name order.
func fillUnseenTests(byTarget map[string]*timings, target string) {
	for pkg, own := range byTarget[target].Packages {
		recorded := maps.Clone(own.Tests)
		for _, name := range slices.Sorted(maps.Keys(byTarget)) {
			if name != target {
				fillTests(recorded, own.Tests, byTarget[name].Packages[pkg].Tests)
			}
		}
	}
}

// fillTests gives each test of theirs that tests lacks theirs's seconds,
// scaled by how the two targets compare on the tests both recorded. A test
// another target ran weighs closer to its own cost than the mean of the
// rest, which on darwin's browser tier is the mean of the one shard that
// finished. Without a test in common the two scales can't be compared, so
// tests is left as it is.
func fillTests(recorded, tests, theirs map[string]float64) {
	var mine, other float64
	for name, s := range recorded {
		if t, ok := theirs[name]; ok {
			mine += s
			other += t
		}
	}
	if mine == 0 || other == 0 {
		return
	}
	for name, t := range theirs {
		if _, ok := tests[name]; !ok {
			tests[name] = t * mine / other
		}
	}
}

func (t *timings) add(o timings) {
	t.CPUs = max(t.CPUs, o.CPUs)
	for pkg, pt := range o.Packages {
		m := t.Packages[pkg]
		m.Seconds += pt.Seconds
		for name, s := range pt.Tests {
			if m.Tests == nil {
				m.Tests = map[string]float64{}
			}
			m.Tests[name] += s
		}
		t.Packages[pkg] = m
	}
}

func (t *timings) write(path string) error {
	b, err := json.MarshalIndent(t, "", "\t")
	if err != nil {
		return fmt.Errorf("encode the timings: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("write the timings: %w", err)
	}
	if err := os.WriteFile(path, append(b, '\n'), 0o644); err != nil {
		return fmt.Errorf("write the timings: %w", err)
	}
	return nil
}

// testEvent is a line of go test -json, as cmd/test2json documents it.
type testEvent struct {
	Action  string
	Package string
	Test    string
	Elapsed float64
	Output  string
}

// goTestLog reads go test -json and writes what go test writes without
// -v: each package's result, and the output of each test that fails. It
// records the timings as it goes.
type goTestLog struct {
	w       io.Writer
	pending []byte
	// held holds each running test's output until it passes or fails.
	held map[string][]string
	// subtests holds the seconds each running test's finished subtests
	// took, as go test reports a test's subtests before the test itself.
	subtests map[string]float64
	timings  timings
	// output holds each running package's output outside its tests.
	output map[string][]string
	run    results
}

func newGoTestLog(w io.Writer) *goTestLog {
	return &goTestLog{w: w, held: map[string][]string{}, subtests: map[string]float64{}, timings: timings{Packages: map[string]packageTimings{}}, output: map[string][]string{}}
}

func (l *goTestLog) Write(p []byte) (int, error) {
	l.pending = append(l.pending, p...)
	for {
		i := bytes.IndexByte(l.pending, '\n')
		if i < 0 {
			return len(p), nil
		}
		line := l.pending[:i+1]
		l.pending = l.pending[i+1:]
		if err := l.line(line); err != nil {
			return len(p), err
		}
	}
}

func (l *goTestLog) flush() error {
	if len(l.pending) == 0 {
		return nil
	}
	line := l.pending
	l.pending = nil
	return l.line(line)
}

func (l *goTestLog) line(line []byte) error {
	var e testEvent
	if err := json.Unmarshal(line, &e); err != nil || e.Action == "" {
		return l.print(string(line))
	}
	key := e.Package + "\x00" + e.Test
	switch {
	case e.Action == "build-output":
		return l.print(e.Output)
	case e.Action == "output" && e.Test == "":
		if e.Output == "PASS\n" {
			return nil
		}
		l.output[e.Package] = append(l.output[e.Package], e.Output)
		return l.print(e.Output)
	case e.Action == "output" && !strings.HasPrefix(e.Output, "=== "):
		l.held[key] = append(l.held[key], e.Output)
	case e.Test != "" && (e.Action == "pass" || e.Action == "skip"):
		l.record(e)
		if e.Action == "pass" {
			l.run.pass(key)
		}
		delete(l.held, key)
	case e.Test != "" && e.Action == "fail":
		l.record(e)
		l.run.failed = append(l.run.failed, goFailure(e.Package, e.Test, l.held[key], true))
		return l.release(func(k string) bool { return k == key })
	case e.Test == "" && slices.Contains([]string{"pass", "fail", "skip"}, e.Action):
		l.record(e)
		prefix := e.Package + "\x00"
		output := l.output[e.Package]
		delete(l.output, e.Package)
		if e.Action == "fail" {
			l.packageFailed(e.Package, output)
			return l.release(func(k string) bool { return strings.HasPrefix(k, prefix) })
		}
		for k := range l.held {
			if strings.HasPrefix(k, prefix) {
				delete(l.held, k)
			}
		}
	}
	return nil
}

// packageFailed records the tests still running when their package
// failed, as a timeout or a crash leaves them, or the package itself when
// none of its tests failed. Neither runs again: a test that hung would hang
// again past the job's limit.
func (l *goTestLog) packageFailed(pkg string, output []string) {
	prefix := pkg + "\x00"
	var running []string
	for k := range l.held {
		if strings.HasPrefix(k, prefix) {
			running = append(running, k)
		}
	}
	slices.Sort(running)
	for _, k := range running {
		l.run.failed = append(l.run.failed, goFailure(pkg, strings.TrimPrefix(k, prefix), l.held[k], false))
	}
	if !slices.ContainsFunc(l.run.failed, func(f failure) bool { return f.Package == pkg }) {
		l.run.failed = append(l.run.failed, failure{Package: pkg, Message: packageMessage(output)})
	}
}

// results is the run's failures, each test's failed subtests in its place,
// and the tests that passed.
func (l *goTestLog) results() results {
	r := l.run
	r.failed = slices.DeleteFunc(slices.Clone(r.failed), func(f failure) bool {
		return f.Test != "" && slices.ContainsFunc(r.failed, func(o failure) bool {
			return o.Package == f.Package && strings.HasPrefix(o.Test, f.Test+"/")
		})
	})
	return r
}

// goFailureLine is a t.Error or t.Fatal line: the test file, the line and
// the message.
var goFailureLine = regexp.MustCompile(`^\s+([\w.-]+\.go):(\d+): (.*)`)

// goFailure is a failed test from its output: a panic's first line, or else
// the last line the test's t methods wrote from a _test.go file, which is a
// t.Fatal's or the last t.Error's unless a log follows it, or else the last
// line with a file and line, or else its first line. A helper's own lines,
// such as the Node logs a sim dumps on failure, follow the t.Fatal that
// failed the test, so they never outrank it.
func goFailure(pkg, test string, output []string, rerun bool) failure {
	f := failure{Package: pkg, Test: test}
	if rerun {
		f.selector, _, _ = strings.Cut(test, "/")
	}
	fromTest := false
	for _, line := range output {
		if t := strings.TrimSpace(line); strings.HasPrefix(t, "panic: ") {
			f.Message = t
			return f
		}
		m := goFailureLine.FindStringSubmatch(line)
		isTest := m != nil && strings.HasSuffix(m[1], "_test.go")
		if m == nil || (fromTest && !isTest) {
			continue
		}
		f.File = strings.TrimPrefix(strings.TrimPrefix(pkg, modulePath)+"/"+m[1], "/")
		f.Line, _ = strconv.Atoi(m[2])
		f.Message = m[3]
		fromTest = fromTest || isTest
	}
	if f.Message == "" {
		for _, line := range output {
			if t := strings.TrimSpace(line); t != "" && !strings.HasPrefix(t, "--- ") {
				f.Message = t
				break
			}
		}
	}
	return f
}

// packageMessage is the first line of a failed package's own output that
// says more than that it failed.
func packageMessage(output []string) string {
	for _, line := range output {
		t := strings.TrimSpace(line)
		if t != "" && t != "FAIL" && !strings.HasPrefix(t, "exit status ") {
			return strings.Join(strings.Fields(t), " ")
		}
	}
	return ""
}

// record adds a package's seconds, or a top-level test's. A test weighs
// at least what its subtests took together: go test times a test whose
// subtests call t.Parallel only until it starts them, so its own seconds
// leave them out.
func (l *goTestLog) record(e testEvent) {
	p := l.timings.Packages[e.Package]
	if e.Test == "" {
		p.Seconds += e.Elapsed
		l.timings.Packages[e.Package] = p
		return
	}
	key := e.Package + "\x00" + e.Test
	seconds := max(e.Elapsed, l.subtests[key])
	delete(l.subtests, key)
	if i := strings.LastIndex(e.Test, "/"); i >= 0 {
		l.subtests[e.Package+"\x00"+e.Test[:i]] += seconds
		return
	}
	if p.Tests == nil {
		p.Tests = map[string]float64{}
	}
	p.Tests[e.Test] += seconds
	l.timings.Packages[e.Package] = p
}

// release prints and drops the held output of the tests match picks, in
// name order, so a test's output comes before its subtests'.
func (l *goTestLog) release(match func(key string) bool) error {
	var keys []string
	for k := range l.held {
		if match(k) {
			keys = append(keys, k)
		}
	}
	slices.Sort(keys)
	for _, k := range keys {
		if err := l.print(strings.Join(l.held[k], "")); err != nil {
			return err
		}
		delete(l.held, k)
	}
	return nil
}

func (l *goTestLog) print(s string) error {
	if _, err := io.WriteString(l.w, s); err != nil {
		return fmt.Errorf("print the go test output: %w", err)
	}
	return nil
}
