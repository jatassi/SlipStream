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
// the change scope. push means a push to main.
var eventRunsEveryTarget = map[string]bool{
	"pr":       false,
	"push":     true,
	"schedule": true,
	"release":  true,
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
	m, err := plan(in, *event, scope)
	if err != nil {
		return err
	}
	matrix, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("encode the matrix: %w", err)
	}
	fmt.Println(string(matrix))
	if err := githubOutput("matrix", string(matrix)); err != nil {
		return err
	}
	return githubOutput("build", strconv.FormatBool(len(m.Include) > 0))
}

func plan(in native.Inputs, event string, scope changescope.Scope) (Matrix, error) {
	if eventRunsEveryTarget[event] {
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
			return Matrix{}, err
		}
		m.Include = append(m.Include, MatrixEntry{t.Name, t.Runner, t.GOOS, t.GOARCH, tc.Xcode})
	}
	return m, nil
}
