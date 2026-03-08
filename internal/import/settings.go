//nolint:revive // package name mirrors domain concept
package importer

import (
	"context"
	"strings"

	"github.com/slipstream/slipstream/internal/database/sqlc"
)

// ValidationLevel defines how strictly files are validated before import.
type ValidationLevel string

const (
	ValidationBasic    ValidationLevel = "basic"    // File exists and size > 0
	ValidationStandard ValidationLevel = "standard" // Size > minimum, valid extension
	ValidationFull     ValidationLevel = "full"     // Size + extension + MediaInfo probe
)

// MatchConflictBehavior defines how to handle conflicts between queue and parse matching.
type MatchConflictBehavior string

const (
	MatchTrustQueue MatchConflictBehavior = "trust_queue" // Trust queue record
	MatchTrustParse MatchConflictBehavior = "trust_parse" // Trust filename parse
	MatchFail       MatchConflictBehavior = "fail"        // Fail with warning
)

// UnknownMediaBehavior defines how to handle files that don't match library items.
type UnknownMediaBehavior string

const (
	UnknownIgnore  UnknownMediaBehavior = "ignore"   // Reject unmatched files
	UnknownAutoAdd UnknownMediaBehavior = "auto_add" // Auto-add to library
)

// ImportSettings contains all import configuration.
type ImportSettings struct {
	// Validation settings
	ValidationLevel   ValidationLevel `json:"validationLevel"`
	MinimumFileSizeMB int             `json:"minimumFileSizeMb"`
	VideoExtensions   []string        `json:"videoExtensions"`

	// Matching settings
	MatchConflictBehavior MatchConflictBehavior `json:"matchConflictBehavior"`
	UnknownMediaBehavior  UnknownMediaBehavior  `json:"unknownMediaBehavior"`
}

// DefaultImportSettings returns the default import settings.
func DefaultImportSettings() ImportSettings {
	return ImportSettings{
		ValidationLevel:   ValidationStandard,
		MinimumFileSizeMB: 100,
		VideoExtensions:   []string{".mkv", ".mp4", ".avi", ".m4v", ".mov", ".wmv", ".ts", ".m2ts", ".webm"},

		MatchConflictBehavior: MatchTrustQueue,
		UnknownMediaBehavior:  UnknownIgnore,
	}
}

// SettingsFromDB converts a sqlc.ImportSetting to ImportSettings.
func SettingsFromDB(db *sqlc.ImportSetting) ImportSettings {
	extensions := strings.Split(db.VideoExtensions, ",")
	for i := range extensions {
		extensions[i] = strings.TrimSpace(extensions[i])
	}

	return ImportSettings{
		ValidationLevel:   ValidationLevel(db.ValidationLevel),
		MinimumFileSizeMB: int(db.MinimumFileSizeMb),
		VideoExtensions:   extensions,

		MatchConflictBehavior: MatchConflictBehavior(db.MatchConflictBehavior),
		UnknownMediaBehavior:  UnknownMediaBehavior(db.UnknownMediaBehavior),
	}
}

// IsValidExtension checks if a file extension is allowed.
func (s *ImportSettings) IsValidExtension(ext string) bool {
	ext = strings.ToLower(ext)
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	for _, allowed := range s.VideoExtensions {
		if strings.EqualFold(allowed, ext) {
			return true
		}
	}
	return false
}

// GetMinimumFileSizeBytes returns the minimum file size in bytes.
func (s *ImportSettings) GetMinimumFileSizeBytes() int64 {
	return int64(s.MinimumFileSizeMB) * 1024 * 1024
}

// GetSettings retrieves import settings from the database.
func (s *Service) GetSettings(ctx context.Context) (*ImportSettings, error) {
	// Ensure settings row exists
	if err := s.queries.EnsureImportSettingsExist(ctx); err != nil {
		return nil, err
	}

	dbSettings, err := s.queries.GetImportSettings(ctx)
	if err != nil {
		return nil, err
	}

	settings := SettingsFromDB(dbSettings)
	return &settings, nil
}

// loadSettingsOrNil loads settings, returning nil on error.
func (s *Service) loadSettingsOrNil(ctx context.Context) *ImportSettings {
	settings, err := s.GetSettings(ctx)
	if err != nil {
		s.logger.Warn().Err(err).Msg("Failed to load import settings, using defaults")
		return nil
	}
	return settings
}

// UpdateSettings updates import settings in the database.
func (s *Service) UpdateSettings(ctx context.Context, settings *ImportSettings) (*ImportSettings, error) {
	extensionsStr := strings.Join(settings.VideoExtensions, ",")

	params := sqlc.UpdateImportSettingsParams{
		ValidationLevel:       string(settings.ValidationLevel),
		MinimumFileSizeMb:     int64(settings.MinimumFileSizeMB),
		VideoExtensions:       extensionsStr,
		MatchConflictBehavior: string(settings.MatchConflictBehavior),
		UnknownMediaBehavior:  string(settings.UnknownMediaBehavior),
	}

	dbSettings, err := s.queries.UpdateImportSettings(ctx, params)
	if err != nil {
		return nil, err
	}

	result := SettingsFromDB(dbSettings)
	return &result, nil
}

// RefreshSettings reloads settings from the database.
func (s *Service) RefreshSettings(ctx context.Context) error {
	_, err := s.GetSettings(ctx)
	return err
}
