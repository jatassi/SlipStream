package proc

import (
	"context"
	"errors"
	"fmt"

	"golang.org/x/sys/windows"
)

// WaitExit returns once the process pid has exited, whether or not it is
// this process's child, or when ctx ends. Its wait outlives a ctx that
// ends first, until the process exits.
func WaitExit(ctx context.Context, pid int) error {
	h, err := windows.OpenProcess(windows.SYNCHRONIZE, false, uint32(pid)) //nolint:gosec // a PID is positive and fits
	if errors.Is(err, windows.ERROR_INVALID_PARAMETER) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("wait for PID %d: %w", pid, err)
	}
	exited := make(chan error, 1)
	go func() {
		defer func() { _ = windows.CloseHandle(h) }()
		_, err := windows.WaitForSingleObject(h, windows.INFINITE)
		exited <- err
	}()
	return waitFor(ctx, pid, exited)
}
