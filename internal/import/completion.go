package importer

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/slipstream/slipstream/internal/downloader/types"
)

// CompletionStatus indicates whether a file is ready for import.
type CompletionStatus string

const (
	CompletionReady      CompletionStatus = "ready"
	CompletionStillOpen  CompletionStatus = "still_open"
	CompletionRecent     CompletionStatus = "recent"
	CompletionNotFound   CompletionStatus = "not_found"
	CompletionTooSmall   CompletionStatus = "too_small"
	CompletionNotVideo   CompletionStatus = "not_video"
)

// FileCompletionResult contains the result of a file completion check.
type FileCompletionResult struct {
	Path       string           `json:"path"`
	Status     CompletionStatus `json:"status"`
	FileSize   int64            `json:"fileSize"`
	ModTime    time.Time        `json:"modTime"`
	SecondsSinceModify float64  `json:"secondsSinceModify"`
	Reason     string           `json:"reason,omitempty"`
}

// MinAgeSeconds is the minimum age in seconds since last modification.
const MinAgeSeconds = 60

// CheckFileCompletion checks if a file is complete and ready for import.
func (s *Service) CheckFileCompletion(ctx context.Context, path string) *FileCompletionResult {
	result := &FileCompletionResult{
		Path: path,
	}

	stat, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			result.Status = CompletionNotFound
			result.Reason = "file not found"
		} else {
			result.Status = CompletionNotFound
			result.Reason = err.Error()
		}
		return result
	}

	result.FileSize = stat.Size()
	result.ModTime = stat.ModTime()
	result.SecondsSinceModify = time.Since(stat.ModTime()).Seconds()

	// Load settings for validation thresholds
	settings, settingsErr := s.GetSettings(ctx)
	if settingsErr != nil {
		s.logger.Warn().Err(settingsErr).Msg("Failed to load import settings for completion check, using defaults")
		settings = nil
	}

	// Check extension
	ext := filepath.Ext(path)
	if settings != nil && len(settings.VideoExtensions) > 0 {
		if !settings.IsValidExtension(ext) {
			result.Status = CompletionNotVideo
			result.Reason = "not a video file"
			return result
		}
	} else if !IsValidVideoExtension(ext) {
		result.Status = CompletionNotVideo
		result.Reason = "not a video file"
		return result
	}

	// Check size (use settings if available)
	minSize := int64(MinFileSizeBytes)
	if settings != nil {
		minSize = settings.GetMinimumFileSizeBytes()
	}
	if stat.Size() < minSize {
		result.Status = CompletionTooSmall
		result.Reason = "file too small"
		return result
	}

	// Check if file was recently modified
	if result.SecondsSinceModify < MinAgeSeconds {
		result.Status = CompletionRecent
		result.Reason = "file was recently modified"
		return result
	}

	// Check if file is still open by trying to get an exclusive lock
	// This is a platform-specific check
	if isFileStillOpen(path) {
		result.Status = CompletionStillOpen
		result.Reason = "file appears to still be open"
		return result
	}

	result.Status = CompletionReady
	return result
}

// WaitForCompletion waits for a file to be ready for import.
func (s *Service) WaitForCompletion(ctx context.Context, path string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if time.Now().After(deadline) {
			return context.DeadlineExceeded
		}

		result := s.CheckFileCompletion(ctx, path)
		if result.Status == CompletionReady {
			return nil
		}

		// Wait before checking again
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Second):
		}
	}
}

// isFileStillOpen attempts to detect if a file is still being written to.
// This is a best-effort check that may not work on all platforms.
func isFileStillOpen(path string) bool {
	// Try to open the file exclusively
	// On most systems, this will fail if another process has the file open for writing

	file, err := os.OpenFile(path, os.O_RDWR|os.O_EXCL, 0)
	if err != nil {
		// Could indicate file is in use, but could also be permission issues
		// Check if it's specifically a "file in use" type error
		return false // Assume not open to avoid blocking imports
	}
	file.Close()
	return false
}

// DetectDownloadCompletion checks if a download folder is complete.
// This handles season packs and multi-file downloads.
type DownloadCompletionResult struct {
	Path           string                   `json:"path"`
	IsComplete     bool                     `json:"isComplete"`
	TotalFiles     int                      `json:"totalFiles"`
	ReadyFiles     int                      `json:"readyFiles"`
	PendingFiles   int                      `json:"pendingFiles"`
	FileResults    []*FileCompletionResult  `json:"fileResults,omitempty"`
}

