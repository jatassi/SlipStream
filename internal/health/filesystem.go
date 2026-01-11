package health

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

// FilesystemChecker provides filesystem health checks.
type FilesystemChecker struct{}

// NewFilesystemChecker creates a new filesystem checker.
func NewFilesystemChecker() *FilesystemChecker {
	return &FilesystemChecker{}
}

// CheckFolderAccessible verifies that a path exists and is a directory.
func (c *FilesystemChecker) CheckFolderAccessible(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("path does not exist: %s", path)
		}
		if os.IsPermission(err) {
			return fmt.Errorf("permission denied: %s", path)
		}
		return fmt.Errorf("cannot access path: %w", err)
	}

	if !info.IsDir() {
		return fmt.Errorf("path is not a directory: %s", path)
	}

	return nil
}

// CheckFolderWritable verifies that a directory is writable.
// This works cross-platform by attempting to create and delete a temp file.
func (c *FilesystemChecker) CheckFolderWritable(path string) error {
	// Generate unique temp file name
	tempFileName := fmt.Sprintf(".slipstream_health_check_%s", uuid.New().String()[:8])
	tempPath := filepath.Join(path, tempFileName)

	// Try to create the file
	file, err := os.Create(tempPath)
	if err != nil {
		if os.IsPermission(err) {
			return fmt.Errorf("folder is read-only: %s", path)
		}
		return fmt.Errorf("cannot write to folder: %w", err)
	}

	// Write some test data
	testData := []byte("health check")
	if _, err := file.Write(testData); err != nil {
		file.Close()
		os.Remove(tempPath)
		return fmt.Errorf("cannot write data: %w", err)
	}

	// Close the file
	if err := file.Close(); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("cannot close file: %w", err)
	}

	// Clean up by removing the temp file
	if err := os.Remove(tempPath); err != nil {
		return fmt.Errorf("cannot remove test file: %w", err)
	}

	return nil
}

// CheckFolderHealth combines accessibility and writability checks.
// Returns (ok, message) where message describes the issue if not ok.
func (c *FilesystemChecker) CheckFolderHealth(path string) (bool, string) {
	// Check accessibility
	if err := c.CheckFolderAccessible(path); err != nil {
		return false, err.Error()
	}

	// Check writability
	if err := c.CheckFolderWritable(path); err != nil {
		return false, err.Error()
	}

	return true, ""
}
