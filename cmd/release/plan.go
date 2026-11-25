package main

import (
	"cmp"
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
	// maxShardTime, when set, is the longest test step a budget may trim
	// the lane's shards to, in place of the job-wide maxShardTime.
	maxShardTime time.Duration
	// onlyWhen, when set, keeps the lane off a run that does not run every
	// target unless the change touches what the lane checks.
	onlyWhen func(touched) bool
	// playwright runs the lane's Linux jobs in Playwright's container, which
	// ships the browsers and their system libraries.
	playwright bool
	// limit is the job's timeout in minutes.
	limit int
	// nonBlocking lanes have their job in quarantine.yml, which each run
	// calls beside tests.yml and outside its gate.
	nonBlocking bool
}

// touched says what the change touches that a lane or the packages
// depend on, as the scope command's outputs say.
type touched struct {
	packages bool
	diagram  bool
}

// lanes is the source of truth for what CI can run where. The build, go
// and e2e lanes run on each target, since cgo and the OS change what they
// prove. The lint lane runs gofmt, go vet and golangci-lint once per OS: no
// Go file is built for one architecture only, and lint compiles the cgo
// shim, so one OS cannot lint another's. On linux it runs the repo-wide
// checks too, which no OS changes. The browser lane runs Chromium and
// Firefox on linux-amd64 and WebKit on both it and darwin, whose WebKit is
// the only one that paints as Safari does, so no other target adds an
// engine. The platform lane runs the platform suite where a run skips the
// go and e2e lanes. The quarantine lane runs on each target where a row of
// ci/quarantine.tsv quarantines a test. Everything else runs once.
// The go lane's shard time keeps a wave's critical path short. An e2e
// shard runs its heaviest packages for most of its test step, so its shard
// time stays clear of go test's 25 minute -timeout. A browser shard ends
// its runs 2 minutes before the job's 45 minute limit.
var lanes = []lane{
	{kind: "build", limit: 45},
	{kind: "lint", tier: "lint", targets: []string{"linux-amd64", "windows-amd64", "darwin-arm64"}, limit: 20},
	{kind: "go", tier: "go", shardTime: 10 * time.Minute, defaultShards: 4, limit: 30},
	// A budget trims no e2e shard past 20 minutes: a longer shard runs
	// longer package parts, each of which go test's 25 minute -timeout
	// bounds.
	{kind: "e2e", tier: "e2e", shardTime: 20 * time.Minute, maxShardTime: 20 * time.Minute, defaultShards: 4, limit: 45},
	{kind: "browser", tier: "browser", targets: []string{"linux-amd64", "darwin-arm64"}, playwright: true, shardTime: 30 * time.Minute, defaultShards: 4, limit: 45},
	{kind: "component", tier: "component", playwright: true, targets: []string{"linux-amd64"}, limit: 20},
	{kind: "diagram", tier: "component", playwright: true, targets: []string{"linux-amd64"}, onlyWhen: func(t touched) bool { return t.diagram }, limit: 15},
	{kind: "go-cache", limit: 45},
	{kind: "platform", tier: "platform", targets: []string{"windows-amd64", "darwin-arm64", "linux-arm64"}, limit: 45},
	{kind: "quarantine", limit: 45, nonBlocking: true},
	{kind: "online", tier: "online", targets: []string{"linux-amd64"}, limit: 20},
}

// lanesOnBuild run against the build job's dist/.
var lanesOnBuild = []string{"e2e", "browser"}

func (l lane) inTargetWorkflow() bool {
	return l.kind == "build" || slices.Contains(lanesOnBuild, l.kind)
}

func (l lane) runsOn(target string) bool {
	return l.targets == nil || slices.Contains(l.targets, target)
}

// reach is which targets a run kind runs a lane on, before the lane's own
// targets narrow them.
type reach int

const (
	// prTargets are the targets every pull request tests.
	prTargets reach = iota
	// packageTargets are every target when the change touches what the
	// packages are built from, and the pull request targets otherwise. A
	// change to the native libraries alone needs no build there, since the
	// platform tests build on each target.
	packageTargets
	allTargets
)

// gates are the gates of the targets the reach takes a change of scope to.
func (r reach) gates(scope changescope.Scope, packaging bool) []native.Gate {
	switch {
	case scope == changescope.Docs:
		return nil
	case r == allTargets || r == packageTargets && packaging:
		return []native.Gate{native.GatePR, native.GateNative}
	default:
		return []native.Gate{native.GatePR}
	}
}

// runKind is what starts a CI run, as it decides which lanes run where.
type runKind struct {
	// lanes maps each lane the run kind runs to its reach.
	lanes map[string]reach
	// everyTarget runs the lanes whatever the change touches.
	everyTarget bool
	packages    bool
	// budget caps the jobs the run's lanes take; zero leaves them uncapped.
	budget int
}

func reaching(r reach, kinds ...string) map[string]reach {
	m := map[string]reach{}
	for _, k := range kinds {
		m[k] = r
	}
	return m
}

var testLanes = []string{"build", "lint", "go", "e2e", "browser", "component", "diagram"}

