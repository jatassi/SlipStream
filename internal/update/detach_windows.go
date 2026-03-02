//go:build windows

package update

import (
	"os/exec"
	"syscall"
)

// setSysProcAttrDetached configures a command to break away from the parent's
// Windows Job Object, ensuring the child process survives if the parent exits
// and the Job Object is closed with JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE.
func setSysProcAttrDetached(cmd *exec.Cmd) {
	const createBreakawayFromJob = 0x01000000
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: createBreakawayFromJob | syscall.CREATE_NEW_PROCESS_GROUP,
	}
}
