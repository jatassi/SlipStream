package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jatassi/slipstream/fakes/world"
)

// summary writes summary.md into the run dir and prints its path. It
// reads the run's files alone, so it works on a live run, a stopped run
// and a downloaded bundle, and journals nothing.
func summary(args []string) int {
	if len(args) != 1 {
		return usage(errors.New("summary: takes one run dir"))
	}
	text, err := world.Summary(args[0])
	if err != nil {
		return failed("summary", err)
	}
	path := filepath.Join(args[0], world.SummaryFile)
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		return failed("summary", err)
	}
	fmt.Println(path)
	return exitOK
}
