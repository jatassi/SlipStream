package metadata

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

var (
	ErrInvalidURL       = errors.New("invalid artwork URL")
	ErrDownloadFailed   = errors.New("artwork download failed")
	ErrInvalidMediaType = errors.New("invalid media type")
)

// ArtworkType represents the type of artwork.
type ArtworkType string

const (
	ArtworkTypePoster     ArtworkType = "poster"
	ArtworkTypeBackdrop   ArtworkType = "backdrop"
	ArtworkTypeLogo       ArtworkType = "logo"
	ArtworkTypeStudioLogo ArtworkType = "studio_logo"
)

// MediaType represents the type of media.
type MediaType string

const (
	MediaTypeMovie  MediaType = "movie"
	MediaTypeSeries MediaType = "series"
)

// ArtworkConfig holds configuration for artwork downloading.
type ArtworkConfig struct {
	// BaseDir is the base directory for storing artwork.
	BaseDir string

	// Timeout is the HTTP request timeout.
	Timeout time.Duration
}

// DefaultArtworkConfig returns default artwork configuration.
func DefaultArtworkConfig() ArtworkConfig {
	return ArtworkConfig{
		BaseDir: "data/artwork",
		Timeout: 30 * time.Second,
	}
}

// ArtworkBroadcaster is an interface for broadcasting artwork events.
type ArtworkBroadcaster interface {
	Broadcast(msgType string, payload interface{})
}

// ArtworkReadyPayload is the payload sent when artwork is ready.
type ArtworkReadyPayload struct {
	MediaType   string `json:"mediaType"`
	MediaID     int    `json:"mediaId"`
	ArtworkType string `json:"artworkType"`
}

// ArtworkDownloader handles downloading and storing artwork images.
type ArtworkDownloader struct {
	config      ArtworkConfig
	httpClient  *http.Client
	logger      *zerolog.Logger
	broadcaster ArtworkBroadcaster
}

// NewArtworkDownloader creates a new ArtworkDownloader.
func NewArtworkDownloader(cfg ArtworkConfig, logger *zerolog.Logger) *ArtworkDownloader {
	subLogger := logger.With().Str("component", "artwork").Logger()
	return &ArtworkDownloader{
		config: cfg,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		logger: &subLogger,
	}
}

// SetBroadcaster sets the broadcaster for artwork events.
func (d *ArtworkDownloader) SetBroadcaster(b ArtworkBroadcaster) {
	d.broadcaster = b
}

// notifyArtworkReady broadcasts that artwork is ready.
func (d *ArtworkDownloader) notifyArtworkReady(mediaType MediaType, mediaID int, artworkType ArtworkType) {
	if d.broadcaster == nil {
		return
	}
	payload := ArtworkReadyPayload{
		MediaType:   string(mediaType),
		MediaID:     mediaID,
		ArtworkType: string(artworkType),
	}
	d.broadcaster.Broadcast("artwork:ready", payload)
}

