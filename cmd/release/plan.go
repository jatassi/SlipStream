package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/cmd/release/internal/changescope"
	"github.com/jatassi/slipstream/cmd/release/internal/native"
)

// lane is one kind of CI job: ci.yml has a job named after each kind, which
// runs once per matrix entry the plan gives it.
type lane struct {
	kind string
	// tier is the dispatch input that selects the lane.
	tier string
	// shards holds how many jobs the lane splits into on each target; a
	// target it leaves out does not run the lane.
	shards map[string]int
	// onlyWhen, when set, keeps the lane off an event that does not run
	// every target unless the change touches what the lane checks.
	onlyWhen func(touched) bool
}

// touched says what the change touches that a lane or the packages
// depend on, as the scope command's outputs say.
type touched struct {
	packages bool
	diagram  bool
}

// lanes is the source of truth for what CI runs where. Gofmt, go vet,
// golangci-lint and the go, e2e and browser tiers run on each target, since
// cgo and the OS change what they prove; everything else runs once.
// The browser tier's shard counts follow its runtime on each runner, which
// is CPU-bound, so it scales out across jobs rather than Playwright workers.
var lanes = []lane{
	{kind: "build", shards: map[string]int{"linux-amd64": 1, "linux-arm64": 1, "windows-amd64": 1, "darwin-arm64": 1}},
	{kind: "lint", tier: "lint", shards: map[string]int{"linux-amd64": 1, "linux-arm64": 1, "windows-amd64": 1, "darwin-arm64": 1}},
	{kind: "lint-repo", tier: "lint", shards: map[string]int{"linux-amd64": 1}},
	{kind: "go", tier: "go", shards: map[string]int{"linux-amd64": 2, "linux-arm64": 2, "windows-amd64": 3, "darwin-arm64": 2}},
	{kind: "e2e", tier: "e2e", shards: map[string]int{"linux-amd64": 1, "linux-arm64": 1, "windows-amd64": 1, "darwin-arm64": 1}},
	{kind: "browser", tier: "browser", shards: map[string]int{"linux-amd64": 4, "linux-arm64": 1, "windows-amd64": 2, "darwin-arm64": 3}},
	{kind: "component", tier: "component", shards: map[string]int{"linux-amd64": 1}},
	{kind: "diagram", tier: "component", shards: map[string]int{"linux-amd64": 1}, onlyWhen: func(t touched) bool { return t.diagram }},
}

// lanesOnBuild run against the build job's dist/.
var lanesOnBuild = []string{"e2e", "browser"}

type event struct {
	// everyTarget runs every target whatever the change touches.
	everyTarget bool
	packages    bool
	// widest caps the scope a change can widen the run to.
	widest changescope.Scope
}

// A stacked pull request merges into another pull request's branch, which
// runs every target itself before it reaches main.
var events = map[string]event{
	"pr":       {widest: changescope.Native},
	"stacked":  {widest: changescope.PR},
	"push":     {everyTarget: true, widest: changescope.Native},
	"dispatch": {everyTarget: true, widest: changescope.Native},
	"schedule": {everyTarget: true, packages: true, widest: changescope.Native},
	"tag":      {everyTarget: true, packages: true, widest: changescope.Native},
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
}

type Matrix struct {
	Include []MatrixEntry `json:"include"`
}

func init() {
	register("plan", "print the CI matrix of each lane for an event: build, lint, lint-repo, go, e2e, browser, component and diagram", runPlan)
}

func runPlan(_ context.Context, args []string) error {
	names := slices.Sorted(maps.Keys(events))
	fs := flag.NewFlagSet("plan", flag.ContinueOnError)
	eventName := fs.String("event", "", "CI event: "+strings.Join(names, ", "))
	scope := changescope.PR
	fs.TextVar(&scope, "scope", scope, "change scope from the scope command: docs, pr, native")
	var t touched
	fs.BoolVar(&t.packages, "package", false, "the change touches what the packages are built from, as the scope command's package output says")
	fs.BoolVar(&t.diagram, "diagram", false, "the change touches what the Diagram check reads, as the scope command's diagram output says")
	targets := fs.String("targets", "", "comma-separated targets to keep; empty keeps every target the event runs")
	tier := fs.String("tier", "all", "lanes to run: all, "+strings.Join(laneTiers(), ", "))
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}
	e, ok := events[*eventName]
	if !ok {
		return fmt.Errorf("-event %q is not one of %s", *eventName, strings.Join(names, ", "))
	}
	in, err := native.LoadInputs(native.InputsPath)
	if err != nil {
		return err
	}
	var keep []string
	if *targets != "" {
		keep = strings.Split(*targets, ",")
	}
	p, err := plan(in, e, scope, t, keep, *tier)
	if err != nil {
		return err
	}
	for _, l := range lanes {
		m, err := json.Marshal(p.Lanes[l.kind])
		if err != nil {
			return fmt.Errorf("encode the %s matrix: %w", l.kind, err)
		}
		fmt.Printf("%s: %s\n", l.kind, m)
		if err := githubOutput(l.kind, string(m)); err != nil {
			return err
		}
	}
	planned, err := json.Marshal(p.Planned())
	if err != nil {
		return fmt.Errorf("encode the planned lanes: %w", err)
	}
	fmt.Printf("planned: %s\npackage: %t\n", planned, p.Package)
	if err := githubOutput("planned", string(planned)); err != nil {
		return err
	}
	return githubOutput("package", strconv.FormatBool(p.Package))
}

func laneTiers() []string {
	var tiers []string
	for _, l := range lanes {
		if l.tier != "" && !slices.Contains(tiers, l.tier) {
			tiers = append(tiers, l.tier)
		}
	}
	return tiers
}

type Plan struct {
	// Lanes holds every lane's matrix, empty when the lane does not run.
	Lanes   map[string]Matrix
	Package bool
}

// Planned says, by lane, whether the lane runs, which the gate job
// checks each lane's result against.
func (p Plan) Planned() map[string]bool {
	planned := map[string]bool{}
	for kind, m := range p.Lanes {
		planned[kind] = len(m.Include) > 0
	}
	return planned
}

func plan(in native.Inputs, e event, scope changescope.Scope, t touched, keep []string, tier string) (Plan, error) {
	if tier != "all" && !slices.Contains(laneTiers(), tier) {
		return Plan{}, fmt.Errorf("-tier %q is not one of all, %s", tier, strings.Join(laneTiers(), ", "))
	}
	packaging := e.packages || t.packages
	if e.everyTarget || packaging {
		scope = changescope.Native
	}
	targets, err := planTargets(in, scopeGates[min(scope, e.widest)], keep)
	if err != nil {
		return Plan{}, err
	}
	p := Plan{Lanes: map[string]Matrix{}, Package: packaging}
	for _, l := range lanes {
		m := Matrix{Include: []MatrixEntry{}}
		if l.selected(tier, packaging) && (e.everyTarget || l.onlyWhen == nil || l.onlyWhen(t)) {
			for _, target := range targets {
				m.Include = append(m.Include, l.entries(target)...)
			}
		}
		p.Lanes[l.kind] = m
	}
	return p, nil
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

func (l lane) entries(t plannedTarget) []MatrixEntry {
	n := l.shards[t.Name]
	entries := make([]MatrixEntry, n)
	for i := range entries {
		entries[i] = MatrixEntry{t.Name, t.Runner, t.GOOS, t.GOARCH, t.xcode, i + 1, n}
	}
	return entries
}
