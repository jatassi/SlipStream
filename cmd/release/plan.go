package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"maps"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jatassi/slipstream/cmd/release/internal/changescope"
	"github.com/jatassi/slipstream/cmd/release/internal/native"
)

// lane is one kind of CI job: tests.yml or target.yml has a job named
// after each kind, which runs once per matrix entry the plan gives it.
type lane struct {
	kind string
	// tier is the dispatch input that selects the lane.
	tier string
	// targets are the only targets the lane runs on; nil runs it on every
	// target the run tests.
	targets []string
	// shardTime is the test step time each shard of a timed lane aims for,
	// and defaultShards how many shards it runs on a target that no recorded
	// run has timed. A lane without a shardTime runs one job per target.
	shardTime     time.Duration
	defaultShards int
	// onlyWhen, when set, keeps the lane off a run that does not run every
	// target unless the change touches what the lane checks.
	onlyWhen func(touched) bool
	// playwright runs the lane's Linux jobs in Playwright's container, which
	// ships the browsers and their system libraries.
	playwright bool
}

// touched says what the change touches that a lane or the packages
// depend on, as the scope command's outputs say.
type touched struct {
	packages bool
	diagram  bool
}

// lanes is the source of truth for what CI runs where. The build, go, e2e
// and browser lanes run on each target, since cgo and the OS change what
// they prove. Gofmt, go vet and golangci-lint run once per OS: no Go file
// is built for one architecture only, and lint compiles the cgo shim, so
// one OS cannot lint another's. Everything else runs once.
// The go lane's shard time keeps a wave's critical path short. An e2e
// shard runs its heaviest packages for most of its test step, so its shard
// time stays clear of go test's 25 minute -timeout. A browser shard ends
// its runs 2 minutes before the job's 45 minute limit.
var lanes = []lane{
	{kind: "build"},
	{kind: "lint", tier: "lint", targets: []string{"linux-amd64", "windows-amd64", "darwin-arm64"}},
	{kind: "lint-repo", tier: "lint", targets: []string{"linux-amd64"}},
	{kind: "go", tier: "go", shardTime: 10 * time.Minute, defaultShards: 4},
	{kind: "e2e", tier: "e2e", shardTime: 20 * time.Minute, defaultShards: 4},
	{kind: "browser", tier: "browser", playwright: true, shardTime: 30 * time.Minute, defaultShards: 4},
	{kind: "component", tier: "component", playwright: true, targets: []string{"linux-amd64"}},
	{kind: "diagram", tier: "component", playwright: true, targets: []string{"linux-amd64"}, onlyWhen: func(t touched) bool { return t.diagram }},
	{kind: "go-cache"},
}

// lanesOnBuild run against the build job's dist/.
var lanesOnBuild = []string{"e2e", "browser"}

func (l lane) inTargetWorkflow() bool {
	return l.kind == "build" || slices.Contains(lanesOnBuild, l.kind)
}

func (l lane) runsOn(target string) bool {
	return l.targets == nil || slices.Contains(l.targets, target)
}

// runKind is what starts a CI run, as it decides which lanes run where.
type runKind struct {
	// lanes are the lanes the run kind runs.
	lanes []string
	// everyTarget runs every target whatever the change touches.
	everyTarget bool
	packages    bool
	// widest caps the scope a change can widen the run to.
	widest changescope.Scope
	// budget caps the jobs the run's lanes take; zero leaves them uncapped.
	budget int
}

var testLanes = []string{"build", "lint", "lint-repo", "go", "e2e", "browser", "component", "diagram"}

