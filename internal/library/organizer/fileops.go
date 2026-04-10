package organizer

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/slipstream/slipstream/internal/pathutil"
)

var (
	ErrHardlinkFailed = errors.New("failed to create hardlink")
	ErrSymlinkFailed  = errors.New("failed to create symlink")
	ErrCrossDevice    = errors.New("cross-device link not supported")
)

// LinkMode defines how files should be linked.
type LinkMode string

const (
	LinkModeHardlink LinkMode = "hardlink"
	LinkModeSymlink  LinkMode = "symlink"
	LinkModeCopy     LinkMode = "copy"
)

// CreateHardlink creates a hardlink from source to destination.
// Returns ErrCrossDevice if source and destination are on different filesystems.
func (s *Service) CreateHardlink(source, dest string) error {
	s.logger.Debug().
		Str("source", source).
		Str("dest", dest).
		Msg("Creating hardlink")

	if err := s.ensureDestDir(dest); err != nil {
		return err
	}

	// Remove existing file if present (overwrite behavior)
	if err := s.removeIfExists(dest); err != nil {
		return err
	}

	if err := os.Link(source, dest); err != nil {
		if isCrossDeviceError(err) {
			return fmt.Errorf("%w: %w", ErrCrossDevice, err)
		}
		return fmt.Errorf("%w: %w", ErrHardlinkFailed, err)
	}

	s.logger.Info().
		Str("source", source).
		Str("dest", dest).
		Msg("Created hardlink")
	return nil
}

// CreateSymlink creates a symlink from source to destination.
func (s *Service) CreateSymlink(source, dest string) error {
	s.logger.Debug().
		Str("source", source).
		Str("dest", dest).
		Msg("Creating symlink")

	if err := s.ensureDestDir(dest); err != nil {
		return err
	}

	if err := s.removeIfExists(dest); err != nil {
		return err
	}

	// Use absolute path for the symlink target
	absSource, err := filepath.Abs(source)
	if err != nil {
		return fmt.Errorf("%w: failed to resolve source path: %w", ErrSymlinkFailed, err)
	}

	if err := os.Symlink(absSource, dest); err != nil {
		return fmt.Errorf("%w: %w", ErrSymlinkFailed, err)
	}

	s.logger.Info().
		Str("source", source).
		Str("dest", dest).
		Msg("Created symlink")
	return nil
}

// LinkOrCopy attempts to create a hardlink, falls back to symlink, then copy.
// Returns the mode that was used and any error.
func (s *Service) LinkOrCopy(source, dest string) (LinkMode, error) {
	// Try hardlink first
	err := s.CreateHardlink(source, dest)
	if err == nil {
		return LinkModeHardlink, nil
	}

	// If cross-device, try symlink
	if errors.Is(err, ErrCrossDevice) {
		s.logger.Debug().Msg("Hardlink failed (cross-device), trying symlink")
		if err := s.CreateSymlink(source, dest); err == nil {
			return LinkModeSymlink, nil
		}
		s.logger.Debug().Err(err).Msg("Symlink failed, falling back to copy")
	} else {
		s.logger.Debug().Err(err).Msg("Hardlink failed, falling back to copy")
	}

	// Fall back to copy
	if err := s.CopyFile(source, dest); err != nil {
		return "", err
	}
	return LinkModeCopy, nil
}

// ImportFile imports a file to the library using hardlinks when possible.
// This is the primary method for importing downloaded files.
// - Source file is preserved (for torrent seeding)
// - Destination gets the renamed file via hardlink (same file, different name)
func (s *Service) ImportFile(source, dest string) (LinkMode, error) {
	return s.LinkOrCopy(source, dest)
}

// DeleteFile removes a file if it exists.
func (s *Service) DeleteFile(path string) error {
	s.logger.Debug().Str("path", path).Msg("Deleting file")

	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return nil // Already gone
		}
		return fmt.Errorf("failed to delete file: %w", err)
	}

	s.logger.Info().Str("path", path).Msg("Deleted file")
	return nil
}

// DeleteUpgradedFile deletes the old file when a new version is imported.
// Only deletes if the file exists and is different from the new file.
func (s *Service) DeleteUpgradedFile(oldPath, newPath string) error {
	if oldPath == "" || pathutil.PathsEqual(oldPath, newPath) {
		return nil
	}

	// Check if old file still exists
	if _, err := os.Stat(oldPath); os.IsNotExist(err) {
		return nil
	}

	s.logger.Info().
		Str("old", oldPath).
		Str("new", newPath).
		Msg("Deleting upgraded file")

	return s.DeleteFile(oldPath)
}

// ensureDestDir creates the destination directory if needed, inheriting permissions.
func (s *Service) ensureDestDir(destPath string) error {
	destDir := filepath.Dir(destPath)

	// Check if directory exists
	info, err := os.Stat(destDir)
	if err == nil && info.IsDir() {
		return nil
	}

	// Get parent directory permissions for inheritance
	parentDir := filepath.Dir(destDir)
	perm := os.FileMode(0o755) // Default

	if parentInfo, err := os.Stat(parentDir); err == nil {
		perm = parentInfo.Mode().Perm()
	}

	if err := os.MkdirAll(destDir, perm); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	return nil
}

// removeIfExists removes a file if it exists (for overwrite behavior).
func (s *Service) removeIfExists(path string) error {
	if _, err := os.Stat(path); err == nil {
		s.logger.Debug().Str("path", path).Msg("Removing existing file for overwrite")
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("failed to remove existing file: %w", err)
		}
	}
	return nil
}

// isCrossDeviceError checks if an error is a cross-device link error.
func isCrossDeviceError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	switch runtime.GOOS {
	case "linux", "darwin":
		// EXDEV: Cross-device link
		return strings.Contains(errStr, "cross-device") ||
			strings.Contains(errStr, "invalid cross-device link")
	case "windows":
		// ERROR_NOT_SAME_DEVICE
		return strings.Contains(errStr, "not on the same disk")
	default:
		return strings.Contains(errStr, "cross-device")
	}
}
