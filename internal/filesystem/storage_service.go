package filesystem

import (
	"context"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/rs/zerolog"
	"github.com/slipstream/slipstream/internal/library/rootfolder"
)

// StorageService provides storage information integrated with root folders
type StorageService struct {
	fsService     *Service
	rootFolderSvc *rootfolder.Service
	logger        *zerolog.Logger
}

// NewStorageService creates a new storage service
func NewStorageService(fsService *Service, rootFolderSvc *rootfolder.Service, logger *zerolog.Logger) *StorageService {
	subLogger := logger.With().Str("component", "storage").Logger()
	return &StorageService{
		fsService:     fsService,
		rootFolderSvc: rootFolderSvc,
		logger:        &subLogger,
	}
}

// GetStorageInfo returns storage information for all drives/volumes with associated root folders
func (s *StorageService) GetStorageInfo(ctx context.Context) ([]StorageInfo, error) {
	// Get all root folders
	rootFolders, err := s.rootFolderSvc.List(ctx)
	if err != nil {
		return nil, err
	}

	// Convert to RootFolderRef with full paths
	rootFolderRefs := make([]RootFolderRef, len(rootFolders))
	for i, rf := range rootFolders {
		rootFolderRefs[i] = RootFolderRef{
			ID:        rf.ID,
			Name:      rf.Name,
			Path:      rf.Path,
			MediaType: rf.MediaType,
		}
	}

	// Get base storage information
	if runtime.GOOS == "windows" {
		return s.getWindowsStorageInfo(rootFolderRefs), nil
	}
	return s.getUnixStorageInfo(rootFolderRefs), nil
}

// getWindowsStorageInfo aggregates storage info for Windows drives
func (s *StorageService) getWindowsStorageInfo(rootFolders []RootFolderRef) []StorageInfo {
	var storage []StorageInfo
	drives := s.fsService.listDrives()

	for _, drive := range drives {
		// Calculate used space and percentage
		var usedSpace int64
		var usedPercent float64
		if drive.TotalSpace > 0 {
			usedSpace = drive.TotalSpace - drive.FreeSpace
			usedPercent = float64(usedSpace) / float64(drive.TotalSpace) * 100
		}

		// Find root folders on this drive
		var driveRootFolders []RootFolderRef
		for _, rf := range rootFolders {
			if s.isPathOnDrive(rf.Path, drive.Letter) {
				driveRootFolders = append(driveRootFolders, rf)
			}
		}

		// Determine label (use drive letter if no label)
		label := drive.Letter
		if drive.Label != "" {
			label = drive.Label
		}

		storage = append(storage, StorageInfo{
			Label:       label,
			Path:        drive.Letter,
			FreeSpace:   drive.FreeSpace,
			TotalSpace:  drive.TotalSpace,
			UsedSpace:   usedSpace,
			UsedPercent: usedPercent,
			Type:        drive.Type,
			RootFolders: driveRootFolders,
		})
	}

	return storage
}

// getUnixStorageInfo aggregates storage info for Unix volumes
func (s *StorageService) getUnixStorageInfo(rootFolders []RootFolderRef) []StorageInfo {
	var storage []StorageInfo
	volumes := s.fsService.ListVolumes()

	for _, volume := range volumes {
		// Calculate used space and percentage
		var usedSpace int64
		var usedPercent float64
		if volume.TotalSpace > 0 {
			usedSpace = volume.TotalSpace - volume.FreeSpace
			usedPercent = float64(usedSpace) / float64(volume.TotalSpace) * 100
		}

		// Find root folders on this volume
		var volumeRootFolders []RootFolderRef
		for _, rf := range rootFolders {
			if s.isPathOnVolume(rf.Path, volume.MountPoint) {
				volumeRootFolders = append(volumeRootFolders, rf)
			}
		}

		// Only include volume if it has root folders or is main boot volume
		if len(volumeRootFolders) == 0 && !s.isMainVolume(&volume) {
			continue
		}

		// Get proper volume label
		label := s.getVolumeLabel(&volume)

		storage = append(storage, StorageInfo{
			Label:       label,
			Path:        volume.MountPoint,
			FreeSpace:   volume.FreeSpace,
			TotalSpace:  volume.TotalSpace,
			UsedSpace:   usedSpace,
			UsedPercent: usedPercent,
			Type:        volume.Type,
			RootFolders: volumeRootFolders,
		})
	}

	return storage
}

// isPathOnDrive checks if a path is on the given Windows drive
func (s *StorageService) isPathOnDrive(path, driveLetter string) bool {
	// Normalize both paths to uppercase for comparison
	upperPath := strings.ToUpper(path)
	upperDrive := strings.ToUpper(driveLetter)

	return strings.HasPrefix(upperPath, upperDrive)
}

// isPathOnVolume checks if a path is on the given Unix volume
func (s *StorageService) isPathOnVolume(path, mountPoint string) bool {
	// Resolve relative paths to absolute paths
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}

	absMountPoint, err := filepath.Abs(mountPoint)
	if err != nil {
		return false
	}

	// Check if the path starts with the mount point
	return strings.HasPrefix(absPath, absMountPoint)
}

// isMainVolume checks if this is the main boot volume
func (s *StorageService) isMainVolume(volume *VolumeInfo) bool {
	// On macOS, the main volume is typically "/"
	if runtime.GOOS == "darwin" {
		return volume.MountPoint == "/"
	}

	// On other Unix, "/" is usually root but might not be main storage
	if runtime.GOOS != "windows" {
		return volume.MountPoint == "/"
	}

	// Windows: this shouldn't be called for Windows volumes
	return false
}

// getVolumeLabel gets a user-friendly volume label
func (s *StorageService) getVolumeLabel(volume *VolumeInfo) string {
	// On macOS, use special handling for proper volume names
	if runtime.GOOS == "darwin" {
		if volume.MountPoint == "/" {
			return "Macintosh HD" // Sensible default for main macOS volume
		}
	}

	// For other cases, use mount point base name
	if volume.MountPoint == "/" {
		return "Root"
	}
	return filepath.Base(volume.MountPoint)
}
