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

// validateFile validates a file for import.
func (s *Service) validateFile(ctx context.Context, path string, settings *ImportSettings) error {
	// Check if file exists
	stat, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return ErrFileNotFound
		}
		return err
	}

	// Check if it's a regular file (not a directory)
	if stat.IsDir() {
		return ErrFileNotFound
	}

	// Check file extension
	ext := strings.ToLower(filepath.Ext(path))
	if settings != nil && len(settings.VideoExtensions) > 0 {
		if !settings.IsValidExtension(ext) {
			return ErrInvalidExtension
		}
	} else if !validVideoExtensions[ext] {
		return ErrInvalidExtension
	}

	// Check minimum file size (use settings if provided, otherwise use default)
	minSize := int64(MinFileSizeBytes)
	if settings != nil {
		minSize = settings.GetMinimumFileSizeBytes()
	}
	if stat.Size() < minSize {
		return ErrFileTooSmall
	}

	// Check for sample file patterns
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

	// Check if it's a directory
	if stat.IsDir() {
		result.Reason = "path is a directory"
		return result, nil
	}

	// Load settings for validation thresholds
	settings, err := s.GetSettings(ctx)
	if err != nil {
		s.logger.Warn().Err(err).Msg("Failed to load import settings, using defaults")
		settings = nil
	}

	// Check extension
	ext := strings.ToLower(filepath.Ext(path))
	if settings != nil && len(settings.VideoExtensions) > 0 {
		if !settings.IsValidExtension(ext) {
			result.Reason = "invalid file extension"
			result.Extension = ext
			return result, nil
		}
	} else if !validVideoExtensions[ext] {
		result.Reason = "invalid file extension"
		result.Extension = ext
		return result, nil
	}
	result.Extension = ext

	// Check minimum size (use settings if available)
	minSize := int64(MinFileSizeBytes)
	if settings != nil {
		minSize = settings.GetMinimumFileSizeBytes()
	}
	if stat.Size() < minSize {
		result.Reason = "file too small"
		return result, nil
	}

	// Check for sample
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
	Valid            bool    `json:"valid"`
	HasVideoStream   bool    `json:"hasVideoStream"`
	HasAudioStream   bool    `json:"hasAudioStream"`
	VideoCodec       string  `json:"videoCodec,omitempty"`
	AudioCodec       string  `json:"audioCodec,omitempty"`
	DurationSeconds  float64 `json:"durationSeconds,omitempty"`
	Resolution       string  `json:"resolution,omitempty"`
	ContainerFormat  string  `json:"containerFormat,omitempty"`
	Reason           string  `json:"reason,omitempty"`
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

// validateFileWithLevel validates a file according to the specified validation level.
func (s *Service) validateFileWithLevel(ctx context.Context, path string, settings *ImportSettings) error {
	if settings == nil {
		return s.validateFile(ctx, path, nil)
	}

	switch settings.ValidationLevel {
	case ValidationBasic:
		// Just check file exists and size > 0
		stat, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				return ErrFileNotFound
			}
			return err
		}
		if stat.Size() == 0 {
			return ErrFileTooSmall
		}
		return nil

	case ValidationStandard:
		// Standard validation (size, extension, sample check)
		return s.validateFile(ctx, path, settings)

	case ValidationFull:
		// Standard validation first
		if err := s.validateFile(ctx, path, settings); err != nil {
			return err
		}

		// Then MediaInfo probe validation
		result, err := s.ValidateWithMediaInfo(ctx, path)
		if err != nil {
			// If probe tool not available, log warning and continue
			if err == ErrNoProbeToolAvailable {
				s.logger.Warn().Str("path", path).Msg("MediaInfo not available for full validation, skipping probe")
				return nil
			}
			return err
		}

		if !result.Valid {
			s.logger.Warn().Str("path", path).Str("reason", result.Reason).Msg("MediaInfo validation failed")
			return ErrInvalidExtension // Using existing error for now
		}

		return nil

	default:
		// Default to standard
		return s.validateFile(ctx, path, settings)
	}
}
