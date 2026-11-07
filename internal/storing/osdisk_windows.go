package storing

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io/fs"
	"os/user"
	"strings"
	"syscall"
	"time"
	"unsafe"

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

// fileIDInfo is FILE_ID_INFO, which x/sys/windows leaves out.
type fileIDInfo struct {
	VolumeSerialNumber uint64
	FileID             [16]byte
}

// fileID is path's volume serial and 128-bit file id where its volume keeps
// ids stable: not on a network drive, whose ids are the server's, and not
// on FAT or exFAT, which derive the id from where the file's entry lies.
func fileID(path string, _ fs.FileInfo) ([]byte, error) {
	root, err := volumeRoot(path)
	if err != nil {
		return nil, err
	}
	if kindOfRoot(root) == Network {
		return nil, nil
	}
	p, err := windows.UTF16PtrFromString(longPath(path))
	if err != nil {
		return nil, &fs.PathError{Op: "CreateFile", Path: path, Err: err}
	}
	h, err := windows.CreateFile(p, 0, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
		nil, windows.OPEN_EXISTING, windows.FILE_FLAG_BACKUP_SEMANTICS, 0)
	if err != nil {
		return nil, &fs.PathError{Op: "CreateFile", Path: path, Err: err}
	}
	defer func() { _ = windows.CloseHandle(h) }()
	var name [windows.MAX_PATH + 1]uint16
	if err := windows.GetVolumeInformationByHandle(h, nil, 0, nil, nil, nil, &name[0], uint32(len(name))); err != nil {
		return nil, &fs.PathError{Op: "GetVolumeInformationByHandle", Path: path, Err: err}
	}
	if !stableFSName(windows.UTF16ToString(name[:])) {
		return nil, nil
	}
	var info fileIDInfo
	if err := windows.GetFileInformationByHandleEx(h, windows.FileIdInfo, (*byte)(unsafe.Pointer(&info)), uint32(unsafe.Sizeof(info))); err != nil { //nolint:gosec // G103: FILE_ID_INFO for a Win32 call
		return nil, &fs.PathError{Op: "GetFileInformationByHandleEx", Path: path, Err: err}
	}
	return append(binary.BigEndian.AppendUint64(nil, info.VolumeSerialNumber), info.FileID[:]...), nil
}

// stableFSName is a local filesystem whose file ids survive a remount:
// FAT, FAT32 and exFAT number a file by where its entry lies.
func stableFSName(name string) bool { return !strings.Contains(strings.ToUpper(name), "FAT") }

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

// ErrNoSpace is the error a write on a full volume fails with.
var ErrNoSpace error = windows.ERROR_DISK_FULL

// NoSpace reports whether err is a write that ran out of space.
func NoSpace(err error) bool {
	return errors.Is(err, windows.ERROR_DISK_FULL) || errors.Is(err, windows.ERROR_HANDLE_DISK_FULL)
}
