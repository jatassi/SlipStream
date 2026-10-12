//go:build unix

package proc

import (
	"errors"
	"syscall"
)

// Alive reports whether a process with pid exists.
func Alive(pid int) bool {
	return !errors.Is(syscall.Kill(pid, 0), syscall.ESRCH)
}
