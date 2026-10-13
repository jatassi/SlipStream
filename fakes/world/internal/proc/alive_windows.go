package proc

import "golang.org/x/sys/windows"

// Alive reports whether a process with pid exists and its process object is
// not yet signaled. Its exit code is set before that, while the process
// still holds its executable.
func Alive(pid int) bool {
	h, err := windows.OpenProcess(windows.SYNCHRONIZE, false, uint32(pid)) //nolint:gosec // a PID is positive and fits
	if err != nil {
		return false
	}
	defer func() { _ = windows.CloseHandle(h) }()
	event, err := windows.WaitForSingleObject(h, 0)
	return err == nil && event == uint32(windows.WAIT_TIMEOUT)
}
