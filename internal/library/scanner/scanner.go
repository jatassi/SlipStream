package scanner

import (
	"context"
	"os"
	"path/filepath"

	"github.com/rs/zerolog"
)

// ScanError represents an error during scanning.
type ScanError struct {
	Path  string `json:"path"`
	Error string `json:"error"`
}

// ScanResult contains the results of scanning a root folder.
type ScanResult struct {
	RootFolderID int64         `json:"rootFolderId"`
	RootPath     string        `json:"rootPath"`
	Movies       []ParsedMedia `json:"movies"`
	Episodes     []ParsedMedia `json:"episodes"`
	Errors       []ScanError   `json:"errors"`
	TotalFiles   int           `json:"totalFiles"`
	Skipped      int           `json:"skipped"`
}

// ScanProgress is sent during scanning to report progress.
type ScanProgress struct {
	RootFolderID int64  `json:"rootFolderId"`
	CurrentPath  string `json:"currentPath"`
	FilesScanned int    `json:"filesScanned"`
	MoviesFound  int    `json:"moviesFound"`
	EpisodesFound int   `json:"episodesFound"`
}

// ProgressCallback is called during scanning to report progress.
type ProgressCallback func(progress ScanProgress)

// Service provides media file scanning operations.
type Service struct {
	logger zerolog.Logger
}

// NewService creates a new scanner service.
func NewService(logger zerolog.Logger) *Service {
	return &Service{
		logger: logger.With().Str("component", "scanner").Logger(),
	}
}

// ScanFolder scans a root folder for media files.
// mediaType should be "movie" or "tv".
func (s *Service) ScanFolder(ctx context.Context, folderPath string, mediaType string, progressCb ProgressCallback) (*ScanResult, error) {
	result := &ScanResult{
		RootPath: folderPath,
		Movies:   make([]ParsedMedia, 0),
		Episodes: make([]ParsedMedia, 0),
		Errors:   make([]ScanError, 0),
	}

	s.logger.Info().
		Str("path", folderPath).
		Str("mediaType", mediaType).
		Msg("Starting folder scan")

	// Walk the directory tree
	err := filepath.WalkDir(folderPath, func(path string, d os.DirEntry, err error) error {
		// Check for cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err != nil {
			result.Errors = append(result.Errors, ScanError{
				Path:  path,
				Error: err.Error(),
			})
			return nil // Continue scanning
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Check if it's a video file
		if !IsVideoFile(d.Name()) {
			return nil
		}

		// Skip sample files
		if IsSampleFile(d.Name()) {
			result.Skipped++
			return nil
		}

		result.TotalFiles++

		// Get file info for size
		info, err := d.Info()
		if err != nil {
			result.Errors = append(result.Errors, ScanError{
				Path:  path,
				Error: err.Error(),
			})
			return nil
		}

		// Parse the file
		parsed := ParsePath(path)
		parsed.FileSize = info.Size()

		// Categorize based on scan type or detection
		if mediaType == "movie" || (!parsed.IsTV && mediaType == "") {
			parsed.IsTV = false
			result.Movies = append(result.Movies, *parsed)
		} else if mediaType == "tv" || parsed.IsTV {
			parsed.IsTV = true
			result.Episodes = append(result.Episodes, *parsed)
		}

		// Report progress
		if progressCb != nil {
			progressCb(ScanProgress{
				RootFolderID:  result.RootFolderID,
				CurrentPath:   path,
				FilesScanned:  result.TotalFiles,
				MoviesFound:   len(result.Movies),
				EpisodesFound: len(result.Episodes),
			})
		}

		return nil
	})

	if err != nil {
		return result, err
	}

	s.logger.Info().
		Str("path", folderPath).
		Int("totalFiles", result.TotalFiles).
		Int("movies", len(result.Movies)).
		Int("episodes", len(result.Episodes)).
		Int("errors", len(result.Errors)).
		Int("skipped", result.Skipped).
		Msg("Folder scan completed")

	return result, nil
}

// ScanFile scans a single file and returns parsed information.
func (s *Service) ScanFile(filePath string) (*ParsedMedia, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}

	if info.IsDir() {
		return nil, os.ErrInvalid
	}

	parsed := ParsePath(filePath)
	parsed.FileSize = info.Size()
	return parsed, nil
}

// DetectMediaType attempts to detect if a folder contains movies or TV shows.
func (s *Service) DetectMediaType(folderPath string) (string, error) {
	movieCount := 0
	tvCount := 0

	err := filepath.WalkDir(folderPath, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}

		if !IsVideoFile(d.Name()) {
			return nil
		}

		parsed := ParseFilename(d.Name())
		if parsed.IsTV {
			tvCount++
		} else {
			movieCount++
		}

		// Sample enough files to make a decision
		if movieCount+tvCount >= 10 {
			return filepath.SkipAll
		}
		return nil
	})

	if err != nil && err != filepath.SkipAll {
		return "", err
	}

	if tvCount > movieCount {
		return "tv", nil
	}
	return "movie", nil
}
