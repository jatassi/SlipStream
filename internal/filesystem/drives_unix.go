//go:build !windows

package filesystem

import (
	"context"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

const (
	osDarwin = "darwin"
)

// listDrives returns nil on Unix systems (no drive concept)
func (s *Service) listDrives() []DriveInfo {
	return nil
}

// ListVolumes returns a list of volumes/mount points on Unix systems
func (s *Service) ListVolumes() []VolumeInfo {
	var volumes []VolumeInfo

	if runtime.GOOS == osDarwin {
		volumes = s.listDarwinVolumes()
	} else {
		volumes = s.listLinuxVolumes()
	}

	return volumes
}

// listDarwinVolumes lists volumes on macOS
func (s *Service) listDarwinVolumes() []VolumeInfo {
	var volumes []VolumeInfo

	cmd := exec.CommandContext(context.Background(), "df", "-k")
	output, err := cmd.Output()
	if err != nil {
		s.logger.Warn().Err(err).Msg("Failed to get volume information")
		return volumes
	}

	lines := strings.Split(string(output), "\n")
	for i := 1; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}

		filesystem := fields[0]
		totalKb, _ := strconv.ParseInt(fields[1], 10, 64)
		usedKb, _ := strconv.ParseInt(fields[2], 10, 64)
		availKb, _ := strconv.ParseInt(fields[3], 10, 64)
		mountPoint := fields[len(fields)-1]

		if shouldSkipDarwinVolume(filesystem, mountPoint) {
			continue
		}

		totalBytes := totalKb * 1024
		freeBytes := availKb * 1024
		_ = usedKb

		volumeType := "local"
		if strings.HasPrefix(filesystem, "//") || strings.Contains(filesystem, "smb") || strings.Contains(filesystem, "nfs") {
			volumeType = "network"
		}

		volumes = append(volumes, VolumeInfo{
			Path:       mountPoint,
			MountPoint: mountPoint,
			Filesystem: filesystem,
			Type:       volumeType,
			FreeSpace:  freeBytes,
			TotalSpace: totalBytes,
		})
	}

	return volumes
}

// shouldSkipDarwinVolume determines if a volume should be skipped
func shouldSkipDarwinVolume(filesystem, mountPoint string) bool {
	systemVolumes := []string{
		"/System/Volumes/VM",
		"/System/Volumes/Preboot",
		"/System/Volumes/Update",
		"/System/Volumes/Recovery",
		"/System/Volumes/xarts",
		"/System/Volumes/iSCPreboot",
		"/System/Volumes/Hardware",
		"/dev",
		"/net",
		"/home",
	}

	for _, skip := range systemVolumes {
		if mountPoint == skip {
			return true
		}
	}

	if strings.HasPrefix(mountPoint, "/System/Volumes/") && mountPoint != "/System/Volumes/Data" {
		return true
	}

	if strings.HasPrefix(filesystem, "map") || strings.HasPrefix(filesystem, "autofs") {
		return true
	}

	if strings.Contains(filesystem, "tmpfs") || strings.Contains(filesystem, "devfs") {
		return true
	}

	return false
}

// listLinuxVolumes lists volumes on Linux
func (s *Service) listLinuxVolumes() []VolumeInfo {
	var volumes []VolumeInfo

	cmd := exec.CommandContext(context.Background(), "df", "-k", "--output=source,fstype,size,used,avail,target")
	output, err := cmd.Output()
	if err != nil {
		s.logger.Warn().Err(err).Msg("Failed to get volume information")
		return volumes
	}

	lines := strings.Split(string(output), "\n")
	for i := 1; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}

		filesystem := fields[0]
		fstype := fields[1]

		if shouldSkipLinuxFilesystem(fstype) {
			continue
		}

		totalKb, _ := strconv.ParseInt(fields[2], 10, 64)
		usedKb, _ := strconv.ParseInt(fields[3], 10, 64)
		availKb, _ := strconv.ParseInt(fields[4], 10, 64)
		mountPoint := strings.Join(fields[5:], " ")

		totalBytes := totalKb * 1024
		freeBytes := availKb * 1024
		_ = usedKb

		volumeType := "local"
		if strings.Contains(filesystem, "//") || strings.Contains(fstype, "nfs") || strings.Contains(fstype, "cifs") {
			volumeType = "network"
		}

		volumes = append(volumes, VolumeInfo{
			Path:       mountPoint,
			MountPoint: mountPoint,
			Filesystem: filesystem,
			Type:       volumeType,
			FreeSpace:  freeBytes,
			TotalSpace: totalBytes,
		})
	}

	return volumes
}

func shouldSkipLinuxFilesystem(fstype string) bool {
	return fstype == "tmpfs" || fstype == "devtmpfs" || fstype == "proc" || fstype == "sysfs"
}
