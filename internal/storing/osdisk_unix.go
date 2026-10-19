//go:build linux || darwin

package storing

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/user"

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
