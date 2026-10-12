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
	kq, err := unix.Kqueue()
	if err != nil {
		return fmt.Errorf("wait for PID %d: %w", pid, err)
	}
	watch := []unix.Kevent_t{{Ident: uint64(pid), Filter: unix.EVFILT_PROC, Flags: unix.EV_ADD | unix.EV_ONESHOT, Fflags: unix.NOTE_EXIT}} //nolint:gosec // a PID is positive and fits
	if _, err := unix.Kevent(kq, watch, nil, nil); err != nil {
		_ = unix.Close(kq)
		if errors.Is(err, unix.ESRCH) {
			return nil
		}
		return fmt.Errorf("wait for PID %d: %w", pid, err)
	}
	exited := make(chan error, 1)
	go func() {
		defer func() { _ = unix.Close(kq) }()
		events := make([]unix.Kevent_t, 1)
		for {
			_, err := unix.Kevent(kq, nil, events, nil)
			if !errors.Is(err, unix.EINTR) {
				exited <- err
				return
			}
		}
	}()
	return waitFor(ctx, pid, exited)
}
