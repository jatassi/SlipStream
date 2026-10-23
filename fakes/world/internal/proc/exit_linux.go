package proc

import (
	"errors"
	"fmt"

	"golang.org/x/sys/unix"
)

// waitExit returns once the process pid has exited.
func waitExit(pid int) error {
	fd, err := unix.PidfdOpen(pid, 0)
	if errors.Is(err, unix.ESRCH) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("wait for PID %d: %w", pid, err)
	}
	defer func() { _ = unix.Close(fd) }()
	fds := []unix.PollFd{{Fd: int32(fd), Events: unix.POLLIN}} //nolint:gosec // a file descriptor fits in int32
	for {
		_, err := unix.Poll(fds, -1)
		if errors.Is(err, unix.EINTR) {
			continue
		}
		if err != nil {
			return fmt.Errorf("wait for PID %d: %w", pid, err)
		}
		return nil
	}
}
