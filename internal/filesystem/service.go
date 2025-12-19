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

// isWindowsSystemDir checks if a directory name is a Windows system directory to hide
func isWindowsSystemDir(name string) bool {
	systemDirs := map[string]bool{
		"$Recycle.Bin":         true,
		"System Volume Information": true,
		"$WinREAgent":          true,
		"Config.Msi":           true,
	}
	return systemDirs[name]
}
