//go:build windows

package main

import (
	"os/exec"
	"syscall"
)

// setSysProcAttrDetached configures a command to break away from the parent's
// Windows Job Object. On Windows, processes launched by a parent that belongs
// to a Job Object with JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE will be terminated
// when the parent exits and the last Job handle is closed. This flag detaches
// the child so it survives independently.
func setSysProcAttrDetached(cmd *exec.Cmd) {
	const createBreakawayFromJob = 0x01000000
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: createBreakawayFromJob | syscall.CREATE_NEW_PROCESS_GROUP,
	}
}
