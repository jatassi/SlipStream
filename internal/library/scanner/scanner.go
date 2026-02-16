package scanner

import (
	"context"
	"errors"
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
	RootFolderID  int64  `json:"rootFolderId"`
	CurrentPath   string `json:"currentPath"`
	FilesScanned  int    `json:"filesScanned"`
	MoviesFound   int    `json:"moviesFound"`
	EpisodesFound int    `json:"episodesFound"`
}

// ProgressCallback is called during scanning to report progress.
type ProgressCallback func(progress ScanProgress)

// Service provides media file scanning operations.
type Service struct {
	logger *zerolog.Logger
}

// NewService creates a new scanner service.
func NewService(logger *zerolog.Logger) *Service {
	subLogger := logger.With().Str("component", "scanner").Logger()
	return &Service{
		logger: &subLogger,
	}
}

// ScanFolder scans a root folder for media files.
// mediaType should be "movie" or "tv".
func (s *Service) ScanFolder(ctx context.Context, folderPath, mediaType string, progressCb ProgressCallback) (*ScanResult, error) {
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

	err := filepath.WalkDir(folderPath, func(path string, d os.DirEntry, walkErr error) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		return s.processEntry(path, d, walkErr, mediaType, result, progressCb)
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

func (s *Service) processEntry(path string, d os.DirEntry, walkErr error, mediaType string, result *ScanResult, progressCb ProgressCallback) error {
	if walkErr != nil {
		result.Errors = append(result.Errors, ScanError{Path: path, Error: walkErr.Error()})
		return nil //nolint:nilerr // Record error but continue scanning
	}

	if d.IsDir() || !IsVideoFile(d.Name()) {
		return nil
	}

	if IsSampleFile(d.Name()) {
		result.Skipped++
		return nil
	}

	result.TotalFiles++

	info, infoErr := d.Info()
	if infoErr != nil {
		result.Errors = append(result.Errors, ScanError{Path: path, Error: infoErr.Error()})
		return nil //nolint:nilerr // Record error but continue scanning
	}

	parsed := ParsePath(path)
	parsed.FileSize = info.Size()
	categorizeMedia(parsed, mediaType, result)

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
}

func categorizeMedia(parsed *ParsedMedia, mediaType string, result *ScanResult) {
	if mediaType == "movie" || (!parsed.IsTV && mediaType == "") {
		parsed.IsTV = false
		result.Movies = append(result.Movies, *parsed)
	} else if mediaType == "tv" || parsed.IsTV {
		parsed.IsTV = true
		result.Episodes = append(result.Episodes, *parsed)
	}
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

	err := filepath.WalkDir(folderPath, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() {
			return nil //nolint:nilerr // Skip errors and directories during type detection
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

	if err != nil && !errors.Is(err, filepath.SkipAll) {
		return "", err
	}

	if tvCount > movieCount {
		return "tv", nil
	}
	return "movie", nil
}
