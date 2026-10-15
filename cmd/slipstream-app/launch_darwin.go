package main

import (
	"fmt"
	"os"
	"syscall"
)

// launch replaces the launcher with node in the same PID, so LaunchServices
// keeps tracking the app.
func launch(node string, args []string) error {
	//nolint:gosec // G204: the slipstream binary beside the launcher
	if err := syscall.Exec(node, append([]string{node}, args...), os.Environ()); err != nil {
		return fmt.Errorf("exec: %w", err)
	}
	return nil
}

// report writes err to stderr, which launchd keeps in the unified log.
func report(err error) {
	_, _ = fmt.Fprintln(os.Stderr, "slipstream-app:", err)
}