// A wave is a pull request into a feature branch, which the train tests
// further after it merges: each push to a feature branch runs the e2e and
// browser lanes on linux-amd64 and the platform suite on the other
// targets, and saves the Go cache the branch's waves restore. A merge is a
// pull request into main, which runs the lanes of both and lints on each
// OS. A change to the native libraries or the packages builds the packages
// on every target. The daily check runs the online conformance tests, and
// as a schedule run each lane on each of its targets too, saving each
// target's Go cache. A dispatch runs what its tier picks, and a tag runs
// every lane but online, with linux-amd64's browsers alone. Each run but a
// dispatch or a tag runs the quarantined tests on its targets. The budgets
// keep a wave and a train inside GitHub Free's 20 concurrent jobs.
var runKinds = map[string]runKind{
	"wave": {lanes: reaching(prTargets, "build", "lint", "go", "component", "diagram", "quarantine"), budget: 8},
	"train": {
		lanes:  map[string]reach{"build": prTargets, "e2e": prTargets, "browser": prTargets, "go-cache": prTargets, "platform": allTargets, "quarantine": allTargets},
		budget: 12,
	},
	"merge": {
		lanes: map[string]reach{
			"build": packageTargets, "lint": allTargets, "go": prTargets, "e2e": prTargets,
			"browser": prTargets, "component": prTargets, "diagram": prTargets, "platform": allTargets, "quarantine": allTargets,
		},
		budget: 18,
	},
	"daily":    {lanes: reaching(allTargets, "online", "quarantine"), everyTarget: true},
	"schedule": {lanes: reaching(allTargets, append(testLanes, "go-cache", "online", "quarantine")...), everyTarget: true, packages: true},
	"dispatch": {lanes: reaching(allTargets, append(testLanes, "platform", "online")...), everyTarget: true},
	"tag": {
		lanes: map[string]reach{
			"build": allTargets, "lint": allTargets, "go": allTargets, "e2e": allTargets,
			"browser": prTargets, "component": allTargets, "diagram": allTargets,
		},
		everyTarget: true,
		packages:    true,
	},
}

// MatrixEntry is one job of a lane. It carries what a job's steps would
// otherwise branch on the runner's OS for.
type MatrixEntry struct {
	Target string `json:"target"`
	Runner string `json:"runner"`
	GOOS   string `json:"goos"`
	GOARCH string `json:"goarch"`
	// DeveloperDir selects the Xcode the target pins, empty where it pins
	// none.
	DeveloperDir string `json:"developer-dir"`
	// Exe ends the name of each of the target's executables.
	Exe string `json:"exe"`
	// Fixtures is where the worlds keep the fixtures they encode, as
	// actions/cache names the path on the target's runner.
	Fixtures string `json:"fixtures"`
	// Timeout is the job's limit in minutes.
	Timeout int `json:"timeout"`
	// Repo runs the repo-wide checks after lint's Go checks.
	Repo   bool `json:"repo"`
	Shard  int  `json:"shard"`
	Shards int  `json:"shards"`
	// Container is the image the job runs in, empty to run on the runner.
	Container string `json:"container"`
	// Timings is the ID of the run whose timings the job splits its tier
	// by, empty when no run recorded the lane on the target.
	Timings string `json:"timings"`
}