// CheckDownloadCompletion checks if all files in a download folder are complete.
func (s *Service) CheckDownloadCompletion(ctx context.Context, downloadPath string) (*DownloadCompletionResult, error) {
	result := &DownloadCompletionResult{
		Path:        downloadPath,
		FileResults: make([]*FileCompletionResult, 0),
	}

	// Find all video files in the download
	files, err := s.findVideoFiles(downloadPath)
	if err != nil {
		return nil, err
	}

	result.TotalFiles = len(files)

	for _, file := range files {
		fileResult := s.CheckFileCompletion(ctx, file)
		result.FileResults = append(result.FileResults, fileResult)

		if fileResult.Status == CompletionReady {
			result.ReadyFiles++
		} else {
			result.PendingFiles++
		}
	}

	result.IsComplete = result.PendingFiles == 0 && result.ReadyFiles > 0

	return result, nil
}

// Archive detection extensions
var archiveExtensions = map[string]bool{
	".rar":  true,
	".zip":  true,
	".7z":   true,
	".tar":  true,
	".gz":   true,
	".bz2":  true,
	".xz":   true,
	".r00":  true, // RAR multi-part
	".r01":  true,
	".r02":  true,
	".part001.rar": true, // Newer RAR multi-part naming
	".part01.rar":  true,
}

// ArchiveAnalysis contains information about archives in a download.
type ArchiveAnalysis struct {
	Path             string   `json:"path"`
	HasArchives      bool     `json:"hasArchives"`
	HasVideoFiles    bool     `json:"hasVideoFiles"`
	ArchiveFiles     []string `json:"archiveFiles"`
	VideoFiles       []string `json:"videoFiles"`
	ExtractionNeeded bool     `json:"extractionNeeded"`
	ExtractionDone   bool     `json:"extractionDone"`
}

// AnalyzeForArchives scans a path for archive files and determines if extraction is needed.
func (s *Service) AnalyzeForArchives(ctx context.Context, downloadPath string) (*ArchiveAnalysis, error) {
	result := &ArchiveAnalysis{
		Path:         downloadPath,
		ArchiveFiles: make([]string, 0),
		VideoFiles:   make([]string, 0),
	}

	err := filepath.Walk(downloadPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		if info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))

		// Check for archive files
		if isArchiveExtension(path) {
			result.ArchiveFiles = append(result.ArchiveFiles, path)
			result.HasArchives = true
		}

		// Check for video files
		if validVideoExtensions[ext] {
			result.VideoFiles = append(result.VideoFiles, path)
			result.HasVideoFiles = true
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Determine if extraction is needed
	// Extraction is needed if we have archives but no video files
	result.ExtractionNeeded = result.HasArchives && !result.HasVideoFiles

	// Extraction is done if we have archives AND video files
	// (assumes extraction completed and extracted files are present)
	result.ExtractionDone = result.HasArchives && result.HasVideoFiles

	return result, nil
}

// isArchiveExtension checks if a file path indicates an archive file.
func isArchiveExtension(path string) bool {
	lower := strings.ToLower(path)

	// Check for multi-part RAR naming patterns
	if strings.Contains(lower, ".part") && strings.HasSuffix(lower, ".rar") {
		return true
	}

	// Check standard extensions
	ext := filepath.Ext(lower)
	return archiveExtensions[ext]
}

// WaitForExtraction waits for archive extraction to complete in a download folder.
// It monitors for the appearance of video files after detecting archives.
func (s *Service) WaitForExtraction(ctx context.Context, downloadPath string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	checkInterval := 10 * time.Second

	s.logger.Debug().
		Str("path", downloadPath).
		Dur("timeout", timeout).
		Msg("Waiting for extraction to complete")

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if time.Now().After(deadline) {
			return context.DeadlineExceeded
		}

		analysis, err := s.AnalyzeForArchives(ctx, downloadPath)
		if err != nil {
			return err
		}

		// If no archives, nothing to wait for
		if !analysis.HasArchives {
			s.logger.Debug().Str("path", downloadPath).Msg("No archives found, nothing to wait for")
			return nil
		}

		// If extraction is done (we have video files), we're good
		if analysis.ExtractionDone {
			s.logger.Debug().
				Str("path", downloadPath).
				Int("videoFiles", len(analysis.VideoFiles)).
				Msg("Extraction complete, video files found")
			return nil
		}

		// Wait before checking again
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(checkInterval):
		}
	}
}

// IsExtractionComplete checks if archive extraction is complete.
func (s *Service) IsExtractionComplete(ctx context.Context, downloadPath string) (bool, error) {
	analysis, err := s.AnalyzeForArchives(ctx, downloadPath)
	if err != nil {
		return false, err
	}

	// No archives = no extraction needed, so "complete"
	if !analysis.HasArchives {
		return true, nil
	}

	// Extraction complete if we have video files
	return analysis.ExtractionDone, nil
}

