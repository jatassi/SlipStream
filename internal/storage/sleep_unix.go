//go:build unix

package storage

import (
	"time"

	"golang.org/x/sys/unix"
)

func sleepOS(d time.Duration) {
	tv := unix.NsecToTimeval(d.Nanoseconds())
	_, _ = unix.Select(0, nil, nil, nil, &tv)
}
