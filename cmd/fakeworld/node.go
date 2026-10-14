package main

import (
	"context"
	"errors"

	"github.com/jatassi/slipstream/fakes/world"
)

func nodeCommand(args []string) int {
	if len(args) != 3 {
		return usage(errors.New("node: takes an action, a run dir and a Node name"))
	}
	a, err := world.ParseNodeAction(args[0])
	if err != nil {
		return usage(err)
	}
	runDir, name := args[1], args[2]
	ctx, stop := interrupted()
	defer stop()
	return journal(ctx, runDir, "node "+args[0], []string{name}, func(ctx context.Context, c world.Controls, _ world.Step) (int, map[string]string) {
		return printed("node", func() (any, error) { return c.Node(ctx, name, a) }), nil
	})
}
