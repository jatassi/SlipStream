package storing

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strconv"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

// PathLimit is the longest absolute path macOS opens, in bytes: PATH_MAX.
func PathLimit() int64 { return 1024 }

func readOnly(st *unix.Statfs_t) bool { return st.Flags&unix.MNT_RDONLY != 0 }

// Volume reads statfs on path itself, which names its mount point.
func (OSDisk) Volume(_ context.Context, path string) (Volume, error) {
	var st unix.Statfs_t
	if err := unix.Statfs(path, &st); err != nil {
		return Volume{}, &fs.PathError{Op: "statfs", Path: path, Err: err}
	}
	mount := unix.ByteSliceToString(st.Mntonname[:])
	return Volume{
		ID:        fmt.Sprintf("%08x%08x", uint32(st.Fsid.Val[0]), uint32(st.Fsid.Val[1])), //nolint:gosec // the id's bits, not a number
		Name:      mount,
		Mount:     mount,
		Kind:      kindOf(&st),
		Total:     int64(st.Blocks) * int64(st.Bsize), //nolint:gosec // block counts fit int64 for any disk
		Free:      int64(st.Bavail) * int64(st.Bsize), //nolint:gosec // block counts fit int64 for any disk
		SampledAt: time.Now(),
	}, nil
}

// Places is "/" and then each volume mounted in /Volumes, leaving out the
// link there to the startup disk.
func (d OSDisk) Places(ctx context.Context) ([]Place, error) {
	des, err := os.ReadDir("/Volumes")
	if err != nil {
		return nil, err //nolint:wrapcheck // a PathError names the call and the path
	}
	paths := []string{"/"}
	for _, de := range des {
		p := "/Volumes/" + de.Name()
		var st unix.Statfs_t
		if unix.Statfs(p, &st) == nil && unix.ByteSliceToString(st.Mntonname[:]) == p {
			paths = append(paths, p)
		}
	}
	return placesAt(ctx, d, paths), nil
}

func KindOf(path string) (Kind, error) {
	var st unix.Statfs_t
	if err := unix.Statfs(path, &st); err != nil {
		return 0, &fs.PathError{Op: "statfs", Path: path, Err: err}
	}
	return kindOf(&st), nil
}

func kindOf(st *unix.Statfs_t) Kind {
	switch {
	case networkFSName(unix.ByteSliceToString(st.Fstypename[:])):
		return Network
	case st.Flags&unix.MNT_REMOVABLE != 0:
		return Removable
	}
	return Local
}

func networkFSName(name string) bool {
	return name == "nfs" || name == "smbfs" || name == "afpfs" || name == "webdav"
}

// deviceOf is st's device in hex.
func deviceOf(st *syscall.Stat_t) string {
	return strconv.FormatUint(uint64(uint32(st.Dev)), 16) //nolint:gosec // the device's bits, not a number
}

func stableIDs(st *unix.Statfs_t) bool { return stableFSName(unix.ByteSliceToString(st.Fstypename[:])) }

// stableFSName is a filesystem whose inode numbers survive a remount: msdos
// and exfat number files by where their entries lie, and a network
// filesystem's numbers are the server's to change.
func stableFSName(name string) bool {
	return !networkFSName(name) && name != "msdos" && name != "exfat"
}

// Clone clones from to to+".part" with clonefile, which makes it whole or
// not at all, and renames it to to. A .part a crash left is removed first,
// because clonefile won't write over a file.
func (OSDisk) Clone(_ context.Context, from, to string) error {
	part := to + ".part"
	if err := os.Remove(part); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err //nolint:wrapcheck // a PathError names the call and the path
	}
	if err := unix.Clonefile(from, part, 0); err != nil {
		return cloneError("clonefile", part, err)
	}
	return os.Rename(part, to) //nolint:wrapcheck // a LinkError names the call and both paths
}
