//go:build unix

package proc

import (
	"errors"
	"os"
	"syscall"
)

// Alive reports whether a process with pid exists.
func Alive(pid int) bool {
	return !errors.Is(syscall.Kill(pid, 0), syscall.ESRCH)
}

// Watched is a process by its PID: Unix does not reuse a PID within the
// seconds a check waits.
type Watched struct{ pid int }

// Watch watches the process pid.
func Watch(pid int) (*Watched, error) { return &Watched{pid}, nil }

// Gone reports whether the watched process no longer exists. A zombie
// still exists until its parent reaps it.
func (w *Watched) Gone() bool { return !Alive(w.pid) }

// Wait returns once the watched process has exited, which may be before
// its parent reaps it.
func (w *Watched) Wait() error { return waitExit(w.pid) }

func (w *Watched) Close() {}

// terminated is false: Kill fails on a process that has exited only once
// it is waited on, with os.ErrProcessDone.
func terminated(*os.Process) bool { return false }
