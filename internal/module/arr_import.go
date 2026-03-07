package module

import (
	"context"

	"github.com/slipstream/slipstream/internal/library/scanner"
	"github.com/slipstream/slipstream/internal/library/slots"
)

// ArrImportSlotsService is the subset of slot operations needed during arr import.
type ArrImportSlotsService interface {
	IsMultiVersionEnabled(ctx context.Context) bool
	InitializeSlotAssignments(ctx context.Context, mediaType string, mediaID int64) error
	DetermineTargetSlot(ctx context.Context, parsed *scanner.ParsedMedia, mediaType string, mediaID int64) (*slots.SlotAssignment, error)
	AssignFileToSlot(ctx context.Context, mediaType string, mediaID, slotID, fileID int64) error
}

// ArrImportMappings holds user-provided mapping choices (root folders, quality profiles, selected items).
type ArrImportMappings struct {
	RootFolderMapping     map[string]int64 // source root folder path -> SlipStream root folder ID
	QualityProfileMapping map[int64]int64  // source quality profile ID -> SlipStream profile ID
	SelectedIDs           []int            // external IDs of selected items (TMDB IDs for movies, TVDB IDs for series)
}

// ImportedEntity is the result of importing a single entity from an external *arr app.
// Return convention: non-nil entity + nil error = created; nil + nil = skipped; nil + error = failed.
type ImportedEntity struct {
	EntityType    EntityType
	EntityID      int64
	Title         string
	FilesImported int
	Errors        []string // per-file errors (non-fatal); entity was still created
}

// ArrImportPreviewItem represents a single item in the import preview.
type ArrImportPreviewItem struct {
	Title            string `json:"title"`
	Year             int    `json:"year"`
	TmdbID           int    `json:"tmdbId"`
	TvdbID           int    `json:"tvdbId,omitempty"`
	HasFile          bool   `json:"hasFile"`
	Quality          string `json:"quality,omitempty"`
	EpisodeCount     int    `json:"episodeCount,omitempty"`
	FileCount        int    `json:"fileCount"`
	QualityProfileID int64  `json:"qualityProfileId"`
	Monitored        bool   `json:"monitored"`
	Status           string `json:"status"`
	SkipReason       string `json:"skipReason,omitempty"`
	PosterURL        string `json:"posterUrl,omitempty"`
}

// MovieArrImportAdapter enables import of movies from external *arr apps (Radarr).
type MovieArrImportAdapter interface {
	ExternalAppName() string
	PreviewMovies(ctx context.Context, reader ArrReader) ([]ArrImportPreviewItem, error)
	ImportMovie(ctx context.Context, movie ArrSourceMovie, mappings ArrImportMappings) (*ImportedEntity, error)
}

// TVArrImportAdapter enables import of TV series from external *arr apps (Sonarr).
type TVArrImportAdapter interface {
	ExternalAppName() string
	PreviewSeries(ctx context.Context, reader ArrReader) ([]ArrImportPreviewItem, error)
	ImportSeries(ctx context.Context, series ArrSourceSeries, reader ArrReader, mappings ArrImportMappings) (*ImportedEntity, error)
}

// ArrReader abstracts reading data from an external *arr database or API.
type ArrReader interface {
	ReadRootFolders(ctx context.Context) ([]ArrSourceRootFolder, error)
	ReadQualityProfiles(ctx context.Context) ([]ArrSourceQualityProfile, error)
	ReadMovies(ctx context.Context) ([]ArrSourceMovie, error)
	ReadSeries(ctx context.Context) ([]ArrSourceSeries, error)
	ReadEpisodes(ctx context.Context, seriesID int64) ([]ArrSourceEpisode, error)
	ReadEpisodeFiles(ctx context.Context, seriesID int64) ([]ArrSourceEpisodeFile, error)
}
