package atomicfile

import (
	"errors"
	"syscall"
)

// SyncDir does nothing: Windows cannot flush a directory handle.
func SyncDir(string) error { return nil }

const errorSharingViolation syscall.Errno = 32

func ephemeral(err error) bool {
	var errno syscall.Errno
	if !errors.As(err, &errno) {
		return false
	}
	return errno == syscall.ERROR_ACCESS_DENIED || errno == syscall.ERROR_FILE_NOT_FOUND || errno == errorSharingViolation
}
