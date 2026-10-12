package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jatassi/slipstream/fakes/world"
)

// stop stops the run through its supervisor, prints the stop report and
// returns once the supervisor has exited.
func stop(f stopFlags) error {
	signals, stopSignals := interrupted()
	defer stopSignals()
	ctx, cancel := context.WithTimeout(signals, f.timeout)
	defer cancel()
	report, err := world.StopRun(ctx, f.runDir)
	if report.Nodes != nil {
		out, merr := json.MarshalIndent(report, "", "  ")
		if merr != nil {
			return fmt.Errorf("stop report: %w", merr)
		}
		fmt.Println(string(out))
	}
	return err
}
