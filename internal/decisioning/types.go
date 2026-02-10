package decisioning

// MediaType represents the type of media being searched.
type MediaType string

const (
	MediaTypeMovie   MediaType = "movie"
	MediaTypeEpisode MediaType = "episode"
	MediaTypeSeason  MediaType = "season"
	MediaTypeSeries  MediaType = "series"
)

// SearchableItem represents a wanted media item for release decisioning.
type SearchableItem struct {
	MediaType MediaType `json:"mediaType"`
	MediaID   int64     `json:"mediaId"`
	Title     string    `json:"title"`
	Year      int       `json:"year,omitempty"`

	// External IDs for search queries
	ImdbID string `json:"imdbId,omitempty"`
	TmdbID int    `json:"tmdbId,omitempty"`
	TvdbID int    `json:"tvdbId,omitempty"`

	// TV-specific fields
	SeriesID      int64 `json:"seriesId,omitempty"`
	SeasonNumber  int   `json:"seasonNumber,omitempty"`
	EpisodeNumber int   `json:"episodeNumber,omitempty"`

	// Quality profile for scoring
	QualityProfileID int64 `json:"qualityProfileId"`

	// Current file info for upgrades.
	// HasFile must be true and CurrentQualityID must be the HIGHEST quality
	// across all file records when item has existing files.
	HasFile          bool `json:"hasFile"`
	CurrentQualityID int  `json:"currentQualityId,omitempty"`

	// Slot targeting (for multi-version mode)
	TargetSlotID *int64 `json:"targetSlotId,omitempty"`
}