// Download downloads artwork from a URL and saves it locally.
// Returns the local file path on success.
func (d *ArtworkDownloader) Download(ctx context.Context, url string, mediaType MediaType, mediaID int, artworkType ArtworkType) (string, error) {
	if url == "" {
		return "", ErrInvalidURL
	}

	// Determine file extension from URL
	ext := d.getExtension(url)
	if ext == "" {
		ext = ".jpg" // Default to jpg
	}

	// Build destination path: {baseDir}/{mediaType}/{id}_{artworkType}{ext}
	// e.g., data/artwork/movie/603_poster.jpg
	dir := filepath.Join(d.config.BaseDir, string(mediaType))
	filename := fmt.Sprintf("%d_%s%s", mediaID, artworkType, ext)
	destPath := filepath.Join(dir, filename)

	// Create directory if needed
	if err := os.MkdirAll(dir, 0o750); err != nil {
		d.logger.Error().Err(err).Str("dir", dir).Msg("Failed to create artwork directory")
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	// Download the file
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := d.httpClient.Do(req)
	if err != nil {
		d.logger.Error().Err(err).Str("url", url).Msg("Artwork download failed")
		return "", fmt.Errorf("%w: %w", ErrDownloadFailed, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		d.logger.Error().Int("status", resp.StatusCode).Str("url", url).Msg("Artwork download failed")
		return "", fmt.Errorf("%w: status %d", ErrDownloadFailed, resp.StatusCode)
	}

	// Create destination file
	file, err := os.Create(destPath)
	if err != nil {
		d.logger.Error().Err(err).Str("path", destPath).Msg("Failed to create artwork file")
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Copy response body to file
	written, err := io.Copy(file, resp.Body)
	if err != nil {
		d.logger.Error().Err(err).Str("path", destPath).Msg("Failed to write artwork file")
		os.Remove(destPath) // Clean up partial file
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	d.logger.Info().
		Str("url", url).
		Str("path", destPath).
		Int64("bytes", written).
		Msg("Artwork downloaded successfully")

	return destPath, nil
}

// DownloadMovieArtwork downloads both poster and backdrop for a movie.
func (d *ArtworkDownloader) DownloadMovieArtwork(ctx context.Context, movie *MovieResult) error {
	if movie == nil {
		return ErrInvalidMediaType
	}

	d.downloadArtworkIfExists(ctx, movie.PosterURL, MediaTypeMovie, movie.ID, ArtworkTypePoster, "movie poster")
	d.downloadArtworkIfExists(ctx, movie.BackdropURL, MediaTypeMovie, movie.ID, ArtworkTypeBackdrop, "movie backdrop")
	d.downloadArtworkIfExists(ctx, movie.LogoURL, MediaTypeMovie, movie.ID, ArtworkTypeLogo, "movie logo")
	d.downloadArtworkIfExists(ctx, movie.StudioLogoURL, MediaTypeMovie, movie.ID, ArtworkTypeStudioLogo, "studio logo")

	return nil
}

func (d *ArtworkDownloader) downloadArtworkIfExists(ctx context.Context, url string, mediaType MediaType, mediaID int, artworkType ArtworkType, logName string) {
	if url == "" {
		return
	}
	path, err := d.Download(ctx, url, mediaType, mediaID, artworkType)
	if err != nil {
		d.logger.Warn().Err(err).Int("movieId", mediaID).Msgf("Failed to download %s", logName)
		return
	}
	d.logger.Debug().Str("path", path).Int("movieId", mediaID).Msgf("Downloaded %s", logName)
	d.notifyArtworkReady(mediaType, mediaID, artworkType)
}

// DownloadSeriesArtwork downloads both poster and backdrop for a series.
// Uses TmdbID for storage since the frontend expects artwork keyed by TMDB ID.
func (d *ArtworkDownloader) DownloadSeriesArtwork(ctx context.Context, series *SeriesResult) error {
	if series == nil {
		return ErrInvalidMediaType
	}

	artworkID := series.TmdbID
	if artworkID == 0 {
		artworkID = series.ID
	}
	if artworkID == 0 {
		d.logger.Warn().Str("title", series.Title).Msg("No valid ID for series artwork download")
		return ErrInvalidMediaType
	}

	d.downloadArtworkIfExists(ctx, series.PosterURL, MediaTypeSeries, artworkID, ArtworkTypePoster, "series poster")
	d.downloadArtworkIfExists(ctx, series.BackdropURL, MediaTypeSeries, artworkID, ArtworkTypeBackdrop, "series backdrop")
	d.downloadArtworkIfExists(ctx, series.LogoURL, MediaTypeSeries, artworkID, ArtworkTypeLogo, "series logo")
	d.downloadArtworkIfExists(ctx, series.NetworkLogoURL, MediaTypeSeries, artworkID, ArtworkTypeStudioLogo, "network logo")

	return nil
}

// GetArtworkPath returns the local path for artwork if it exists.
func (d *ArtworkDownloader) GetArtworkPath(mediaType MediaType, mediaID int, artworkType ArtworkType) string {
	// Try common extensions
	extensions := []string{".jpg", ".jpeg", ".png", ".webp", ".svg"}

	for _, ext := range extensions {
		filename := fmt.Sprintf("%d_%s%s", mediaID, artworkType, ext)
		path := filepath.Join(d.config.BaseDir, string(mediaType), filename)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}

// HasArtwork checks if artwork exists locally.
func (d *ArtworkDownloader) HasArtwork(mediaType MediaType, mediaID int, artworkType ArtworkType) bool {
	return d.GetArtworkPath(mediaType, mediaID, artworkType) != ""
}

// DeleteArtwork removes artwork for a media item.
func (d *ArtworkDownloader) DeleteArtwork(mediaType MediaType, mediaID int) error {
	dir := filepath.Join(d.config.BaseDir, string(mediaType))

	// Find and delete all artwork files for this media ID
	pattern := filepath.Join(dir, fmt.Sprintf("%d_*", mediaID))
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("failed to find artwork files: %w", err)
	}

	for _, path := range matches {
		if err := os.Remove(path); err != nil {
			d.logger.Warn().Err(err).Str("path", path).Msg("Failed to delete artwork file")
		} else {
			d.logger.Debug().Str("path", path).Msg("Deleted artwork file")
		}
	}

	return nil
}

// getExtension extracts the file extension from a URL.
func (d *ArtworkDownloader) getExtension(url string) string {
	// Find the last path segment
	lastSlash := strings.LastIndex(url, "/")
	if lastSlash == -1 {
		return ""
	}

	filename := url[lastSlash+1:]

	// Remove query string if present
	if qmark := strings.Index(filename, "?"); qmark != -1 {
		filename = filename[:qmark]
	}

	// Get extension
	if dot := strings.LastIndex(filename, "."); dot != -1 {
		ext := strings.ToLower(filename[dot:])
		// Validate it's a known image extension
		switch ext {
		case ".jpg", ".jpeg", ".png", ".webp", ".gif", ".svg":
			return ext
		}
	}

	return ""
}
