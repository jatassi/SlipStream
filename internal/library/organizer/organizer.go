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
	config NamingConfig
	logger *zerolog.Logger
}

// NewService creates a new organizer service.
func NewService(config *NamingConfig, logger *zerolog.Logger) *Service {
	subLogger := logger.With().Str("component", "organizer").Logger()
	return &Service{
		config: *config,
		logger: &subLogger,
	}
}

// GetConfig returns the current naming configuration.
func (s *Service) GetConfig() NamingConfig {
	return s.config
}

// SetConfig updates the naming configuration.
func (s *Service) SetConfig(config *NamingConfig) {
	s.config = *config
}

// GenerateMoviePath generates the full path for a movie.
func (s *Service) GenerateMoviePath(rootPath string, tokens *MovieTokens) string {
	folder := s.config.FormatMovieFolder(tokens)
	return filepath.Join(rootPath, folder)
}

// GenerateMovieFilename generates the filename for a movie (without extension).
func (s *Service) GenerateMovieFilename(tokens *MovieTokens) string {
	return s.config.FormatMovieFile(tokens)
}

// GenerateSeriesPath generates the path for a TV series.
func (s *Service) GenerateSeriesPath(rootPath string, tokens *SeriesTokens) string {
	folder := s.config.FormatSeriesFolder(tokens)
	return filepath.Join(rootPath, folder)
}

// GenerateSeasonPath generates the path for a season folder.
func (s *Service) GenerateSeasonPath(seriesPath string, seasonNumber int) string {
	folder := s.config.FormatSeasonFolder(seasonNumber)
	return filepath.Join(seriesPath, folder)
}

// GenerateEpisodeFilename generates the filename for an episode (without extension).
func (s *Service) GenerateEpisodeFilename(tokens *EpisodeTokens) string {
	return s.config.FormatEpisodeFile(tokens)
}

// MoveFile moves a file from source to destination, creating directories as needed.
func (s *Service) MoveFile(sourcePath, destPath string) error {
	s.logger.Debug().
		Str("source", sourcePath).
		Str("dest", destPath).
		Msg("Moving file")

	// Create destination directory if needed
	destDir := filepath.Dir(destPath)
	if err := os.MkdirAll(destDir, 0o750); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Try rename first (works if same filesystem)
	err := os.Rename(sourcePath, destPath)
	if err == nil {
		s.logger.Info().
			Str("source", sourcePath).
			Str("dest", destPath).
			Msg("Moved file")
		return nil
	}

	// Fall back to copy + delete for cross-filesystem moves
	if err := s.copyFile(sourcePath, destPath); err != nil {
		return err
	}

	if err := os.Remove(sourcePath); err != nil {
		s.logger.Warn().Err(err).Str("path", sourcePath).Msg("Failed to remove source file after copy")
	}

	s.logger.Info().
		Str("source", sourcePath).
		Str("dest", destPath).
		Msg("Moved file (copy + delete)")
	return nil
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

// RenameFile renames a file in place.
func (s *Service) RenameFile(oldPath, newFilename string) (string, error) {
	dir := filepath.Dir(oldPath)
	ext := filepath.Ext(oldPath)
	newPath := filepath.Join(dir, newFilename+ext)

	if oldPath == newPath {
		return newPath, nil // No change needed
	}

	if err := os.Rename(oldPath, newPath); err != nil {
		return "", fmt.Errorf("failed to rename file: %w", err)
	}

	s.logger.Info().
		Str("old", oldPath).
		Str("new", newPath).
		Msg("Renamed file")
	return newPath, nil
}

// OrganizeMovie moves and renames a movie file to the proper location.
func (s *Service) OrganizeMovie(sourcePath, rootPath string, tokens *MovieTokens) (string, error) {
	// Generate destination path
	movieDir := s.GenerateMoviePath(rootPath, tokens)
	filename := s.GenerateMovieFilename(tokens)
	ext := filepath.Ext(sourcePath)
	destPath := filepath.Join(movieDir, filename+ext)

	if sourcePath == destPath {
		return destPath, nil // Already in correct location
	}

	if err := s.MoveFile(sourcePath, destPath); err != nil {
		return "", err
	}

	return destPath, nil
}

// OrganizeEpisode moves and renames an episode file to the proper location.
func (s *Service) OrganizeEpisode(sourcePath, seriesPath string, seasonNumber int, tokens *EpisodeTokens) (string, error) {
	// Generate destination path
	seasonDir := s.GenerateSeasonPath(seriesPath, seasonNumber)
	filename := s.GenerateEpisodeFilename(tokens)
	ext := filepath.Ext(sourcePath)
	destPath := filepath.Join(seasonDir, filename+ext)

	if sourcePath == destPath {
		return destPath, nil // Already in correct location
	}

	if err := s.MoveFile(sourcePath, destPath); err != nil {
		return "", err
	}

	return destPath, nil
}

// CleanEmptyFolders removes empty folders in a directory tree.
func (s *Service) CleanEmptyFolders(rootPath string) error {
	return filepath.WalkDir(rootPath, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil //nolint:nilerr // Skip filesystem errors during walk
		}

		if !d.IsDir() || path == rootPath {
			return nil
		}

		// Check if directory is empty
		entries, readErr := os.ReadDir(path)
		if readErr != nil {
			return nil //nolint:nilerr // Skip unreadable directories
		}

		if len(entries) == 0 {
			if err := os.Remove(path); err == nil {
				s.logger.Debug().Str("path", path).Msg("Removed empty folder")
			}
		}

		return nil
	})
}

// PreviewMovieRename returns what the movie filename would be without making changes.
func (s *Service) PreviewMovieRename(tokens *MovieTokens) (folder, filename string) {
	folder = s.config.FormatMovieFolder(tokens)
	filename = s.config.FormatMovieFile(tokens)
	return
}

// PreviewEpisodeRename returns what the episode filename would be without making changes.
func (s *Service) PreviewEpisodeRename(tokens *EpisodeTokens, seasonNumber int) (seasonFolder, filename string) {
	seasonFolder = s.config.FormatSeasonFolder(seasonNumber)
	filename = s.config.FormatEpisodeFile(tokens)
	return
}
