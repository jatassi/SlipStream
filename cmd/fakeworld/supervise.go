package main

import (
	"context"
	"encoding/json"
	"io"
	"os"

	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/world"
)

// supervise launches the world and holds it until its stop channel stops
// it. Its stdin is start's: EOF before it reports abandons the launch.
func supervise(f superviseFlags) int {
	ctx, abandon := context.WithCancel(context.Background())
	defer abandon()
	go func() {
		_, _ = io.Copy(io.Discard, os.Stdin)
		abandon()
	}()
	run, err := launch(ctx, f)
	if err != nil {
		writeReport(report{Error: err.Error()})
		return exitFailed
	}
	writeReport(report{OK: true})
	<-run.Done()
	if _, err := run.Stop(); err != nil {
		return exitFailed
	}
	return exitOK
}

func launch(ctx context.Context, f superviseFlags) (*world.Run, error) {
	s, err := scenario.Load(f.scenario)
	if err != nil {
		return nil, err
	}
	return world.Launch(ctx, world.Options{Binary: f.binary, Scenario: s, RunDir: f.runDir, RunDirCreated: true})
}

func writeReport(r report) {
	line, err := json.Marshal(r)
	if err != nil {
		panic("fakeworld supervise: " + err.Error())
	}
	_, _ = os.Stdout.Write(append(line, '\n'))
}