// NeedsExtractionWait checks if a download path needs to wait for extraction.
func (s *Service) NeedsExtractionWait(ctx context.Context, downloadPath string) (bool, error) {
	analysis, err := s.AnalyzeForArchives(ctx, downloadPath)
	if err != nil {
		return false, err
	}

	return analysis.ExtractionNeeded, nil
}

// GetExtractedVideoFiles returns video files that appear to be from extraction.
func (s *Service) GetExtractedVideoFiles(ctx context.Context, downloadPath string) ([]string, error) {
	analysis, err := s.AnalyzeForArchives(ctx, downloadPath)
	if err != nil {
		return nil, err
	}

	return analysis.VideoFiles, nil
}

// DownloadStatus represents the status of a download from the client's perspective.
type DownloadStatus string

const (
	DownloadStatusActive    DownloadStatus = "active"
	DownloadStatusComplete  DownloadStatus = "complete"
	DownloadStatusStalled   DownloadStatus = "stalled"
	DownloadStatusPaused    DownloadStatus = "paused"
	DownloadStatusError     DownloadStatus = "error"
	DownloadStatusNotFound  DownloadStatus = "not_found"
)

// DownloadStatusResult contains information about a download's status.
type DownloadStatusResult struct {
	ClientID     int64          `json:"clientId"`
	DownloadID   string         `json:"downloadId"`
	Status       DownloadStatus `json:"status"`
	Progress     float64        `json:"progress"`
	IsComplete   bool           `json:"isComplete"`
	IsStalled    bool           `json:"isStalled"`
	CanImport    bool           `json:"canImport"`
	Reason       string         `json:"reason,omitempty"`
}

// CheckDownloadStatus checks the status of a download via the download client.
func (s *Service) CheckDownloadStatus(ctx context.Context, clientID int64, downloadID string) (*DownloadStatusResult, error) {
	result := &DownloadStatusResult{
		ClientID:   clientID,
		DownloadID: downloadID,
	}

	client, err := s.downloader.GetClient(ctx, clientID)
	if err != nil {
		result.Status = DownloadStatusError
		result.Reason = "failed to get download client"
		return result, nil
	}

	items, err := client.List(ctx)
	if err != nil {
		result.Status = DownloadStatusError
		result.Reason = "failed to list downloads"
		return result, nil
	}

	for _, item := range items {
		if item.ID == downloadID {
			result.Progress = item.Progress

			switch item.Status {
			case types.StatusCompleted, types.StatusSeeding:
				result.Status = DownloadStatusComplete
				result.IsComplete = true
				result.CanImport = true
			case types.StatusPaused:
				result.Status = DownloadStatusPaused
				result.IsStalled = true
				result.Reason = "download is paused"
			case types.StatusError:
				result.Status = DownloadStatusError
				result.IsStalled = true
				result.Reason = "download has an error"
			case types.StatusDownloading:
				result.Status = DownloadStatusActive
			case types.StatusQueued:
				result.Status = DownloadStatusActive
				result.Reason = "download is queued"
			default:
				result.Status = DownloadStatusActive
			}

			return result, nil
		}
	}

	result.Status = DownloadStatusNotFound
	result.Reason = "download not found in client"
	return result, nil
}

// IsDownloadReadyForImport checks if a download is complete and ready for import.
// Returns false if the download is stalled, paused, or has errors.
func (s *Service) IsDownloadReadyForImport(ctx context.Context, clientID int64, downloadID string) (bool, string) {
	status, err := s.CheckDownloadStatus(ctx, clientID, downloadID)
	if err != nil {
		return false, "failed to check download status"
	}

	if status.IsStalled {
		return false, status.Reason
	}

	return status.CanImport, ""
}

// ShouldSkipStalledDownload determines if a download should be skipped due to stalled status.
func (s *Service) ShouldSkipStalledDownload(ctx context.Context, clientID int64, downloadID string) bool {
	status, err := s.CheckDownloadStatus(ctx, clientID, downloadID)
	if err != nil {
		return false // Don't skip if we can't determine status
	}

	return status.IsStalled
}

// MonitorForCompletion monitors a path and triggers import when complete.
func (s *Service) MonitorForCompletion(ctx context.Context, path string, callback func(string)) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			stat, err := os.Stat(path)
			if err != nil {
				continue
			}

			if stat.IsDir() {
				// Check download folder
				result, err := s.CheckDownloadCompletion(ctx, path)
				if err != nil {
					continue
				}

				if result.IsComplete {
					callback(path)
					return
				}
			} else {
				// Check single file
				result := s.CheckFileCompletion(ctx, path)
				if result.Status == CompletionReady {
					callback(path)
					return
				}
			}
		}
	}
}
