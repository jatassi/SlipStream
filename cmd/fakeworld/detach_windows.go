package main

import (
	"syscall"

	"golang.org/x/sys/windows"
)

// detached gives the supervisor no console and its own process group, so
// the console's Ctrl+C and close never reach it.
func detached() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{CreationFlags: windows.DETACHED_PROCESS | windows.CREATE_NEW_PROCESS_GROUP}
}
