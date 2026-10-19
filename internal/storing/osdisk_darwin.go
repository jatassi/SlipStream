package storing

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"time"

	"golang.org/x/sys/unix"
)

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

// KindOf is local, removable or network for the volume holding path.
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