// fixtureCaches is os.UserCacheDir's slipstream-fixtures on each OS, where
// fakes/internal/releases keeps the fixtures it encodes.
var fixtureCaches = map[string]string{
	"linux":   "~/.cache/slipstream-fixtures",
	"darwin":  "~/Library/Caches/slipstream-fixtures",
	"windows": `~\AppData\Local\slipstream-fixtures`,
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
	register("plan", "print the CI matrix of each lane for a run kind: build, lint, go, e2e, browser, component, diagram, go-cache, platform, quarantine and online", runPlan)
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
	image, err := playwrightImage(root)
	if err != nil {
		return err
	}
	q, err := loadQuarantine(root)
	if err != nil {
		return err
	}
	plans := func(rec map[laneTarget]recorded) (Plan, error) {
		return plan(in, k, scope, t, keep, *tier, image, q, rec)
	}
	p, err := plans(nil)
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
	if p, err = plans(rec); err != nil {
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
// @playwright/test version that the web app's package.json pins, so the
// browsers in the container are the ones that version drives.
func playwrightImage(root string) (string, error) {
	packageJSON, err := os.ReadFile(filepath.Join(root, "web", "package.json"))
	if err != nil {
		return "", fmt.Errorf("read the web app's package.json: %w", err)
	}
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
// ID, and the test step seconds its shards took together, zero when its
// timings predate the seconds.
type recorded struct {
	run     string
	seconds float64
}

// loadRecords reads the timings in dir, a subdirectory per run named by its
// ID, as fetchTimings lays them out: one run for each lane on a target. Each
// file is named <target>-<lane>-<shard>.json, as the jobs record it; the
// name, not the content, says the lane, since timings from before the plan
// sized lanes carry no tier. A missing dir holds none.
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
			lane, _, _ := strings.Cut(strings.TrimPrefix(strings.TrimSuffix(filepath.Base(f), ".json"), t.Target+"-"), "-")
			key := laneTarget{lane, t.Target}
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
	// floor is the fewest shards whose test steps end inside the job's
	// limit, which no budget takes the lane below.
	floor int
	run   string
}

// plan runs the Linux jobs of the playwright lanes in image, and splits
// each timed lane on a target into the shards its recorded test time
// needs to fit the lane's shard time, then into fewer to fit the run
// kind's budget. It runs the quarantine lane on each target where a row of
// quarantine reaches a lane.
func plan(in native.Inputs, k runKind, scope changescope.Scope, t touched, keep []string, tier, image string, quarantine []quarantined, rec map[laneTarget]recorded) (Plan, error) {
	if tier != "all" && !slices.Contains(laneTiers(), tier) {
		return Plan{}, fmt.Errorf("-tier %q is not one of all, %s", tier, strings.Join(laneTiers(), ", "))
	}
	packaging := k.packages || t.packages
	if k.everyTarget || packaging {
		scope = changescope.Native
	}
	var runs []sized
	for _, l := range lanes {
		r, ok := k.lanes[l.kind]
		if !ok || (!k.everyTarget && l.onlyWhen != nil && !l.onlyWhen(t)) {
			continue
		}
		targets, err := planTargets(in, r.gates(scope, packaging), keep)
		if err != nil {
			return Plan{}, err
		}
		for _, target := range targets {
			if l.runsOn(target.Name) && l.selected(tier, packaging, target.Name) && l.hasTests(quarantine, target.Name) {
				runs = append(runs, l.sized(target, rec[laneTarget{l.kind, target.Name}]))
			}
		}
	}
	runs = withoutCoveredPlatform(runs)
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

// withoutCoveredPlatform drops the platform lane from each target that runs
// the go and e2e lanes, since the platform suite is a part of them.
func withoutCoveredPlatform(runs []sized) []sized {
	full := map[string]int{}
	for _, s := range runs {
		if s.lane.kind == "go" || s.lane.kind == "e2e" {
			full[s.target.Name]++
		}
	}
	return slices.DeleteFunc(runs, func(s sized) bool { return s.lane.kind == "platform" && full[s.target.Name] == 2 })
}

// maxShardTime is the longest test step a shard may plan: the e2e and
// browser jobs end at 45 minutes, and setup takes up to 5.
const maxShardTime = 40 * time.Minute

// sized gives a timed lane the shards its recorded test time on the target
// needs. A lane no run recorded the seconds of there takes its default
// shards, each a full shard time, which a budget trims no further than a
// recorded time of that length.
func (l lane) sized(t plannedTarget, r recorded) sized {
	s := sized{lane: l, target: t, shards: 1, run: r.run}
	if l.shardTime == 0 {
		return s
	}
	seconds := cmp.Or(r.seconds, float64(l.defaultShards)*l.shardTime.Seconds())
	s.load = seconds / l.shardTime.Seconds()
	s.shards = max(1, int(math.Ceil(s.load)))
	s.floor = int(math.Ceil(seconds / cmp.Or(l.maxShardTime, maxShardTime).Seconds()))
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
			if s.shards > max(1, s.floor) && (best < 0 || s.load/float64(s.shards-1) < runs[best].load/float64(runs[best].shards-1)) {
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

// hasTests says whether the quarantine lane has a test to run on target,
// which every other lane always has.
func (l lane) hasTests(quarantine []quarantined, target string) bool {
	return l.kind != "quarantine" || slices.ContainsFunc(quarantine, func(q quarantined) bool { return q.laneRunsOn(target) })
}

// selected says whether a run of tier runs the lane on target. The build
// lane runs for packaging and for the lanes that run against its dist/.
func (l lane) selected(tier string, packaging bool, target string) bool {
	switch {
	case tier == "all":
		return true
	case l.kind == "build":
		return packaging || slices.ContainsFunc(lanes, func(o lane) bool {
			return o.tier == tier && slices.Contains(lanesOnBuild, o.kind) && o.runsOn(target)
		})
	default:
		return l.tier == tier
	}
}

func (s sized) entries(image string) []MatrixEntry {
	t := s.target
	if !s.lane.playwright || t.GOOS != "linux" {
		image = ""
	}
	e := MatrixEntry{
		Target: t.Name, Runner: t.Runner, GOOS: t.GOOS, GOARCH: t.GOARCH,
		Fixtures: fixtureCaches[t.GOOS], Timeout: s.lane.limit, Repo: s.lane.kind == "lint" && t.GOOS == "linux",
		Shards: s.shards, Container: image, Timings: s.run,
	}
	if t.xcode != "" {
		e.DeveloperDir = "/Applications/Xcode_" + t.xcode + ".app/Contents/Developer"
	}
	if t.GOOS == "windows" {
		e.Exe = ".exe"
		// Windows runners take an e2e shard past 35 minutes with every test
		// passing.
		if s.lane.kind == "e2e" {
			e.Timeout += 5
		}
	}
	entries := make([]MatrixEntry, s.shards)
	for i := range entries {
		entries[i] = e
		entries[i].Shard = i + 1
	}
	return entries
}
