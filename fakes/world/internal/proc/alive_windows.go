package proc

import (
	"errors"
	"fmt"
	"os"

	"golang.org/x/sys/windows"
)

// Alive reports whether a process with pid exists and its process object is
// not yet signaled. Its exit code is set before that, while the process
// still holds its executable.
func Alive(pid int) bool {
	h, err := windows.OpenProcess(windows.SYNCHRONIZE, false, uint32(pid)) //nolint:gosec // a PID is positive and fits
	if err != nil {
		return false
	}
	defer func() { _ = windows.CloseHandle(h) }()
	return running(h)
}

func running(h windows.Handle) bool {
	event, err := windows.WaitForSingleObject(h, 0)
	return err == nil && event == uint32(windows.WAIT_TIMEOUT)
}

// Watched holds a handle to its process, and Windows reuses no PID while a
// handle to its process is open.
type Watched struct{ h windows.Handle }

// Watch watches the process pid. Take it while pid still names that
// process: once the process is reaped, Windows can give pid to another.
func Watch(pid int) (*Watched, error) {
	h, err := windows.OpenProcess(windows.SYNCHRONIZE, false, uint32(pid)) //nolint:gosec // a PID is positive and fits
	if errors.Is(err, windows.ERROR_INVALID_PARAMETER) {
		return &Watched{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("watch PID %d: %w", pid, err)
	}
	return &Watched{h}, nil
}

// Gone reports whether the watched process has exited, as Alive does.
func (w *Watched) Gone() bool { return w.h == 0 || !running(w.h) }

func (w *Watched) Close() {
	if w.h != 0 {
		_ = windows.CloseHandle(w.h)
	}
}

// terminated reports whether p has exited. Kill fails on one that has:
// TerminateProcess with access denied until Wait releases p's handle, and
// with EINVAL after, when WithHandle fails too.
func terminated(p *os.Process) bool {
	signaled := false
	err := p.WithHandle(func(h uintptr) { signaled = !running(windows.Handle(h)) })
	return signaled || err != nil
}
