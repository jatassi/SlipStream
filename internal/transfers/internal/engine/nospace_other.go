//go:build !windows

package engine

import (
	"errors"
	"syscall"
)

// noSpace reports whether err is a write that ran out of space.
func noSpace(err error) bool { return errors.Is(err, syscall.ENOSPC) }
