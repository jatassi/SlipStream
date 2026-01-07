package filesystem

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/rs/zerolog"
)

// Service provides filesystem browsing capabilities
type Service struct {
	logger zerolog.Logger
}

// NewService creates a new filesystem service
func NewService(logger zerolog.Logger) *Service {
	return &Service{
		logger: logger.With().Str("component", "filesystem").Logger(),
	}
}

// BrowseDirectory lists directories at the given path
// If path is empty, returns root directories (drives on Windows, / on Unix)
func (s *Service) BrowseDirectory(path string) (*BrowseResult, error) {
	// Handle empty path - return root
	if path == "" {
		return s.browseRoot()
	}

	// Validate and clean the path
	cleanPath, err := s.validatePath(path)
	if err != nil {
		return nil, err
	}

	// Read directory contents
	entries, err := os.ReadDir(cleanPath)
	if err != nil {
		if os.IsPermission(err) {
			return nil, ErrAccessDenied
		}
		return nil, ErrPathNotFound
	}

	// Filter to directories only and build result
	var dirEntries []DirectoryEntry
	for _, entry := range entries {
		if entry.IsDir() {
			// Skip hidden directories on Unix (starting with .)
			if runtime.GOOS != "windows" && strings.HasPrefix(entry.Name(), ".") {
				continue
			}
			// Skip system directories on Windows
			if runtime.GOOS == "windows" && isWindowsSystemDir(entry.Name()) {
				continue
			}

			dirEntries = append(dirEntries, DirectoryEntry{
				Name:  entry.Name(),
				Path:  filepath.Join(cleanPath, entry.Name()),
				IsDir: true,
			})
		}
	}

	// Sort directories alphabetically (case-insensitive)
	sort.Slice(dirEntries, func(i, j int) bool {
		return strings.ToLower(dirEntries[i].Name) < strings.ToLower(dirEntries[j].Name)
	})

	// Calculate parent path
	parent := filepath.Dir(cleanPath)
	if parent == cleanPath {
		// We're at root
		parent = ""
	}

	return &BrowseResult{
		Path:    cleanPath,
		Parent:  parent,
		Entries: dirEntries,
	}, nil
}

// browseRoot returns the root browsing result (drives on Windows, / on Unix)
func (s *Service) browseRoot() (*BrowseResult, error) {
	if runtime.GOOS == "windows" {
		drives := s.listDrives()
		return &BrowseResult{
			Path:   "",
			Drives: drives,
		}, nil
	}

	// Unix - browse /
	return s.BrowseDirectory("/")
}

// validatePath validates and cleans a filesystem path
func (s *Service) validatePath(path string) (string, error) {
	// Clean the path to resolve . and ..
	cleaned := filepath.Clean(path)

	// Convert to absolute path
	absPath, err := filepath.Abs(cleaned)
	if err != nil {
		return "", ErrInvalidPath
	}

	// Validate path format based on OS
	if runtime.GOOS == "windows" {
		if !isValidWindowsPath(absPath) {
			return "", ErrInvalidPath
		}
	} else {
		if !strings.HasPrefix(absPath, "/") {
			return "", ErrInvalidPath
		}
	}

	// Verify path exists and is a directory
	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", ErrPathNotFound
		}
		if os.IsPermission(err) {
			return "", ErrAccessDenied
		}
		return "", ErrInvalidPath
	}

	if !info.IsDir() {
		return "", ErrNotDirectory
	}

	return absPath, nil
}

// isValidWindowsPath checks if a path is a valid Windows path
func isValidWindowsPath(path string) bool {
	if len(path) < 3 {
		return false
	}

	// Check for drive letter: C:\
	if len(path) >= 3 && path[1] == ':' && (path[2] == '\\' || path[2] == '/') {
		letter := path[0]
		if (letter >= 'A' && letter <= 'Z') || (letter >= 'a' && letter <= 'z') {
			return true
		}
	}

	// Check for UNC path: \\server\share
	if strings.HasPrefix(path, "\\\\") {
		parts := strings.Split(path[2:], "\\")
		if len(parts) >= 2 && parts[0] != "" && parts[1] != "" {
			return true
		}
	}

	return false
}

// GetStorageInfo returns storage information for all drives/volumes
func (s *Service) GetStorageInfo(rootFolders []RootFolderRef) ([]StorageInfo, error) {
	var storage []StorageInfo

	if runtime.GOOS == "windows" {
		storage = s.getWindowsStorageInfo(rootFolders)
	} else {
		storage = s.getUnixStorageInfo(rootFolders)
	}

	return storage, nil
}

// getWindowsStorageInfo aggregates storage info for Windows drives
func (s *Service) getWindowsStorageInfo(rootFolders []RootFolderRef) []StorageInfo {
	var storage []StorageInfo
	drives := s.listDrives()

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
			if strings.HasPrefix(strings.ToUpper(rf.Name), drive.Letter) {
				// This is a simplified check - in reality we'd need the full path
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
func (s *Service) getUnixStorageInfo(rootFolders []RootFolderRef) []StorageInfo {
	var storage []StorageInfo
	volumes := s.ListVolumes()

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
			// This is a simplified check - in reality we'd need the full path
			if strings.HasPrefix(rf.Name, volume.MountPoint) {
				volumeRootFolders = append(volumeRootFolders, rf)
			}
		}

		// Use mount point as label if no better label available
		label := volume.MountPoint
		if label == "/" {
			label = "Root"
		} else {
			// Extract just the last component for cleaner display
			label = filepath.Base(volume.MountPoint)
		}

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

// isWindowsSystemDir checks if a directory name is a Windows system directory to hide
func isWindowsSystemDir(name string) bool {
	systemDirs := map[string]bool{
		"$Recycle.Bin":              true,
		"System Volume Information": true,
		"$WinREAgent":               true,
		"Config.Msi":                true,
	}
	return systemDirs[name]
}
