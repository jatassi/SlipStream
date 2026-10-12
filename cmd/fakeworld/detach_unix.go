//go:build unix

package main

import "syscall"

// detached puts the supervisor in its own session, so the terminal's
// Ctrl+C and hangup never reach it.
func detached() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setsid: true}
}
