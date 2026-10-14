package main

import (
	"context"
	"encoding/json"
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
	if _, err := run.Stop(); err != nil {
		return exitFailed
	}
	return exitOK
}

func abandonOnStdinEOF(stdin io.Reader, abandon context.CancelFunc) {
	_, _ = io.Copy(io.Discard, stdin)
	abandon()
}

func launch(ctx context.Context, f superviseFlags) (*world.Run, error) {
	s, err := scenario.Load(f.scenario)
	if err != nil {
		return nil, err
	}
	return world.Launch(ctx, world.Options{Binary: f.binary, Scenario: s, RunDir: f.runDir, RunDirCreated: true, Origin: world.Origin{Kind: world.FromSteps}})
}

func writeReport(r report) {
	line, err := json.Marshal(r)
	if err != nil {
		panic("fakeworld supervise: " + err.Error())
	}
	_, _ = os.Stdout.Write(append(line, '\n'))
}
