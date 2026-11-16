//go:build unix

package notifying

import (
	"os/exec"
	"syscall"
)

var passedVars = []string{"PATH", "HOME", "TMPDIR"}

func sameVar(a, b string) bool { return a == b }

// startTree starts cmd as the leader of its own process group, which
// stopping it kills whole. end kills what is left of the group once the
// script has exited.
func startTree(cmd *exec.Cmd) (end func(), err error) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
	if err := cmd.Start(); err != nil {
		return nil, err //nolint:wrapcheck // Run reports it as why the script didn't start
	}
	return func() { _ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }, nil
}