// A wave is a pull request into a feature branch, which the train tests
// further after it merges: each push to a feature branch runs the e2e and
// browser lanes, and saves the Go cache the branch's waves restore. A merge
// is a pull request into main. A push to main ran its tests on its pull
// request, so it only saves the caches pull requests restore. The budgets
// keep a wave and a train inside GitHub Free's 20 concurrent jobs.
var runKinds = map[string]runKind{
	"wave":     {lanes: []string{"build", "lint", "lint-repo", "go", "component", "diagram"}, widest: changescope.PR, budget: 8},
	"train":    {lanes: []string{"build", "e2e", "browser", "go-cache"}, widest: changescope.PR, budget: 12},
	"merge":    {lanes: testLanes, widest: changescope.Native},
	"push":     {lanes: []string{"lint", "go-cache"}, everyTarget: true, widest: changescope.Native},
	"dispatch": {lanes: testLanes, everyTarget: true, widest: changescope.Native},
	"schedule": {lanes: testLanes, everyTarget: true, packages: true, widest: changescope.Native},
	"tag":      {lanes: testLanes, everyTarget: true, packages: true, widest: changescope.Native},
}

var scopeGates = map[changescope.Scope][]native.Gate{
	changescope.Docs:   nil,
	changescope.PR:     {native.GatePR},
	changescope.Native: {native.GatePR, native.GateNative},
}

type MatrixEntry struct {
	Target string `json:"target"`
	Runner string `json:"runner"`
	GOOS   string `json:"goos"`
	GOARCH string `json:"goarch"`
	// Xcode is the Xcode version the runner selects for the target.
	Xcode  string `json:"xcode,omitempty"`
	Shard  int    `json:"shard"`
	Shards int    `json:"shards"`
	// Container is the image the job runs in, empty to run on the runner.
	Container string `json:"container"`
	// Timings is the ID of the run whose timings the job splits its tier
	// by, empty when no run recorded the lane on the target.
	Timings string `json:"timings"`
}

type Matrix struct {
	Include []MatrixEntry `json:"include"`
}

func (m Matrix) forTarget(target string) Matrix {
	kept := Matrix{Include: []MatrixEntry{}}
	for _, e := range m.Include {
		if e.Target == target {
			kept.Include = append(kept.Include, e)
		}
	}
	return kept
}

type TargetCall struct {
	Target  string `json:"target"`
	Build   Matrix `json:"build"`
	E2E     Matrix `json:"e2e"`
	Browser Matrix `json:"browser"`
}

type TargetMatrix struct {
	Include []TargetCall `json:"include"`
}

func init() {
	register("plan", "print the CI matrix of each lane for a run kind: build, lint, lint-repo, go, e2e, browser, component, diagram and go-cache", runPlan)
}

func runPlan(ctx context.Context, args []string) error {
	names := slices.Sorted(maps.Keys(runKinds))
	flags := flag.NewFlagSet("plan", flag.ContinueOnError)
	kindName := flags.String("kind", "", "run kind: "+strings.Join(names, ", "))
	scope := changescope.PR
	flags.TextVar(&scope, "scope", scope, "change scope from the scope command: docs, pr, native")
	var t touched
	flags.BoolVar(&t.packages, "package", false, "the change touches what the packages are built from, as the scope command's package output says")
	flags.BoolVar(&t.diagram, "diagram", false, "the change touches what the Diagram check reads, as the scope command's diagram output says")
	targets := flags.String("targets", "", "comma-separated targets to keep; empty keeps every target the run kind runs")
	tier := flags.String("tier", "all", "lanes to run: all, "+strings.Join(laneTiers(), ", "))
	dir := flags.String("timings", "", "directory of recorded timings, a subdirectory per run named by its ID, to size the timed lanes' shards by")
	fetch := flags.Bool("fetch-timings", false, "first download into -timings the timings of the newest run that recorded each timed lane on each target, from GITHUB_REPOSITORY's artifacts through gh")
	if err := flags.Parse(args); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}
	k, ok := runKinds[*kindName]
	if !ok {
		return fmt.Errorf("-kind %q is not one of %s", *kindName, strings.Join(names, ", "))
	}
	if *fetch && *dir == "" {
		return errors.New("-fetch-timings needs -timings")
	}
	in, err := native.LoadInputs(native.InputsPath)
	if err != nil {
		return err
	}
	var keep []string
	if *targets != "" {
		keep = strings.Split(*targets, ",")
	}
	root, err := repoRoot(ctx)
	if err != nil {
		return err
	}
	packageJSON, err := os.ReadFile(filepath.Join(root, "web", "package.json"))
	if err != nil {
		return fmt.Errorf("read the web app's package.json: %w", err)
	}
	image, err := playwrightImage(packageJSON)
	if err != nil {
		return err
	}
	p, err := plan(in, k, scope, t, keep, *tier, image, nil)
	if err != nil {
		return err
	}
	if *fetch {
		if err := fetchTimings(ctx, *dir, os.Getenv("GITHUB_REPOSITORY"), p.timed()); err != nil {
			return err
		}
	}
	rec, err := loadRecords(*dir)
	if err != nil {
		return err
	}
	if p, err = plan(in, k, scope, t, keep, *tier, image, rec); err != nil {
		return err
	}
	for _, l := range lanes {
		m, err := json.Marshal(p.Lanes[l.kind])
		if err != nil {
			return fmt.Errorf("encode the %s matrix: %w", l.kind, err)
		}
		fmt.Printf("%s: %s\n", l.kind, m)
	}
	budget := "none"
	if k.budget > 0 {
		budget = strconv.Itoa(k.budget)
	}
	fmt.Printf("jobs: %d, budget %s\npackage: %t\n", p.Jobs(), budget, p.Package)
	encoded, err := json.Marshal(p.output())
	if err != nil {
		return fmt.Errorf("encode the plan: %w", err)
	}
	return githubOutput("plan", string(encoded))
}

