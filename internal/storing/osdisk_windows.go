package storing

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os/user"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
)

const nativeStyle = Windows

// Directory access rights that x/sys/windows leaves out.
const (
	fileAddFile         = 0x2
	fileAddSubdirectory = 0x4
)

func attrsOf(de fs.DirEntry) uint32 {
	info, err := de.Info()
	if err != nil {
		return 0
	}
	if d, ok := info.Sys().(*syscall.Win32FileAttributeData); ok {
		return d.FileAttributes
	}
	return 0
}

func listable(dir string) bool { return openDir(dir, windows.FILE_LIST_DIRECTORY) == nil }

func account() (string, error) {
	u, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("the account SlipStream runs as: %w", err)
	}
	return u.Username, nil
}

func service() (bool, error) {
	s, err := svc.IsWindowsService()
	if err != nil {
		return false, fmt.Errorf("whether SlipStream runs as a Windows service: %w", err)
	}
	return s, nil
}

// Writable asks for the right to add files and folders to dir, which the
// kernel checks against its ACL without writing, and reads the volume's
// read-only flag.
func (OSDisk) Writable(_ context.Context, dir string) (bool, error) {
	switch err := openDir(dir, fileAddFile|fileAddSubdirectory); {
	case errors.Is(err, windows.ERROR_ACCESS_DENIED), errors.Is(err, windows.ERROR_WRITE_PROTECT):
		return false, nil
	case err != nil:
		return false, &fs.PathError{Op: "open", Path: dir, Err: err}
	}
	root, err := volumeRoot(dir)
	if err != nil {
		return false, err
	}
	_, _, flags, err := volumeInfo(root)
	if err != nil {
		return false, err
	}
	return flags&windows.FILE_READ_ONLY_VOLUME == 0, nil
}

// Volume reads the volume holding path: its root, label and serial, and
// the bytes free to this account.
func (OSDisk) Volume(_ context.Context, path string) (Volume, error) {
	root, err := volumeRoot(path)
	if err != nil {
		return Volume{}, err
	}
	label, serial, _, err := volumeInfo(root)
	if err != nil {
		return Volume{}, err
	}
	p, err := windows.UTF16PtrFromString(longPath(path))
	if err != nil {
		return Volume{}, &fs.PathError{Op: "GetDiskFreeSpaceEx", Path: path, Err: err}
	}
	var free, total uint64
	if err := windows.GetDiskFreeSpaceEx(p, &free, &total, nil); err != nil {
		return Volume{}, &fs.PathError{Op: "GetDiskFreeSpaceEx", Path: path, Err: err}
	}
	return Volume{
		ID:        fmt.Sprintf("%08X", serial),
		Name:      volumeName(root, label),
		Mount:     displayRoot(root) + `\`,
		Kind:      kindOfRoot(root),
		Total:     int64(total),
		Free:      int64(free),
		SampledAt: time.Now(),
	}, nil
}

// Places is each fixed, removable and network drive; optical drives hold
// no library.
func (d OSDisk) Places(ctx context.Context) ([]Place, error) {
	mask, err := windows.GetLogicalDrives()
	if err != nil {
		return nil, fmt.Errorf("GetLogicalDrives: %w", err)
	}
	var paths []string
	for i := range 26 {
		if mask&(1<<i) == 0 {
			continue
		}
		root := []uint16{uint16('A' + i), ':', '\\', 0}
		switch windows.GetDriveType(&root[0]) {
		case windows.DRIVE_FIXED, windows.DRIVE_REMOVABLE, windows.DRIVE_REMOTE:
			paths = append(paths, windows.UTF16ToString(root))
		}
	}
	return placesAt(ctx, d, paths), nil
}

func KindOf(path string) (Kind, error) {
	if isUNC(path) {
		return Network, nil
	}
	root, err := volumeRoot(path)
	if err != nil {
		return 0, err
	}
	return kindOfRoot(root), nil
}

func kindOfRoot(root string) Kind {
	if isUNC(root) {
		return Network
	}
	p, err := windows.UTF16PtrFromString(root)
	if err != nil {
		return Local
	}
	switch windows.GetDriveType(p) {
	case windows.DRIVE_REMOTE:
		return Network
	case windows.DRIVE_REMOVABLE:
		return Removable
	}
	return Local
}

// volumeRoot is the root of the volume holding path, such as `D:\`,
// `\\nas\media\` or a folder a volume is mounted on, with a trailing
// separator.
func volumeRoot(path string) (string, error) {
	p, err := windows.UTF16FromString(longPath(path))
	if err != nil {
		return "", &fs.PathError{Op: "GetVolumePathName", Path: path, Err: err}
	}
	var buf [windows.MAX_LONG_PATH]uint16
	if err := windows.GetVolumePathName(&p[0], &buf[0], uint32(len(buf))); err != nil {
		return "", &fs.PathError{Op: "GetVolumePathName", Path: path, Err: err}
	}
	return windows.UTF16ToString(buf[:]), nil
}

func volumeInfo(root string) (label string, serial, flags uint32, err error) {
	p, err := windows.UTF16PtrFromString(root)
	if err != nil {
		return "", 0, 0, &fs.PathError{Op: "GetVolumeInformation", Path: root, Err: err}
	}
	var name [windows.MAX_PATH + 1]uint16
	if err := windows.GetVolumeInformation(p, &name[0], uint32(len(name)), &serial, nil, &flags, nil, 0); err != nil {
		return "", 0, 0, &fs.PathError{Op: "GetVolumeInformation", Path: root, Err: err}
	}
	return windows.UTF16ToString(name[:]), serial, flags, nil
}

func openDir(dir string, access uint32) error {
	p, err := windows.UTF16PtrFromString(longPath(dir))
	if err != nil {
		return err //nolint:wrapcheck // callers name the folder
	}
	h, err := windows.CreateFile(p, access, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
		nil, windows.OPEN_EXISTING, windows.FILE_FLAG_BACKUP_SEMANTICS, 0)
	if err != nil {
		return err //nolint:wrapcheck // callers name the folder
	}
	return windows.CloseHandle(h) //nolint:wrapcheck // callers name the folder
}

// longPath gives a path too long for a folder the \\?\ prefix the wide
// calls need to read it, as package os does for its own calls: 248, MAX_PATH
// less room for an 8.3 file name.
func longPath(p string) string {
	switch {
	case len(p) < 248 || strings.HasPrefix(p, `\\?\`) || strings.HasPrefix(p, `\\.\`):
		return p
	case strings.HasPrefix(p, `\\`):
		return `\\?\UNC\` + p[2:]
	}
	return `\\?\` + p
}
