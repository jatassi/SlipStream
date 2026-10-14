package main

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/jatassi/slipstream/fakes/world"
)

var fsCommands = map[string]func(runDir string, args []string) int{
	"stat":  fsStat,
	"match": fsMatch,
}

func fsCommand(args []string) int {
	if len(args) < 2 {
		return usage(errors.New("fs: takes stat or match and a run dir"))
	}
	command, ok := fsCommands[args[0]]
	if !ok {
		return usage(fmt.Errorf("fs: unknown command %q", args[0]))
	}
	return command(args[1], args[2:])
}

// inRun is path made absolute, refused unless it lies in the run dir or
// the run's scratch dir.
func inRun(m world.Manifest, path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("path: %w", err)
	}
	for _, dir := range []string{m.RunDir, m.ScratchDir} {
		if rel, err := filepath.Rel(dir, abs); err == nil && filepath.IsLocal(rel) {
			return abs, nil
		}
	}
	return "", fmt.Errorf("%s is outside the run dir %s and the scratch dir %s", abs, m.RunDir, m.ScratchDir)
}

func fsStat(runDir string, args []string) int {
	if len(args) != 1 {
		return usage(errors.New("fs stat: takes a run dir and a path"))
	}
	ctx, stop := interrupted()
	defer stop()
	return read(ctx, runDir, "fs stat", args, func(context.Context) int {
		return printed("fs stat", func() (any, error) {
			m, err := world.ReadLastManifest(runDir)
			if err != nil {
				return nil, err
			}
			path, err := inRun(m, args[0])
			if err != nil {
				return nil, err
			}
			return world.StatFile(path)
		})
	})
}

func fsMatch(runDir string, args []string) int {
	fs := newFlagSet("fs match")
	var release string
	fs.StringVar(&release, "release", "", "")
	positional, err := parse(fs, args)
	if err != nil {
		return usage(err)
	}
	if len(positional) != 1 {
		return usage(errors.New("fs match: takes a run dir and a path"))
	}
	if err := required(fs, "release"); err != nil {
		return usage(err)
	}
	ctx, stop := interrupted()
	defer stop()
	return read(ctx, runDir, "fs match", args, func(context.Context) int {
		return printed("fs match", func() (any, error) {
			m, err := world.ReadLastManifest(runDir)
			if err != nil {
				return nil, err
			}
			path, err := inRun(m, positional[0])
			if err != nil {
				return nil, err
			}
			return world.MatchRelease(m, path, release)
		})
	})
}
