package storing

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

func readOnly(st *unix.Statfs_t) bool { return st.Flags&unix.ST_RDONLY != 0 }

// Volume reads statfs on path itself and names the volume by the mount
// that holds it in /proc/self/mountinfo.
func (OSDisk) Volume(_ context.Context, path string) (Volume, error) {
	var st unix.Statfs_t
	if err := unix.Statfs(path, &st); err != nil {
		return Volume{}, &fs.PathError{Op: "statfs", Path: path, Err: err}
	}
	var s unix.Stat_t
	if err := unix.Stat(path, &s); err != nil {
		return Volume{}, &fs.PathError{Op: "stat", Path: path, Err: err}
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return Volume{}, fmt.Errorf("volume of %s: %w", path, err)
	}
	ms, err := readMountinfo()
	if err != nil {
		return Volume{}, err
	}
	dev := device(uint64(s.Dev)) //nolint:unconvert // Dev is uint32 on some architectures
	m, ok := mountOf(ms, dev, resolved)
	if !ok {
		return Volume{}, fmt.Errorf("volume of %s: no mount holds it", path)
	}
	id := fmt.Sprintf("%08x%08x", uint32(st.Fsid.Val[0]), uint32(st.Fsid.Val[1])) //nolint:gosec // the id's bits, not a number
	if st.Fsid.Val == [2]int32{} {
		id = "dev-" + dev
	}
	bsize := int64(st.Bsize) //nolint:unconvert // Bsize is int32 on some architectures
	return Volume{
		ID:        id,
		Name:      m.Point,
		Mount:     m.Point,
		Kind:      kindOf(int64(st.Type), dev), //nolint:unconvert // Type is int32 on some architectures
		Total:     int64(st.Blocks) * bsize,    //nolint:gosec // block counts fit int64 for any disk
		Free:      int64(st.Bavail) * bsize,    //nolint:gosec // block counts fit int64 for any disk
		SampledAt: time.Now(),
	}, nil
}

// Places is "/" and then each real mount that is a folder.
func (d OSDisk) Places(ctx context.Context) ([]Place, error) {
	ms, err := readMountinfo()
	if err != nil {
		return nil, err
	}
	paths := []string{"/"}
	for _, m := range realMounts(ms) {
		if info, err := os.Stat(m.Point); m.Point != "/" && err == nil && info.IsDir() {
			paths = append(paths, m.Point)
		}
	}
	return placesAt(ctx, d, paths), nil
}

func KindOf(path string) (Kind, error) {
	var st unix.Statfs_t
	if err := unix.Statfs(path, &st); err != nil {
		return 0, &fs.PathError{Op: "statfs", Path: path, Err: err}
	}
	var s unix.Stat_t
	if err := unix.Stat(path, &s); err != nil {
		return 0, &fs.PathError{Op: "stat", Path: path, Err: err}
	}
	return kindOf(int64(st.Type), device(uint64(s.Dev))), nil //nolint:unconvert // Type and Dev are 32-bit on some architectures
}

func kindOf(magic int64, dev string) Kind {
	switch {
	case networkMagic(magic):
		return Network
	case removable(dev):
		return Removable
	}
	return Local
}

// networkMagic is NFS, SMB, CIFS or SMB2 (which SMB3 shares) by its
// statfs f_type.
func networkMagic(t int64) bool {
	switch t {
	case unix.NFS_SUPER_MAGIC, unix.SMB_SUPER_MAGIC, unix.CIFS_SUPER_MAGIC, unix.SMB2_SUPER_MAGIC:
		return true
	}
	return false
}

func stableIDs(st *unix.Statfs_t) bool { return stableMagic(int64(st.Type)) } //nolint:unconvert // Type is int32 on some architectures

// stableMagic is a filesystem whose inode numbers survive a remount by its
// statfs f_type: vfat and exFAT number files anew on each mount, and a
// network filesystem's numbers are the server's to change.
func stableMagic(t int64) bool {
	return !networkMagic(t) && t != unix.MSDOS_SUPER_MAGIC && t != unix.EXFAT_SUPER_MAGIC
}

// removable reads the block device's removable flag, which a partition
// leaves to its disk. The path is not cleaned: the kernel resolves ".."
// after the device's link.
func removable(dev string) bool {
	for _, p := range []string{"/sys/dev/block/" + dev + "/removable", "/sys/dev/block/" + dev + "/../removable"} {
		if b, err := os.ReadFile(p); err == nil { //nolint:gosec // a sysfs file named by a device number
			return strings.TrimSpace(string(b)) == "1"
		}
	}
	return false
}

func device(dev uint64) string { return fmt.Sprintf("%d:%d", unix.Major(dev), unix.Minor(dev)) }

func readMountinfo() ([]Mount, error) {
	b, err := os.ReadFile("/proc/self/mountinfo")
	if err != nil {
		return nil, err //nolint:wrapcheck // a PathError names the call and the path
	}
	return parseMountinfo(bytes.NewReader(b))
}
