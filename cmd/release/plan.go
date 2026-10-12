package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/jatassi/slipstream/cmd/release/internal/native"
)

// eventRunsNativeTargets says which CI events build the native-gated targets
// regardless of the native-touch rule. push means a push to main.
var eventRunsNativeTargets = map[string]bool{
	"pr":       false,
	"push":     true,
	"schedule": true,
	"release":  true,
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

func runPlan(args []string) error {
	events := make([]string, 0, len(eventRunsNativeTargets))
	for e := range eventRunsNativeTargets {
		events = append(events, e)
	}
	sort.Strings(events)
	fs := flag.NewFlagSet("plan", flag.ContinueOnError)
	event := fs.String("event", "", "CI event: "+strings.Join(events, ", "))
	touchesNative := fs.Bool("native", false, "the change touches native code (from touches-native)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if _, ok := eventRunsNativeTargets[*event]; !ok {
		return fmt.Errorf("-event %q is not one of %s", *event, strings.Join(events, ", "))
	}
	in, err := native.LoadInputs(native.InputsPath)
	if err != nil {
		return err
	}
	m, err := plan(in, *event, *touchesNative)
	if err != nil {
		return err
	}
	matrix, err := json.Marshal(m)
	if err != nil {
		return err
	}
	fmt.Println(string(matrix))
	return githubOutput("matrix", string(matrix))
}

func plan(in native.Inputs, event string, touchesNative bool) (Matrix, error) {
	gates := []native.Gate{native.GatePR}
	if eventRunsNativeTargets[event] || touchesNative {
		gates = append(gates, native.GateNative)
	}
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