// playwrightImage is the tag of Playwright's container for the
// @playwright/test version that package.json pins, so the browsers in the
// container are the ones that version drives.
func playwrightImage(packageJSON []byte) (string, error) {
	var pkg struct {
		DevDependencies map[string]string `json:"devDependencies"`
	}
	if err := json.Unmarshal(packageJSON, &pkg); err != nil {
		return "", fmt.Errorf("read the web app's package.json: %w", err)
	}
	v := pkg.DevDependencies["@playwright/test"]
	if !exactVersion.MatchString(v) {
		return "", fmt.Errorf("@playwright/test %q in the web app's package.json isn't an exact version, which Playwright's image tags need", v)
	}
	return "mcr.microsoft.com/playwright:v" + v + "-noble", nil
}

var exactVersion = regexp.MustCompile(`^\d+\.\d+\.\d+$`)

func laneTiers() []string {
	var tiers []string
	for _, l := range lanes {
		if l.tier != "" && !slices.Contains(tiers, l.tier) {
			tiers = append(tiers, l.tier)
		}
	}
	return tiers
}

// laneTarget is a lane on one target.
type laneTarget struct{ lane, target string }

// recorded is what one run recorded of a timed lane on a target: the run's
// ID, and the test step seconds its shards took together.
type recorded struct {
	run     string
	seconds float64
}

// loadRecords reads the timings in dir, a subdirectory per run named by its
// ID, as fetchTimings lays them out: one run for each lane on a target. A
// missing dir holds none.
func loadRecords(dir string) (map[laneTarget]recorded, error) {
	rec := map[laneTarget]recorded{}
	if dir == "" {
		return rec, nil
	}
	runs, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return rec, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read the recorded timings: %w", err)
	}
	for _, run := range runs {
		var files []string
		err := filepath.WalkDir(filepath.Join(dir, run.Name()), func(path string, d fs.DirEntry, err error) error {
			if err == nil && !d.IsDir() && filepath.Ext(path) == ".json" {
				files = append(files, path)
			}
			return err
		})
		if err != nil {
			return nil, fmt.Errorf("read the recorded timings: %w", err)
		}
		for _, f := range files {
			b, err := os.ReadFile(f)
			if err != nil {
				return nil, fmt.Errorf("read the recorded timings: %w", err)
			}
			var t timings
			if err := json.Unmarshal(b, &t); err != nil {
				return nil, fmt.Errorf("%s: %w", f, err)
			}
			key := laneTarget{t.Tier, t.Target}
			rec[key] = recorded{run.Name(), rec[key].seconds + t.Seconds}
		}
	}
	return rec, nil
}

