package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/jatassi/slipstream/fakes/world"
)

type doctorFlags struct {
	runDir, worktree string
	json             bool
	timeout          time.Duration
}

func parseDoctor(args []string) (doctorFlags, error) {
	if len(args) == 0 {
		return doctorFlags{}, errors.New("doctor: takes a run dir")
	}
	fs := newFlagSet("doctor")
	f := doctorFlags{runDir: args[0]}
	fs.StringVar(&f.worktree, "worktree", ".", "")
	fs.BoolVar(&f.json, "json", false, "")
	fs.DurationVar(&f.timeout, "timeout", defaultTimeout, "")
	if err := noArguments(fs, args[1:]); err != nil {
		return doctorFlags{}, err
	}
	return f, positiveTimeout(fs, f.timeout)
}

// doctor runs doctor's checks on the run and exits 1 when any fails. On a
// live run it journals itself and keeps its report as the step's stdout,
// where summary reads the last one; a run whose socket does not answer
// gets the report unjournaled, which says so.
func doctor(args []string) int {
	f, err := parseDoctor(args)
	if err != nil {
		return usage(err)
	}
	signals, stop := interrupted()
	defer stop()
	ctx, cancel := context.WithTimeout(signals, f.timeout)
	defer cancel()
	c, err := world.Dial(f.runDir)
	var s world.Step
	if err == nil {
		s, err = c.StartStep(ctx, world.StepStart{Verb: "doctor", Args: args[1:]})
	}
	if err != nil {
		return f.report(ctx, nil)
	}
	files := map[string]string{"stdout": world.StepOutputs(s.N, "")["stdout"]}
	out, err := create(f.runDir, files["stdout"])
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "fakeworld doctor: %v\n", err)
		files = nil
	}
	exit := f.report(world.InStep(ctx, s.N), out)
	if out != nil {
		_ = out.Close()
	}
	if _, err := c.EndStep(ctx, s.N, world.StepEnd{Exit: exit, Files: files}); err != nil {
		return failed("doctor", err)
	}
	return exit
}

func (f doctorFlags) report(ctx context.Context, keep *os.File) int {
	r := world.Doctor(ctx, f.runDir, f.worktree)
	text := r.String()
	if f.json {
		text = string(jsonLine(r))
	}
	fmt.Print(text)
	if keep != nil {
		if _, err := keep.WriteString(text); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "fakeworld doctor: keep the report: %v\n", err)
		}
	}
	if !r.OK {
		return exitFailed
	}
	return exitOK
}
