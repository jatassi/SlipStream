//nolint:revive // package name mirrors domain concept
package importer

import (
	"context"
	"strings"

	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/import/renamer"
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

	// TV Renaming settings
	RenameEpisodes           bool                     `json:"renameEpisodes"`
	ReplaceIllegalCharacters bool                     `json:"replaceIllegalCharacters"`
	ColonReplacement         renamer.ColonReplacement `json:"colonReplacement"`
	CustomColonReplacement   string                   `json:"customColonReplacement,omitempty"`

	// Episode format patterns
	StandardEpisodeFormat string `json:"standardEpisodeFormat"`
	DailyEpisodeFormat    string `json:"dailyEpisodeFormat"`
	AnimeEpisodeFormat    string `json:"animeEpisodeFormat"`

	// Folder patterns
	SeriesFolderFormat   string `json:"seriesFolderFormat"`
	SeasonFolderFormat   string `json:"seasonFolderFormat"`
	SpecialsFolderFormat string `json:"specialsFolderFormat"`

	// Multi-episode style
	MultiEpisodeStyle renamer.MultiEpisodeStyle `json:"multiEpisodeStyle"`

	// Movie Renaming settings
	RenameMovies      bool   `json:"renameMovies"`
	MovieFolderFormat string `json:"movieFolderFormat"`
	MovieFileFormat   string `json:"movieFileFormat"`
}

// DefaultImportSettings returns the default import settings.
func DefaultImportSettings() ImportSettings {
	return ImportSettings{
		ValidationLevel:   ValidationStandard,
		MinimumFileSizeMB: 100,
		VideoExtensions:   []string{".mkv", ".mp4", ".avi", ".m4v", ".mov", ".wmv", ".ts", ".m2ts", ".webm"},

		MatchConflictBehavior: MatchFail,
		UnknownMediaBehavior:  UnknownIgnore,

		RenameEpisodes:           true,
		ReplaceIllegalCharacters: true,
		ColonReplacement:         renamer.ColonSmart,

		StandardEpisodeFormat: "{Series Title} - S{season:00}E{episode:00} - {Quality Title} {MediaInfo VideoDynamicRangeType}",
		DailyEpisodeFormat:    "{Series Title} - {Air-Date} - {Episode Title} {Quality Full}",
		AnimeEpisodeFormat:    "{Series Title} - S{season:00}E{episode:00} - {Episode Title} {Quality Full}",

		SeriesFolderFormat:   "{Series Title}",
		SeasonFolderFormat:   "Season {season}",
		SpecialsFolderFormat: "Specials",

		MultiEpisodeStyle: renamer.StyleExtend,

		RenameMovies:      true,
		MovieFolderFormat: "{Movie Title} ({Year})",
		MovieFileFormat:   "{Movie Title} ({Year}) - {Quality Title}",
	}
}

// ToRenamerSettings converts ImportSettings to renamer.Settings.
func (s *ImportSettings) ToRenamerSettings() renamer.Settings {
	return renamer.Settings{
		RenameEpisodes:           s.RenameEpisodes,
		ReplaceIllegalCharacters: s.ReplaceIllegalCharacters,
		ColonReplacement:         s.ColonReplacement,
		CustomColonReplacement:   s.CustomColonReplacement,

		StandardEpisodeFormat: s.StandardEpisodeFormat,
		DailyEpisodeFormat:    s.DailyEpisodeFormat,
		AnimeEpisodeFormat:    s.AnimeEpisodeFormat,

		SeriesFolderFormat:   s.SeriesFolderFormat,
		SeasonFolderFormat:   s.SeasonFolderFormat,
		SpecialsFolderFormat: s.SpecialsFolderFormat,

		MultiEpisodeStyle: s.MultiEpisodeStyle,

		RenameMovies:      s.RenameMovies,
		MovieFolderFormat: s.MovieFolderFormat,
		MovieFileFormat:   s.MovieFileFormat,

		CaseMode: renamer.CaseDefault,
	}
}

