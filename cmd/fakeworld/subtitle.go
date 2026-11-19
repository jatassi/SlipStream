package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/world"
)

// subtitleCommand prints a catalog subtitle's file as its provider serves
// it, so an Admin's upload can follow a file's speech or other speech.
func subtitleCommand(args []string) int {
	fs := newFlagSet("subtitle")
	var episode int
	fs.IntVar(&episode, "episode", 0, "")
	positional, err := parse(fs, args)
	if err != nil {
		return usage(err)
	}
	if len(positional) != 2 {
		return usage(errors.New("subtitle: takes a run dir and a catalog subtitle's id"))
	}
	runDir, id := positional[0], positional[1]
	ctx, stop := interrupted()
	defer stop()
	return read(ctx, runDir, "subtitle", args, func(context.Context) int {
		m, err := world.ReadLastManifest(runDir)
		if err != nil {
			return failed("subtitle", err)
		}
		data, err := os.ReadFile(filepath.Join(runDir, "scenario.yaml"))
		if err != nil {
			return failed("subtitle", err)
		}
		s, err := scenario.Parse(m.Scenario.Name, data)
		if err != nil {
			return failed("subtitle", err)
		}
		text, err := world.CatalogText(s, id, episode)
		if err != nil {
			return failed("subtitle", err)
		}
		if _, err := os.Stdout.Write(text); err != nil {
			return failed("subtitle", err)
		}
		return exitOK
	})
}
