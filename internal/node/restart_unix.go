//go:build unix

package node

import (
	"io"
	"syscall"
)

// processRestart execs exe in place with args: the same PID, the same
// stdio, and every other descriptor closed, since Go opens each one
// close-on-exec. A failed exec returns, reported.
func processRestart(exe string, args, environ []string, logsDir string, stderr io.Writer) func() error {
	return func() error {
		err := syscall.Exec(exe, args, successorEnviron(environ)) //nolint:gosec // G204: this process's own executable and arguments
		return restartFailed(logsDir, stderr, err)
	}
}
