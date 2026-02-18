package movies

import "time"

// Movie represents a movie in the library.
type Movie struct {
	ID               int64       `json:"id"`
	Title            string      `json:"title"`
	SortTitle        string      `json:"sortTitle"`
	Year             int         `json:"year,omitempty"`
	TmdbID           int         `json:"tmdbId,omitempty"`
	ImdbID           string      `json:"imdbId,omitempty"`
	Overview         string      `json:"overview,omitempty"`
	Runtime          int         `json:"runtime,omitempty"`
	Path             string      `json:"path,omitempty"`
	RootFolderID     int64       `json:"rootFolderId,omitempty"`
	QualityProfileID int64       `json:"qualityProfileId,omitempty"`
	Monitored        bool        `json:"monitored"`
	Status           string      `json:"status"`
	StatusMessage    *string     `json:"statusMessage"`
	ActiveDownloadID *string     `json:"activeDownloadId"`
	AddedAt          time.Time   `json:"addedAt"`
	UpdatedAt        time.Time   `json:"updatedAt,omitempty"`
	SizeOnDisk       int64       `json:"sizeOnDisk,omitempty"`
	MovieFiles       []MovieFile `json:"movieFiles,omitempty"`

	ReleaseDate           *time.Time `json:"releaseDate,omitempty"`
	PhysicalReleaseDate   *time.Time `json:"physicalReleaseDate,omitempty"`
	TheatricalReleaseDate *time.Time `json:"theatricalReleaseDate,omitempty"`

	Studio        string `json:"studio,omitempty"`
	TvdbID        int    `json:"tvdbId,omitempty"`
	ContentRating string `json:"contentRating,omitempty"`

	AddedBy         *int64 `json:"addedBy,omitempty"`
	AddedByUsername string `json:"addedByUsername,omitempty"`
}

// MovieFile represents a movie file on disk.
type MovieFile struct {
	ID            int64     `json:"id"`
	MovieID       int64     `json:"movieId"`
	Path          string    `json:"path"`
	Size          int64     `json:"size"`
	Quality       string    `json:"quality,omitempty"`
	VideoCodec    string    `json:"videoCodec,omitempty"`
	AudioCodec    string    `json:"audioCodec,omitempty"`
	AudioChannels string    `json:"audioChannels,omitempty"`
	DynamicRange  string    `json:"dynamicRange,omitempty"`
	Resolution    string    `json:"resolution,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
	SlotID        *int64    `json:"slotId,omitempty"`
}

// CreateMovieInput contains fields for creating a movie.
type CreateMovieInput struct {
	Title            string `json:"title"`
	Year             int    `json:"year,omitempty"`
	TmdbID           int    `json:"tmdbId,omitempty"`
	ImdbID           string `json:"imdbId,omitempty"`
	Overview         string `json:"overview,omitempty"`
	Runtime          int    `json:"runtime,omitempty"`
	Path             string `json:"path,omitempty"`
	RootFolderID     int64  `json:"rootFolderId"`
	QualityProfileID int64  `json:"qualityProfileId"`
	Monitored        bool   `json:"monitored"`

	Studio        string `json:"studio,omitempty"`
	TvdbID        int    `json:"tvdbId,omitempty"`
	ContentRating string `json:"contentRating,omitempty"`

	// Release dates (YYYY-MM-DD strings)
	ReleaseDate           string `json:"releaseDate,omitempty"`           // Digital/streaming release date
	PhysicalReleaseDate   string `json:"physicalReleaseDate,omitempty"`   // Bluray release date
	TheatricalReleaseDate string `json:"theatricalReleaseDate,omitempty"` // Theatrical release date

	AddedBy *int64 `json:"-"`
}

// UpdateMovieInput contains fields for updating a movie.
type UpdateMovieInput struct {
	Title            *string `json:"title,omitempty"`
	Year             *int    `json:"year,omitempty"`
	TmdbID           *int    `json:"tmdbId,omitempty"`
	ImdbID           *string `json:"imdbId,omitempty"`
	Overview         *string `json:"overview,omitempty"`
	Runtime          *int    `json:"runtime,omitempty"`
	Path             *string `json:"path,omitempty"`
	RootFolderID     *int64  `json:"rootFolderId,omitempty"`
	QualityProfileID *int64  `json:"qualityProfileId,omitempty"`
	Monitored        *bool   `json:"monitored,omitempty"`

	Studio        *string `json:"studio,omitempty"`
	ContentRating *string `json:"contentRating,omitempty"`

	// Release dates (YYYY-MM-DD strings)
	ReleaseDate           *string `json:"releaseDate,omitempty"`           // Digital/streaming release date
	PhysicalReleaseDate   *string `json:"physicalReleaseDate,omitempty"`   // Bluray release date
	TheatricalReleaseDate *string `json:"theatricalReleaseDate,omitempty"` // Theatrical release date
}

// ListMoviesOptions contains options for listing movies.
type ListMoviesOptions struct {
	Search       string `json:"search,omitempty"`
	Monitored    *bool  `json:"monitored,omitempty"`
	RootFolderID *int64 `json:"rootFolderId,omitempty"`
}

// CreateMovieFileInput contains fields for creating a movie file.
type CreateMovieFileInput struct {
	Path             string `json:"path"`
	Size             int64  `json:"size"`
	Quality          string `json:"quality,omitempty"`
	QualityID        *int64 `json:"qualityId,omitempty"`
	VideoCodec       string `json:"videoCodec,omitempty"`
	AudioCodec       string `json:"audioCodec,omitempty"`
	AudioChannels    string `json:"audioChannels,omitempty"`
	DynamicRange     string `json:"dynamicRange,omitempty"`
	Resolution       string `json:"resolution,omitempty"`
	OriginalPath     string `json:"originalPath,omitempty"`     // Source path before import
	OriginalFilename string `json:"originalFilename,omitempty"` // Original filename before rename
}

// generateSortTitle creates a sort-friendly title by removing leading articles.
func generateSortTitle(title string) string {
	prefixes := []string{"The ", "A ", "An "}
	for _, prefix := range prefixes {
		if len(title) > len(prefix) && title[:len(prefix)] == prefix {
			return title[len(prefix):]
		}
	}
	return title
}
