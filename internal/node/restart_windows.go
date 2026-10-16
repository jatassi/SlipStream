package node

import (
	"context"
	"io"
	"os"
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"
)

// processRestart starts exe with args as a new process that inherits this
// one's stdio, since Windows has no exec in place; this process then
// exits, and the successor waits for it to release the data dir. The app
// gets no console window.
func processRestart(exe string, args, environ []string, logsDir string, stderr io.Writer) func() error {
	return func() error {
		cmd := exec.CommandContext(context.Background(), exe, args[1:]...) //nolint:gosec // G204: this process's own executable and arguments
		cmd.Env = successorEnviron(environ)
		cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
		if len(args) > 1 && args[1] == string(ModeApp) {
			cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: windows.CREATE_NO_WINDOW}
		}
		if err := cmd.Start(); err != nil {
			return restartFailed(logsDir, stderr, err)
		}
		_ = cmd.Process.Release()
		return nil
	}
}
