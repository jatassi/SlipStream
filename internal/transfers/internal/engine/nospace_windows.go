package engine

import (
	"errors"
	"syscall"
)

// noSpace reports whether err is a write that ran out of space:
// ERROR_HANDLE_DISK_FULL or ERROR_DISK_FULL.
func noSpace(err error) bool {
	return errors.Is(err, syscall.Errno(39)) || errors.Is(err, syscall.Errno(112))
}