// fetchTimings downloads into dir, for each lane on a target, the timings
// of the newest run that recorded it, as each shard's timings-<target>-<lane>-<shard>
// artifact. A run's first shard always records, so its artifact finds the run.
func fetchTimings(ctx context.Context, dir, repo string, timed []laneTarget) error {
	for _, lt := range timed {
		prefix := "timings-" + lt.target + "-" + lt.lane + "-"
		find := exec.CommandContext(ctx, "gh", "api", "repos/"+repo+"/actions/artifacts?per_page=1&name="+prefix+"1",
			"--jq", ".artifacts[] | select(.expired | not) | .workflow_run.id")
		find.Stderr = os.Stderr
		out, err := find.Output()
		if err != nil {
			return fmt.Errorf("find the newest timings of %s on %s: %w", lt.lane, lt.target, err)
		}
		id := strings.TrimSpace(string(out))
		if id == "" {
			fmt.Printf("%s on %s: no run has recorded timings\n", lt.lane, lt.target)
			continue
		}
		fmt.Printf("%s on %s: timings from run %s\n", lt.lane, lt.target, id)
		if err := run(ctx, "", nil, "gh", "run", "download", id, "--repo", repo, "--pattern", prefix+"*", "--dir", filepath.Join(dir, id)); err != nil {
			return err
		}
	}
	return nil
}

type Plan struct {
	// Lanes holds every lane's matrix, empty when the lane does not run.
	Lanes   map[string]Matrix
	Package bool
}

func (p Plan) PlannedJobs() map[string]bool {
	planned := map[string]bool{"target": len(p.Lanes["build"].Include) > 0}
	for _, l := range lanes {
		if !l.inTargetWorkflow() {
			planned[l.kind] = len(p.Lanes[l.kind].Include) > 0
		}
	}
	return planned
}

// output is the plan as tests.yml reads it: which jobs run, whether the run
// builds the packages, the matrix of each target's call to target.yml, and
// the matrix of every other lane.
func (p Plan) output() map[string]any {
	out := map[string]any{"planned": p.PlannedJobs(), "package": p.Package, "target": p.Targets()}
	for _, l := range lanes {
		if !l.inTargetWorkflow() {
			out[l.kind] = p.Lanes[l.kind]
		}
	}
	return out
}

// Jobs is how many jobs the run's lanes take.
func (p Plan) Jobs() int {
	n := 0
	for _, m := range p.Lanes {
		n += len(m.Include)
	}
	return n
}

// timed is each timed lane on each target the plan runs it on.
func (p Plan) timed() []laneTarget {
	var timed []laneTarget
	for _, l := range lanes {
		for _, e := range p.Lanes[l.kind].Include {
			if l.shardTime > 0 && e.Shard == 1 {
				timed = append(timed, laneTarget{l.kind, e.Target})
			}
		}
	}
	return timed
}

func (p Plan) Targets() TargetMatrix {
	m := TargetMatrix{Include: []TargetCall{}}
	for _, b := range p.Lanes["build"].Include {
		m.Include = append(m.Include, TargetCall{
			Target:  b.Target,
			Build:   p.Lanes["build"].forTarget(b.Target),
			E2E:     p.Lanes["e2e"].forTarget(b.Target),
			Browser: p.Lanes["browser"].forTarget(b.Target),
		})
	}
	return m
}

// sized is a lane on a target with the shards the plan gives it.
type sized struct {
	lane   lane
	target plannedTarget
	shards int
	// load is the lane's test time on the target in shard times.
	load float64
	run  string
}

