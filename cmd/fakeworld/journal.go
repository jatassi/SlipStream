package main

import (
	"context"
	"errors"

	"github.com/jatassi/slipstream/fakes/world"
)

// stepBody is a verb's work inside its journaled step. It returns the
// step's exit and the files it kept.
type stepBody func(ctx context.Context, c world.Controls, s world.Step) (exit int, files map[string]string)

// journal runs body as one step of the live run in runDir: step.start
// with verb and its arguments after the run dir, then body, then step.end
// with body's exit.
func journal(ctx context.Context, runDir, verb string, args []string, body stepBody) int {
	c, err := world.Dial(runDir)
	if err != nil {
		return failed(verb, err)
	}
	return journalOn(ctx, c, verb, args, body)
}

func journalOn(ctx context.Context, c world.Controls, verb string, args []string, body stepBody) int {
	s, err := c.StartStep(ctx, world.StepStart{Verb: verb, Args: args})
	if err != nil {
		return failed(verb, err)
	}
	exit, files := body(world.InStep(ctx, s.N), c, s)
	if _, err := c.EndStep(ctx, s.N, world.StepEnd{Exit: exit, Files: files}); err != nil {
		return failed(verb, err)
	}
	return exit
}

// read runs a reader: journaled on a live run, so a replay re-runs its
// wait, and as it is on a stopped run or a bundle.
func read(ctx context.Context, runDir, verb string, args []string, body func(ctx context.Context) int) int {
	c, err := world.Dial(runDir)
	if errors.Is(err, world.ErrNotLive) {
		return body(ctx)
	}
	if err != nil {
		return failed(verb, err)
	}
	return journalOn(ctx, c, verb, args, func(ctx context.Context, _ world.Controls, _ world.Step) (int, map[string]string) {
		return body(ctx), nil
	})
}
