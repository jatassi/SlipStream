package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jatassi/slipstream/fakes/world"
)

func cueCommand(args []string) int {
	if len(args) < 2 || len(args) > 3 {
		return usage(errors.New("cue: takes a run dir, a cue name and at most one JSON object of arguments"))
	}
	runDir, name := args[0], args[1]
	var raw json.RawMessage
	if len(args) == 3 {
		if !json.Valid([]byte(args[2])) {
			return usage(fmt.Errorf("cue: arguments %s are not JSON", args[2]))
		}
		raw = json.RawMessage(args[2])
	}
	ctx, stop := interrupted()
	defer stop()
	return journal(ctx, runDir, "cue", args[1:], func(ctx context.Context, c world.Controls, _ world.Step) (int, map[string]string) {
		return printed("cue", func() (any, error) { return c.Cue(ctx, name, raw) }), nil
	})
}