// plan runs the Linux jobs of the playwright lanes in image, and splits
// each timed lane on a target into the shards its recorded test time
// needs to fit the lane's shard time, then into fewer to fit the run
// kind's budget.
func plan(in native.Inputs, k runKind, scope changescope.Scope, t touched, keep []string, tier, image string, rec map[laneTarget]recorded) (Plan, error) {
	if tier != "all" && !slices.Contains(laneTiers(), tier) {
		return Plan{}, fmt.Errorf("-tier %q is not one of all, %s", tier, strings.Join(laneTiers(), ", "))
	}
	packaging := k.packages || t.packages
	if k.everyTarget || packaging {
		scope = changescope.Native
	}
	targets, err := planTargets(in, scopeGates[min(scope, k.widest)], keep)
	if err != nil {
		return Plan{}, err
	}
	var runs []sized
	for _, l := range lanes {
		if !slices.Contains(k.lanes, l.kind) || !l.selected(tier, packaging) || (!k.everyTarget && l.onlyWhen != nil && !l.onlyWhen(t)) {
			continue
		}
		for _, target := range targets {
			if l.runsOn(target.Name) {
				runs = append(runs, l.sized(target, rec[laneTarget{l.kind, target.Name}]))
			}
		}
	}
	fit(runs, k.budget)
	p := Plan{Lanes: map[string]Matrix{}, Package: packaging}
	for _, l := range lanes {
		p.Lanes[l.kind] = Matrix{Include: []MatrixEntry{}}
	}
	for _, s := range runs {
		m := p.Lanes[s.lane.kind]
		m.Include = append(m.Include, s.entries(image)...)
		p.Lanes[s.lane.kind] = m
	}
	return p, nil
}

// sized gives a timed lane the shards its recorded test time on the target
// needs, or its default shards when no run recorded it there.
func (l lane) sized(t plannedTarget, r recorded) sized {
	s := sized{lane: l, target: t, shards: 1, run: r.run}
	switch {
	case l.shardTime == 0:
	case r.run == "":
		s.shards = l.defaultShards
		s.load = float64(l.defaultShards)
	default:
		s.load = r.seconds / l.shardTime.Seconds()
		s.shards = max(1, int(math.Ceil(s.load)))
	}
	return s
}

// fit takes shards away until the jobs fit the budget, each from the lane
// whose shards would then run the least past their shard time. A budget of
// zero leaves every lane its shards.
func fit(runs []sized, budget int) {
	jobs := 0
	for _, s := range runs {
		jobs += s.shards
	}
	for ; budget > 0 && jobs > budget; jobs-- {
		best := -1
		for i, s := range runs {
			if s.shards > 1 && (best < 0 || s.load/float64(s.shards-1) < runs[best].load/float64(runs[best].shards-1)) {
				best = i
			}
		}
		if best < 0 {
			return
		}
		runs[best].shards--
	}
}

type plannedTarget struct {
	native.Target
	xcode string
}

func planTargets(in native.Inputs, gates []native.Gate, keep []string) ([]plannedTarget, error) {
	for _, name := range keep {
		if _, err := in.Target(name); err != nil {
			return nil, err
		}
	}
	var targets []plannedTarget
	for _, t := range in.Targets {
		if !slices.Contains(gates, t.Gate) || (keep != nil && !slices.Contains(keep, t.Name)) {
			continue
		}
		tc, err := in.Toolchain(t)
		if err != nil {
			return nil, err
		}
		targets = append(targets, plannedTarget{t, tc.Xcode})
	}
	return targets, nil
}

// selected says whether a run of tier runs the lane. The build lane runs
// for packaging and for the lanes that run against its dist/.
func (l lane) selected(tier string, packaging bool) bool {
	switch {
	case tier == "all":
		return true
	case l.kind == "build":
		return packaging || slices.ContainsFunc(lanes, func(o lane) bool { return o.tier == tier && slices.Contains(lanesOnBuild, o.kind) })
	default:
		return l.tier == tier
	}
}

func (s sized) entries(image string) []MatrixEntry {
	t := s.target
	if !s.lane.playwright || t.GOOS != "linux" {
		image = ""
	}
	entries := make([]MatrixEntry, s.shards)
	for i := range entries {
		entries[i] = MatrixEntry{t.Name, t.Runner, t.GOOS, t.GOARCH, t.xcode, i + 1, s.shards, image, s.run}
	}
	return entries
}
