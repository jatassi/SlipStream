//go:build !windows

package filesystem

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// listDrives returns nil on Unix systems (no drive concept)
func (s *Service) listDrives() []DriveInfo {
	return nil
}

// ListVolumes returns a list of volumes/mount points on Unix systems
func (s *Service) ListVolumes() []VolumeInfo {
	var volumes []VolumeInfo

	if runtime.GOOS == "darwin" {
		volumes = s.listDarwinVolumes()
	} else {
		volumes = s.listLinuxVolumes()
	}

	return volumes
}

// listDarwinVolumes lists volumes on macOS
func (s *Service) listDarwinVolumes() []VolumeInfo {
	var volumes []VolumeInfo

	// Use df to get volume information
	cmd := exec.Command("df", "-k")
	output, err := cmd.Output()
	if err != nil {
		s.logger.Warn().Err(err).Msg("Failed to get volume information")
		return volumes
	}

	lines := strings.Split(string(output), "\n")
	for i := 1; i < len(lines); i++ { // Skip header
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

		// Filter out system volumes that shouldn't be shown
		if shouldSkipDarwinVolume(filesystem, mountPoint) {
			continue
		}

		totalBytes := totalKb * 1024
		freeBytes := availKb * 1024
		_ = usedKb // Suppress unused warning

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

// getDarwinVolumeLabels gets volume labels using diskutil
func (s *Service) getDarwinVolumeLabels() map[string]string {
	labels := make(map[string]string)

	_ = labels // Suppress unused warning
	return labels
	return labels
}

// getDarwinVolumeLabel gets the proper volume label for a filesystem
func (s *Service) getDarwinVolumeLabel(filesystem, mountPoint string, labels map[string]string) string {
	// For root filesystem, try to get the actual volume name
	if mountPoint == "/" {
		// Try to get volume name from system profiler
		cmd := exec.Command("system_profiler", "SPStorageDataType", "-json")
		output, err := cmd.Output()
		if err == nil {
			// Simple parsing for volume name - look for "Macintosh HD" or similar
			if strings.Contains(string(output), "Macintosh HD") {
				return "Macintosh HD"
			}
			// Look for volume name pattern
			lines := strings.Split(string(output), "\n")
			for _, line := range lines {
				if strings.Contains(line, "Volume Name") && strings.Contains(line, "Macintosh") {
					// Extract the volume name
					parts := strings.Split(line, ":")
					if len(parts) > 1 {
						name := strings.TrimSpace(strings.Trim(parts[1], `",`))
						if name != "" {
							return name
						}
					}
				}
			}
		}

		// Fallback: try diskutil info for the root device
		if strings.HasPrefix(filesystem, "/dev/") {
			device := strings.TrimPrefix(filesystem, "/dev/")
			cmd := exec.Command("diskutil", "info", device)
			output, err := cmd.Output()
			if err == nil {
				lines := strings.Split(string(output), "\n")
				for _, line := range lines {
					line = strings.TrimSpace(line)
					if strings.HasPrefix(line, "Volume Name:") {
						parts := strings.SplitN(line, ":", 2)
						if len(parts) > 1 {
							name := strings.TrimSpace(parts[1])
							if name != "" {
								return name
							}
						}
					}
				}
			}
		}

		return "Macintosh HD" // Sensible default
	}

	// For other mount points, use the last component
	if mountPoint == "/" {
		return "Macintosh HD"
	}
	return filepath.Base(mountPoint)
}

// shouldSkipDarwinVolume determines if a volume should be skipped
func shouldSkipDarwinVolume(filesystem, mountPoint string) bool {
	// Skip system volumes that aren't relevant for media storage
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

	// Skip APFS snapshots and system volumes
	if strings.HasPrefix(mountPoint, "/System/Volumes/") && mountPoint != "/System/Volumes/Data" {
		return true
	}

	// Skip small/ephemeral filesystems
	if strings.HasPrefix(filesystem, "map") || strings.HasPrefix(filesystem, "autofs") {
		return true
	}

	// Skip tmpfs and other virtual filesystems
	if strings.Contains(filesystem, "tmpfs") || strings.Contains(filesystem, "devfs") {
		return true
	}

	return false
}

// listLinuxVolumes lists volumes on Linux
func (s *Service) listLinuxVolumes() []VolumeInfo {
	var volumes []VolumeInfo

	// Use df to get volume information
	cmd := exec.Command("df", "-k", "--output=source,fstype,size,used,avail,target")
	output, err := cmd.Output()
	if err != nil {
		s.logger.Warn().Err(err).Msg("Failed to get volume information")
		return volumes
	}

	lines := strings.Split(string(output), "\n")
	for i := 1; i < len(lines); i++ { // Skip header
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
		totalKb, _ := strconv.ParseInt(fields[2], 10, 64)
		usedKb, _ := strconv.ParseInt(fields[3], 10, 64)
		availKb, _ := strconv.ParseInt(fields[4], 10, 64)
		mountPoint := strings.Join(fields[5:], " ") // Handle mount points with spaces

		// Skip pseudo-filesystems
		if fstype == "tmpfs" || fstype == "devtmpfs" || fstype == "proc" || fstype == "sysfs" {
			continue
		}

		totalBytes := totalKb * 1024
		freeBytes := availKb * 1024
		_ = usedKb // Suppress unused warning

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