// SettingsFromDB converts a sqlc.ImportSetting to ImportSettings.
func SettingsFromDB(db *sqlc.ImportSetting) ImportSettings {
	extensions := strings.Split(db.VideoExtensions, ",")
	for i := range extensions {
		extensions[i] = strings.TrimSpace(extensions[i])
	}

	var customColon string
	if db.CustomColonReplacement.Valid {
		customColon = db.CustomColonReplacement.String
	}

	return ImportSettings{
		ValidationLevel:   ValidationLevel(db.ValidationLevel),
		MinimumFileSizeMB: int(db.MinimumFileSizeMb),
		VideoExtensions:   extensions,

		MatchConflictBehavior: MatchConflictBehavior(db.MatchConflictBehavior),
		UnknownMediaBehavior:  UnknownMediaBehavior(db.UnknownMediaBehavior),

		RenameEpisodes:           db.RenameEpisodes,
		ReplaceIllegalCharacters: db.ReplaceIllegalCharacters,
		ColonReplacement:         renamer.ColonReplacement(db.ColonReplacement),
		CustomColonReplacement:   customColon,

		StandardEpisodeFormat: db.StandardEpisodeFormat,
		DailyEpisodeFormat:    db.DailyEpisodeFormat,
		AnimeEpisodeFormat:    db.AnimeEpisodeFormat,

		SeriesFolderFormat:   db.SeriesFolderFormat,
		SeasonFolderFormat:   db.SeasonFolderFormat,
		SpecialsFolderFormat: db.SpecialsFolderFormat,

		MultiEpisodeStyle: renamer.MultiEpisodeStyle(db.MultiEpisodeStyle),

		RenameMovies:      db.RenameMovies,
		MovieFolderFormat: db.MovieFolderFormat,
		MovieFileFormat:   db.MovieFileFormat,
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
		ValidationLevel:          string(settings.ValidationLevel),
		MinimumFileSizeMb:        int64(settings.MinimumFileSizeMB),
		VideoExtensions:          extensionsStr,
		MatchConflictBehavior:    string(settings.MatchConflictBehavior),
		UnknownMediaBehavior:     string(settings.UnknownMediaBehavior),
		RenameEpisodes:           settings.RenameEpisodes,
		ReplaceIllegalCharacters: settings.ReplaceIllegalCharacters,
		ColonReplacement:         string(settings.ColonReplacement),
		StandardEpisodeFormat:    settings.StandardEpisodeFormat,
		DailyEpisodeFormat:       settings.DailyEpisodeFormat,
		AnimeEpisodeFormat:       settings.AnimeEpisodeFormat,
		SeriesFolderFormat:       settings.SeriesFolderFormat,
		SeasonFolderFormat:       settings.SeasonFolderFormat,
		SpecialsFolderFormat:     settings.SpecialsFolderFormat,
		MultiEpisodeStyle:        string(settings.MultiEpisodeStyle),
		RenameMovies:             settings.RenameMovies,
		MovieFolderFormat:        settings.MovieFolderFormat,
		MovieFileFormat:          settings.MovieFileFormat,
	}

	if settings.CustomColonReplacement != "" {
		params.CustomColonReplacement.String = settings.CustomColonReplacement
		params.CustomColonReplacement.Valid = true
	}

	dbSettings, err := s.queries.UpdateImportSettings(ctx, params)
	if err != nil {
		return nil, err
	}

	// Update the renamer with new settings
	result := SettingsFromDB(dbSettings)
	renSettings := result.ToRenamerSettings()
	s.UpdateRenamerSettings(&renSettings)

	return &result, nil
}

// RefreshSettings reloads settings from the database and updates the renamer.
func (s *Service) RefreshSettings(ctx context.Context) error {
	settings, err := s.GetSettings(ctx)
	if err != nil {
		return err
	}

	renSettings := settings.ToRenamerSettings()
	s.UpdateRenamerSettings(&renSettings)
	return nil
}
