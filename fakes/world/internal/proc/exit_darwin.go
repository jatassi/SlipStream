package proc

import (
	"errors"
	"fmt"

	"golang.org/x/sys/unix"
)

// waitExit returns once the process pid has exited.
func waitExit(pid int) error {
	kq, err := unix.Kqueue()
	if err != nil {
		return fmt.Errorf("wait for PID %d: %w", pid, err)
	}
	defer func() { _ = unix.Close(kq) }()
	watch := []unix.Kevent_t{{Ident: uint64(pid), Filter: unix.EVFILT_PROC, Flags: unix.EV_ADD | unix.EV_ONESHOT, Fflags: unix.NOTE_EXIT}} //nolint:gosec // a PID is positive and fits
	if _, err := unix.Kevent(kq, watch, nil, nil); errors.Is(err, unix.ESRCH) {
		return nil
	} else if err != nil {
		return fmt.Errorf("wait for PID %d: %w", pid, err)
	}
	events := make([]unix.Kevent_t, 1)
	for {
		_, err := unix.Kevent(kq, nil, events, nil)
		if errors.Is(err, unix.EINTR) {
			continue
		}
		if err != nil {
			return fmt.Errorf("wait for PID %d: %w", pid, err)
		}
		return nil
	}
}
