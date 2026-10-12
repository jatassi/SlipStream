package proc

import (
	"context"
	"errors"
	"fmt"

	"golang.org/x/sys/unix"
)

// WaitExit returns once the process pid has exited, whether or not it is
// this process's child, or when ctx ends. Its wait outlives a ctx that
// ends first, until the process exits.
func WaitExit(ctx context.Context, pid int) error {
	fd, err := unix.PidfdOpen(pid, 0)
	if errors.Is(err, unix.ESRCH) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("wait for PID %d: %w", pid, err)
	}
	exited := make(chan error, 1)
	go func() {
		defer func() { _ = unix.Close(fd) }()
		fds := []unix.PollFd{{Fd: int32(fd), Events: unix.POLLIN}} //nolint:gosec // a file descriptor fits in int32
		for {
			_, err := unix.Poll(fds, -1)
			if !errors.Is(err, unix.EINTR) {
				exited <- err
				return
			}
		}
	}()
	return waitFor(ctx, pid, exited)
}
