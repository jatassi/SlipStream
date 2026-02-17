package arrimport

import (
	"context"
	"fmt"
)

// Reader defines the interface for reading data from a source application.
type Reader interface {
	Validate(ctx context.Context) error
	ReadRootFolders(ctx context.Context) ([]SourceRootFolder, error)
	ReadQualityProfiles(ctx context.Context) ([]SourceQualityProfile, error)
	ReadMovies(ctx context.Context) ([]SourceMovie, error)
	ReadSeries(ctx context.Context) ([]SourceSeries, error)
	ReadEpisodes(ctx context.Context, seriesID int64) ([]SourceEpisode, error)
	ReadEpisodeFiles(ctx context.Context, seriesID int64) ([]SourceEpisodeFile, error)
	Close() error
}

// NewReader creates a Reader based on the connection config.
func NewReader(cfg ConnectionConfig) (Reader, error) {
	if cfg.DBPath != "" {
		return newSQLiteReader(cfg)
	}
	if cfg.URL != "" && cfg.APIKey != "" {
		return newAPIReader(cfg), nil
	}
	return nil, fmt.Errorf("invalid connection config: must provide either dbPath or url+apiKey")
}
