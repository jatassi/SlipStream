//go:build linux || darwin

package storing

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/user"
	"syscall"

	"golang.org/x/sys/unix"
)

const nativeStyle = Unix

func attrsOf(fs.DirEntry) uint32 { return 0 }

func listable(dir string) bool { return unix.Access(dir, unix.R_OK|unix.X_OK) == nil }

// account is the user SlipStream runs as, or its uid when the user
// database has no name for it, as in a container run with --user.
func account() (string, error) {
	u, err := user.Current()
	if errors.As(err, new(user.UnknownUserIdError)) {
		return fmt.Sprintf("uid %d", os.Getuid()), nil
	}
	if err != nil {
		return "", fmt.Errorf("the account SlipStream runs as: %w", err)
	}
	return u.Username, nil
}

func service() (bool, error) { return false, nil }

func (OSDisk) Writable(_ context.Context, dir string) (bool, error) {
	switch err := unix.Access(dir, unix.W_OK|unix.X_OK); {
	case errors.Is(err, unix.EACCES), errors.Is(err, unix.EPERM), errors.Is(err, unix.EROFS):
		return false, nil
	case err != nil:
		return false, &fs.PathError{Op: "access", Path: dir, Err: err}
	}
	var st unix.Statfs_t
	if err := unix.Statfs(dir, &st); err != nil {
		return false, &fs.PathError{Op: "statfs", Path: dir, Err: err}
	}
	return !readOnly(&st), nil
}

// fileID is info's inode where path's volume keeps inodes stable.
func fileID(path string, info fs.FileInfo) ([]byte, error) {
	var st unix.Statfs_t
	if err := unix.Statfs(path, &st); err != nil {
		return nil, &fs.PathError{Op: "statfs", Path: path, Err: err}
	}
	if !stableIDs(&st) {
		return nil, nil
	}
	return binary.BigEndian.AppendUint64(nil, info.Sys().(*syscall.Stat_t).Ino), nil //nolint:errcheck,forcetypeassert // os.Stat answers a *syscall.Stat_t on Unix
}

// ErrNoSpace is the error a write on a full volume fails with.
var ErrNoSpace error = syscall.ENOSPC

// NoSpace reports whether err is a write that ran out of space.
func NoSpace(err error) bool { return errors.Is(err, syscall.ENOSPC) }
