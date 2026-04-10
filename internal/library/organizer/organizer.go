package organizer

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/rs/zerolog"
)

// Service provides file organization operations.
type Service struct {
	logger *zerolog.Logger
}

// NewService creates a new organizer service.
func NewService(logger *zerolog.Logger) *Service {
	subLogger := logger.With().Str("component", "organizer").Logger()
	return &Service{
		logger: &subLogger,
	}
}

// CopyFile copies a file from source to destination.
func (s *Service) CopyFile(sourcePath, destPath string) error {
	return s.copyFile(sourcePath, destPath)
}

// copyFile performs the actual file copy.
func (s *Service) copyFile(sourcePath, destPath string) error {
	// Create destination directory if needed
	destDir := filepath.Dir(destPath)
	if err := os.MkdirAll(destDir, 0o750); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	source, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer source.Close()

	dest, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dest.Close()

	_, err = io.Copy(dest, source)
	if err != nil {
		os.Remove(destPath) // Clean up on failure
		return fmt.Errorf("failed to copy file: %w", err)
	}

	// Copy file permissions
	sourceInfo, err := os.Stat(sourcePath)
	if err == nil {
		if err := os.Chmod(destPath, sourceInfo.Mode()); err != nil {
			s.logger.Warn().Err(err).Str("path", destPath).Msg("Failed to set file permissions")
		}
	}

	return nil
}
