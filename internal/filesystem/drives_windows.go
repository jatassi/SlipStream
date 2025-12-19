//go:build windows

package filesystem

import (
	"syscall"
	"unsafe"
)

// listDrives returns a list of available drives on Windows
func (s *Service) listDrives() []DriveInfo {
	var drives []DriveInfo

	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	getLogicalDrives := kernel32.NewProc("GetLogicalDrives")
	getDriveType := kernel32.NewProc("GetDriveTypeW")

	// Get bitmask of available drives
	mask, _, _ := getLogicalDrives.Call()

	for i := 0; i < 26; i++ {
		if mask&(1<<uint(i)) != 0 {
			letter := string(rune('A' + i))
			drivePath := letter + ":\\"

			// Get drive type
			driveType := getDriveTypeValue(getDriveType, drivePath)

			// Skip CD-ROM and unknown drives for browsing
			if driveType == "cdrom" || driveType == "unknown" {
				continue
			}

			drive := DriveInfo{
				Letter: letter + ":",
				Type:   driveType,
			}

			// Try to get free space
			freeSpace := getDiskFreeSpace(drivePath)
			if freeSpace > 0 {
				drive.FreeSpace = freeSpace
			}

			drives = append(drives, drive)
		}
	}

	return drives
}

// getDriveTypeValue returns the type of drive as a string
func getDriveTypeValue(proc *syscall.LazyProc, path string) string {
	pathPtr, _ := syscall.UTF16PtrFromString(path)
	ret, _, _ := proc.Call(uintptr(unsafe.Pointer(pathPtr)))

	switch ret {
	case 2:
		return "removable"
	case 3:
		return "fixed"
	case 4:
		return "network"
	case 5:
		return "cdrom"
	case 6:
		return "ramdisk"
	default:
		return "unknown"
	}
}

// getDiskFreeSpace returns free space in bytes for a drive
func getDiskFreeSpace(path string) int64 {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	getDiskFreeSpaceEx := kernel32.NewProc("GetDiskFreeSpaceExW")

	pathPtr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return 0
	}

	var freeBytesAvailable, totalBytes, totalFreeBytes uint64
	ret, _, _ := getDiskFreeSpaceEx.Call(
		uintptr(unsafe.Pointer(pathPtr)),
		uintptr(unsafe.Pointer(&freeBytesAvailable)),
		uintptr(unsafe.Pointer(&totalBytes)),
		uintptr(unsafe.Pointer(&totalFreeBytes)),
	)

	if ret == 0 {
		return 0
	}

	return int64(freeBytesAvailable)
}
