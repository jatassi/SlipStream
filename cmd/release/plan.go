package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/cmd/release/internal/changescope"
	"github.com/jatassi/slipstream/cmd/release/internal/native"
)

// eventRunsEveryTarget says which CI events build every target regardless of
// the change scope. push means a push to main, and tag a push of a v* tag.
var eventRunsEveryTarget = map[string]bool{
	"pr":       false,
	"push":     true,
	"schedule": true,
	"tag":      true,
}

// eventPackages says which CI events package and smoke-test every target
// regardless of what the change touches.
var eventPackages = map[string]bool{"tag": true, "schedule": true}

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
	Xcode string `json:"xcode,omitempty"`
}

type Matrix struct {
	Include []MatrixEntry `json:"include"`
}

func init() {
	register("plan", "print the CI matrix of targets for an event", runPlan)
}

func runPlan(_ context.Context, args []string) error {
	events := make([]string, 0, len(eventRunsEveryTarget))
	for e := range eventRunsEveryTarget {
		events = append(events, e)
	}
	sort.Strings(events)
	fs := flag.NewFlagSet("plan", flag.ContinueOnError)
	event := fs.String("event", "", "CI event: "+strings.Join(events, ", "))
	scope := changescope.PR
	fs.TextVar(&scope, "scope", scope, "change scope from the scope command: docs, pr, native")
	touched := fs.Bool("package", false, "the change touches what the packages are built from, as the scope command's package output says")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}
	if _, ok := eventRunsEveryTarget[*event]; !ok {
		return fmt.Errorf("-event %q is not one of %s", *event, strings.Join(events, ", "))
	}
	in, err := native.LoadInputs(native.InputsPath)
	if err != nil {
		return err
	}
	p, err := plan(in, *event, scope, *touched)
	if err != nil {
		return err
	}
	matrix, err := json.Marshal(p.Matrix)
	if err != nil {
		return fmt.Errorf("encode the matrix: %w", err)
	}
	fmt.Println(string(matrix))
	if err := githubOutput("matrix", string(matrix)); err != nil {
		return err
	}
	if err := githubOutput("build", strconv.FormatBool(len(p.Matrix.Include) > 0)); err != nil {
		return err
	}
	return githubOutput("package", strconv.FormatBool(p.Package))
}

// Plan is what CI runs for an event: the targets it builds, and whether
// each of them also packages and smoke-tests.
type Plan struct {
	Matrix  Matrix
	Package bool
}

// plan decides the CI run for event. touched says the change touches what
// the packages are built from; packaging covers every target, since each
// target builds its own packages.
func plan(in native.Inputs, event string, scope changescope.Scope, touched bool) (Plan, error) {
	packaging := eventPackages[event] || touched
	if eventRunsEveryTarget[event] || packaging {
		scope = changescope.Native
	}
	gates := scopeGates[scope]
	m := Matrix{Include: []MatrixEntry{}}
	for _, t := range in.Targets {
		if !slices.Contains(gates, t.Gate) {
			continue
		}
		tc, err := in.Toolchain(t)
		if err != nil {
			return Plan{}, err
		}
		m.Include = append(m.Include, MatrixEntry{t.Name, t.Runner, t.GOOS, t.GOARCH, tc.Xcode})
	}
	return Plan{m, packaging}, nil
}
