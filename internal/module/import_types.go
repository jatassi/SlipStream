package module

// CompletedDownload represents a download that has finished and is ready for import.
// Populated by the framework from the download mapping and download client data.
type CompletedDownload struct {
	DownloadID   string
	ClientID     int64
	ClientName   string
	DownloadPath string // Full path to the download directory or file
	Size         int64

	// ModuleType and EntityType are DERIVED by the framework from the legacy
	// download_mappings columns (movie_id, series_id, episode_id, is_season_pack).
	ModuleType Type
	EntityType EntityType
	EntityID   int64

	// Group download info (season packs, full albums, etc.)
	IsGroupDownload bool
	GroupEntityType EntityType // Parent entity type (e.g., "season", "series")
	GroupEntityID   int64      // Parent entity ID
	SeasonNumber    *int       // TV convenience field (from download_mappings.season_number)

	TargetSlotID *int64
	Source       string // "auto-search", "manual-search", "portal-request"

	MappingID      int64
	ImportAttempts int64
}

// MatchedEntity represents a library entity matched to a download or file.
type MatchedEntity struct {
	ModuleType Type
	EntityType EntityType
	EntityID   int64   // Primary entity
	EntityIDs  []int64 // All matched entities (populated for multi-entity files, e.g., S01E01E02)
	Title      string  // Display title

	RootFolder string  // Destination root folder path
	Confidence float64 // 0.0–1.0
	Source     string  // "queue", "parse", "manual"

	IsUpgrade          bool
	ExistingFileID     *int64
	ExistingFilePath   string
	CandidateQualityID int
	ExistingQualityID  int
	QualityProfileID   int64

	// TokenData carries template variable values for path/naming resolution.
	// Populated by the module's ImportHandler. Keys are well-known names that map
	// to renamer.TokenContext fields (e.g., "MovieTitle", "SeriesTitle", "SeasonNumber").
	// The framework adds quality/media-info keys during import.
	TokenData map[string]any

	// GroupInfo for multi-file downloads (populated for season packs, albums, etc.)
	GroupInfo *GroupMatchInfo
}

// GroupMatchInfo carries parent entity info for group downloads.
type GroupMatchInfo struct {
	ParentEntityType EntityType
	ParentEntityID   int64
	IsSeasonPack     bool
	IsCompleteSeries bool
}

// QualityInfo describes the quality of a file being imported.
type QualityInfo struct {
	QualityID  int
	Quality    string // "1080p", "720p", etc.
	Source     string // "WEBDL", "BluRay", etc.
	Codec      string
	HDRFormats []string
}

// ImportResult is the result of a module importing a file (creating the file record).
type ImportResult struct {
	FileID          int64  // Created file record ID
	DestinationPath string // Final path in library
	QualityID       int    // Quality ID assigned to the file record
}

// MediaInfoFieldDecl declares a media info field relevant to the module.
type MediaInfoFieldDecl struct {
	Name     string // "video_codec", "audio_codec", "resolution", "audio_channels", "dynamic_range"
	Required bool   // If true, probe failure prevents import completion
}
