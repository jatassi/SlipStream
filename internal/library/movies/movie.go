package movies

import "time"

// Movie represents a movie in the library.
type Movie struct {
	ID               int64      `json:"id"`
	Title            string     `json:"title"`
	SortTitle        string     `json:"sortTitle"`
	Year             int        `json:"year,omitempty"`
	TmdbID           int        `json:"tmdbId,omitempty"`
	ImdbID           string     `json:"imdbId,omitempty"`
	Overview         string     `json:"overview,omitempty"`
	Runtime          int        `json:"runtime,omitempty"`
	Path             string     `json:"path,omitempty"`
	RootFolderID     int64      `json:"rootFolderId,omitempty"`
	QualityProfileID int64      `json:"qualityProfileId,omitempty"`
	Monitored        bool       `json:"monitored"`
	Status           string     `json:"status"` // "missing", "downloading", "available"
	AddedAt          time.Time  `json:"addedAt"`
	UpdatedAt        time.Time  `json:"updatedAt,omitempty"`
	HasFile          bool       `json:"hasFile"`
	SizeOnDisk       int64      `json:"sizeOnDisk,omitempty"`
	MovieFiles       []MovieFile `json:"movieFiles,omitempty"`

	// Release dates
	ReleaseDate         *time.Time `json:"releaseDate,omitempty"`         // Digital/streaming release date
	PhysicalReleaseDate *time.Time `json:"physicalReleaseDate,omitempty"` // Bluray release date

	// Availability
	Released           bool   `json:"released"`           // True if release date is in the past
	AvailabilityStatus string `json:"availabilityStatus"` // Badge text: "Available" or "Unreleased"
}

// MovieFile represents a movie file on disk.
type MovieFile struct {
	ID         int64     `json:"id"`
	MovieID    int64     `json:"movieId"`
	Path       string    `json:"path"`
	Size       int64     `json:"size"`
	Quality    string    `json:"quality,omitempty"`
	VideoCodec string    `json:"videoCodec,omitempty"`
	AudioCodec string    `json:"audioCodec,omitempty"`
	Resolution string    `json:"resolution,omitempty"`
	CreatedAt  time.Time `json:"createdAt"`
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

	// Release dates (YYYY-MM-DD strings)
	ReleaseDate         string `json:"releaseDate,omitempty"`         // Digital/streaming release date
	PhysicalReleaseDate string `json:"physicalReleaseDate,omitempty"` // Bluray release date
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

	// Release dates (YYYY-MM-DD strings)
	ReleaseDate         *string `json:"releaseDate,omitempty"`         // Digital/streaming release date
	PhysicalReleaseDate *string `json:"physicalReleaseDate,omitempty"` // Bluray release date
}

// ListMoviesOptions contains options for listing movies.
type ListMoviesOptions struct {
	Search       string `json:"search,omitempty"`
	Monitored    *bool  `json:"monitored,omitempty"`
	RootFolderID *int64 `json:"rootFolderId,omitempty"`
	Page         int    `json:"page,omitempty"`
	PageSize     int    `json:"pageSize,omitempty"`
}

// CreateMovieFileInput contains fields for creating a movie file.
type CreateMovieFileInput struct {
	Path       string `json:"path"`
	Size       int64  `json:"size"`
	Quality    string `json:"quality,omitempty"`
	VideoCodec string `json:"videoCodec,omitempty"`
	AudioCodec string `json:"audioCodec,omitempty"`
	Resolution string `json:"resolution,omitempty"`
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
