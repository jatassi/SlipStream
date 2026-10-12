package proc

import "syscall"

const (
	processQueryLimitedInformation = 0x1000
	stillActive                    = 259
)

// Alive reports whether a process with pid exists and has not exited.
func Alive(pid int) bool {
	h, err := syscall.OpenProcess(processQueryLimitedInformation, false, uint32(pid)) //nolint:gosec // a PID is positive and fits
	if err != nil {
		return false
	}
	defer func() { _ = syscall.CloseHandle(h) }()
	var code uint32
	return syscall.GetExitCodeProcess(h, &code) == nil && code == stillActive
}
