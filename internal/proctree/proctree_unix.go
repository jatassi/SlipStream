//go:build unix

package proctree

import (
	"os/exec"
	"syscall"
)

// Start starts cmd as the leader of its own process group, which end or
// cmd's context ending kills whole.
func Start(cmd *exec.Cmd) (end func(), err error) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
	if err := cmd.Start(); err != nil {
		return nil, err //nolint:wrapcheck // the caller reports it as why its program didn't start
	}
	return func() { _ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }, nil
}
