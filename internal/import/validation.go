package importer

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	// MinFileSizeMB is the minimum file size to consider valid (default 50MB).
	MinFileSizeMB = 50
	// MinFileSizeBytes is the minimum size in bytes.
	MinFileSizeBytes = MinFileSizeMB * 1024 * 1024
)

// validVideoExtensions are accepted video file extensions.
var validVideoExtensions = map[string]bool{
	".mkv":  true,
	".mp4":  true,
	".avi":  true,
	".m4v":  true,
	".mov":  true,
	".wmv":  true,
	".ts":   true,
	".m2ts": true,
	".webm": true,
	".flv":  true,
	".ogm":  true,
	".divx": true,
	".vob":  true,
}

// samplePatterns are patterns that indicate a sample file.
var samplePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)[-._]sample[-._]`),
	regexp.MustCompile(`(?i)^sample[-._]`),
	regexp.MustCompile(`(?i)[-._]sample$`),
	regexp.MustCompile(`(?i)/samples?/`),
}

// isValidExtensionWithSettings checks whether an extension is valid,
// preferring custom settings when available.
func isValidExtensionWithSettings(ext string, settings *ImportSettings) bool {
	if settings != nil && len(settings.VideoExtensions) > 0 {
		return settings.IsValidExtension(ext)
	}
	return validVideoExtensions[strings.ToLower(ext)]
}

// minimumFileSizeBytes returns the minimum file size from settings or the default.
func minimumFileSizeBytes(settings *ImportSettings) int64 {
	if settings != nil {
		return settings.GetMinimumFileSizeBytes()
	}
	return int64(MinFileSizeBytes)
}

// validateFile validates a file for import.
func (s *Service) validateFile(_ context.Context, path string, settings *ImportSettings) error {
	stat, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return ErrFileNotFound
		}
		return err
	}

	if stat.IsDir() {
		return ErrFileNotFound
	}

	ext := strings.ToLower(filepath.Ext(path))
	if !isValidExtensionWithSettings(ext, settings) {
		return ErrInvalidExtension
	}

	if stat.Size() < minimumFileSizeBytes(settings) {
		return ErrFileTooSmall
	}

	if isSampleFile(path) {
		return ErrSampleFile
	}

	return nil
}

// ValidateForImport validates a file and returns detailed validation result.
func (s *Service) ValidateForImport(ctx context.Context, path string) (*ValidationResult, error) {
	result := &ValidationResult{
		Path:  path,
		Valid: false,
	}

	stat, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			result.Reason = "file not found"
			return result, nil
		}
		return nil, err
	}

	result.FileSize = stat.Size()

	if stat.IsDir() {
		result.Reason = "path is a directory"
		return result, nil
	}

	settings, err := s.GetSettings(ctx)
	if err != nil {
		s.logger.Warn().Err(err).Msg("Failed to load import settings, using defaults")
		settings = nil
	}

	ext := strings.ToLower(filepath.Ext(path))
	result.Extension = ext
	if !isValidExtensionWithSettings(ext, settings) {
		result.Reason = "invalid file extension"
		return result, nil
	}

	if stat.Size() < minimumFileSizeBytes(settings) {
		result.Reason = "file too small"
		return result, nil
	}

	if isSampleFile(path) {
		result.Reason = "file appears to be a sample"
		result.IsSample = true
		return result, nil
	}

	result.Valid = true
	return result, nil
}

// ValidationResult contains detailed validation information.
type ValidationResult struct {
	Path      string `json:"path"`
	Valid     bool   `json:"valid"`
	Reason    string `json:"reason,omitempty"`
	Extension string `json:"extension,omitempty"`
	FileSize  int64  `json:"fileSize"`
	IsSample  bool   `json:"isSample,omitempty"`
}

// isSampleFile checks if a file path indicates a sample file.
func isSampleFile(path string) bool {
	for _, pattern := range samplePatterns {
		if pattern.MatchString(path) {
			return true
		}
	}
	return false
}

// IsValidVideoExtension checks if an extension is valid for import.
func IsValidVideoExtension(ext string) bool {
	return validVideoExtensions[strings.ToLower(ext)]
}

// GetValidExtensions returns a list of valid video extensions.
func GetValidExtensions() []string {
	exts := make([]string, 0, len(validVideoExtensions))
	for ext := range validVideoExtensions {
		exts = append(exts, ext)
	}
	return exts
}

// MediaInfoValidationResult contains the result of MediaInfo validation.
type MediaInfoValidationResult struct {
	Valid           bool    `json:"valid"`
	HasVideoStream  bool    `json:"hasVideoStream"`
	HasAudioStream  bool    `json:"hasAudioStream"`
	VideoCodec      string  `json:"videoCodec,omitempty"`
	AudioCodec      string  `json:"audioCodec,omitempty"`
	DurationSeconds float64 `json:"durationSeconds,omitempty"`
	Resolution      string  `json:"resolution,omitempty"`
	ContainerFormat string  `json:"containerFormat,omitempty"`
	Reason          string  `json:"reason,omitempty"`
}

// ValidateWithMediaInfo performs full validation using MediaInfo probe.
// This is used when ValidationLevel is set to "full".
func (s *Service) ValidateWithMediaInfo(ctx context.Context, path string) (*MediaInfoValidationResult, error) {
	result := &MediaInfoValidationResult{
		Valid: false,
	}

	// Check if MediaInfo is available
	if s.mediainfo == nil || !s.mediainfo.IsAvailable() {
		result.Reason = "MediaInfo probe tool not available"
		return result, ErrNoProbeToolAvailable
	}

	// Probe the file
	info, err := s.mediainfo.Probe(ctx, path)
	if err != nil {
		result.Reason = "MediaInfo probe failed: " + err.Error()
		return result, err
	}

	// Check for video stream
	if info.VideoCodec != "" {
		result.HasVideoStream = true
		result.VideoCodec = info.VideoCodec
	}

	// Check for audio stream
	if info.AudioCodec != "" {
		result.HasAudioStream = true
		result.AudioCodec = info.AudioCodec
	}

	// Extract other info
	result.Resolution = info.VideoResolution
	result.DurationSeconds = info.Duration.Seconds()
	result.ContainerFormat = info.ContainerFormat

	// Validation checks
	if !result.HasVideoStream {
		result.Reason = "file has no video stream"
		return result, nil
	}

	// A video file must have duration > 0 (at least 1 second)
	if result.DurationSeconds <= 0 {
		result.Reason = "file has invalid duration"
		return result, nil
	}

	// File is valid
	result.Valid = true
	return result, nil
}
