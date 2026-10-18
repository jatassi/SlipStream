package main

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

// testbinPath is the package that hands a test the binaries under test, so
// a package whose tests depend on it belongs to the e2e tier.
const testbinPath = "github.com/jatassi/slipstream/fakes/testbin"

// testGoPackages runs the go or e2e tier's packages, or this shard's part
// of them, through go test -json, printing what go test prints without -v.
func testGoPackages(ctx context.Context, root string, o testOptions, tier string) error {
	_, target, bundle, err := goTargetBundle(ctx)
	if err != nil {
		return err
	}
	env := bundleEnv(bundle)
	flags := []string{"-race", "-shuffle=on"}
	if len(o.goArgs) > 0 {
		return goTest(ctx, root, env, append(flags, o.goArgs...), os.Stdout)
	}
	if o.filter != "" {
		flags = append(flags, "-run", o.filter)
	}
	if o.repeat > 1 {
		flags = append(flags, "-count", strconv.Itoa(o.repeat))
	}
	pkgs, err := listGoPackages(ctx, root, env)
	if err != nil {
		return err
	}
	pkgs = slices.DeleteFunc(pkgs, func(p goPackage) bool { return p.e2e != (tier == "e2e") })
	past, err := loadTimings(o.timings, target.Name)
	if err != nil {
		return err
	}
	s := cmp.Or(o.shard, shard{1, 1})
	runs := partition(pkgs, past, s.count, o.filter == "")[s.index-1]
	fmt.Printf("%s tier, shard %s on %s:\n", tier, &s, target.Name)
	for _, r := range runs {
		fmt.Print(r.listing())
	}
	recorded, err := runShard(ctx, root, env, flags, runs)
	if o.record != "" {
		recorded.Target = target.Name
		if err := recorded.write(o.record); err != nil {
			return err
		}
	}
	return err
}

// runShard runs a shard's go tests at once, as go test runs packages, so
// a package's part runs beside the whole packages rather than after them.
func runShard(ctx context.Context, root string, env, flags []string, runs []goRun) (timings, error) {
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
	for _, l := range logs {
		recorded.add(l.timings)
	}
	return recorded, errors.Join(errs...)
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
	// e2e says the package's tests depend on fakes/testbin.
	e2e bool
}

// listGoPackages lists every package of the module, with the test
// packages go list -test adds folded into the package they test.
func listGoPackages(ctx context.Context, root string, env []string) ([]goPackage, error) {
	const format = "{{.ImportPath}}\t{{len .TestGoFiles}}\t{{len .XTestGoFiles}}\t" +
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
		if len(fields) != 4 {
			return nil, fmt.Errorf("go list: unexpected line %q", line)
		}
		path := fields[0]
		switch {
		case strings.Contains(path, " "):
			// A package recompiled for a test, such as "p [p.test]".
		case strings.HasSuffix(path, ".test"):
			testbin[strings.TrimSuffix(path, ".test")] = fields[3] != ""
		default:
			pkgs = append(pkgs, goPackage{path: path, tests: fields[1] != "0" || fields[2] != "0"})
		}
	}
	for i := range pkgs {
		pkgs[i].e2e = testbin[pkgs[i].path]
	}
	return pkgs, nil
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
// timings have not seen weighs as the mean of those they have, or 1
// without timings, which splits the packages evenly; a package with no
// test files weighs nothing. A shard runs its packages on every CPU at
// once, so it can finish no sooner than its longest package: with
// splitTests, a package longer than a shard's share spread over the
// timed runner's CPUs splits by its top-level tests, into a part per
// such span, on as many shards.
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
		items = append(items, splitPackage(p.path, w, tests, min(shards, len(tests), int(math.Ceil(w/span))))...)
	}
	return packShards(items, shards)
}

// weigh gives each package its seconds and sums them. Each shard
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
	mean := 1.0
	if known > 0 {
		mean = known / float64(seen)
	}
	for _, p := range pkgs {
		t, ok := past.Packages[p.path]
		switch {
		case !p.tests:
			seconds[p.path] = 0
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
// far, keeping a package's parts on different shards.
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
			slices.Sort(whole[i])
			out[i] = append(out[i], goRun{packages: whole[i]})
		}
		for _, part := range parts[i] {
			out[i] = append(out[i], part.run(items))
		}
	}
	return out
}

// splitPackage packs a package's top-level tests into n parts, longest
// first, scaled so the parts weigh the package's seconds.
func splitPackage(pkg string, seconds float64, tests map[string]float64, n int) []weighed {
	names := slices.SortedFunc(maps.Keys(tests), func(a, b string) int { return cmp.Or(cmp.Compare(tests[b], tests[a]), cmp.Compare(a, b)) })
	var sum float64
	for _, name := range names {
		sum += tests[name]
	}
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
// holds none. A package split across shards sums its parts.
func loadTimings(dir, target string) (timings, error) {
	merged := timings{Target: target, Packages: map[string]packageTimings{}}
	if dir == "" {
		return merged, nil
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
		if t.Target == target {
			merged.add(t)
		}
	}
	return merged, nil
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
	held    map[string][]string
	timings timings
}

func newGoTestLog(w io.Writer) *goTestLog {
	return &goTestLog{w: w, held: map[string][]string{}, timings: timings{Packages: map[string]packageTimings{}}}
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
		return l.print(e.Output)
	case e.Action == "output" && !strings.HasPrefix(e.Output, "=== "):
		l.held[key] = append(l.held[key], e.Output)
	case e.Test != "" && (e.Action == "pass" || e.Action == "skip"):
		l.record(e)
		delete(l.held, key)
	case e.Test != "" && e.Action == "fail":
		l.record(e)
		return l.release(func(k string) bool { return k == key })
	case e.Test == "" && slices.Contains([]string{"pass", "fail", "skip"}, e.Action):
		l.record(e)
		prefix := e.Package + "\x00"
		if e.Action == "fail" {
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

func (l *goTestLog) record(e testEvent) {
	p := l.timings.Packages[e.Package]
	switch {
	case e.Test == "":
		p.Seconds += e.Elapsed
	case !strings.Contains(e.Test, "/"):
		if p.Tests == nil {
			p.Tests = map[string]float64{}
		}
		p.Tests[e.Test] += e.Elapsed
	}
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
