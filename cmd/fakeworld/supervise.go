package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/world"
)

func supervise(f superviseFlags) int {
	ctx, abandon := context.WithCancel(context.Background())
	defer abandon()
	go abandonOnStdinEOF(os.Stdin, abandon)
	run, err := launch(ctx, f)
	if err != nil {
		writeReport(report{Error: err.Error()})
		return exitFailed
	}
	writeReport(report{OK: true})
	<-run.Done()
	if _, err := run.Stop(ctx); err != nil {
		return exitFailed
	}
	return exitOK
}

func abandonOnStdinEOF(stdin io.Reader, abandon context.CancelFunc) {
	_, _ = io.Copy(io.Discard, stdin)
	abandon()
}

func launch(ctx context.Context, f superviseFlags) (*world.Run, error) {
	s, err := loadScenario(f.scenario, f.scenarioFile)
	if err != nil {
		return nil, err
	}
	return world.Launch(ctx, world.Options{Binary: f.binary, Scenario: s, RunDir: f.runDir, RunDirCreated: true, Origin: originOf(f.testCommand)})
}

func originOf(testCommand string) world.Origin {
	if testCommand == "" {
		return world.Origin{Kind: world.FromSteps}
	}
	return world.Origin{Kind: world.FromTest, Command: testCommand}
}

// loadScenario is the catalog Scenario name, or the Scenario name the
// file holds.
func loadScenario(name, file string) (scenario.Scenario, error) {
	if file == "" {
		return scenario.Load(name)
	}
	data, err := os.ReadFile(file)
	if err != nil {
		return scenario.Scenario{}, fmt.Errorf("scenario %s: %w", name, err)
	}
	return scenario.Parse(name, data)
}

func writeReport(r report) {
	line, err := json.Marshal(r)
	if err != nil {
		panic("fakeworld supervise: " + err.Error())
	}
	_, _ = os.Stdout.Write(append(line, '\n'))
}
