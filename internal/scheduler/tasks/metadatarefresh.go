package tasks

import (
	"context"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/library/librarymanager"
	"github.com/slipstream/slipstream/internal/library/movies"
	"github.com/slipstream/slipstream/internal/library/tv"
	"github.com/slipstream/slipstream/internal/scheduler"
)

// MetadataRefreshTask handles scheduled metadata refresh for the entire library.
type MetadataRefreshTask struct {
	libraryManager *librarymanager.Service
	movieService   *movies.Service
	tvService      *tv.Service
	logger         zerolog.Logger
}

// NewMetadataRefreshTask creates a new metadata refresh task.
func NewMetadataRefreshTask(lm *librarymanager.Service, ms *movies.Service, ts *tv.Service, logger zerolog.Logger) *MetadataRefreshTask {
	return &MetadataRefreshTask{
		libraryManager: lm,
		movieService:   ms,
		tvService:      ts,
		logger:         logger.With().Str("task", "metadata-refresh").Logger(),
	}
}

// Run executes the metadata refresh task for all movies and series.
func (t *MetadataRefreshTask) Run(ctx context.Context) error {
	t.logger.Info().Msg("Starting scheduled metadata refresh")

	var lastErr error

	// Refresh all movies
	moviesRefreshed, moviesErr := t.refreshMovies(ctx)
	if moviesErr != nil {
		lastErr = moviesErr
	}

	// Refresh all series
	seriesRefreshed, seriesErr := t.refreshSeries(ctx)
	if seriesErr != nil {
		lastErr = seriesErr
	}

	t.logger.Info().
		Int("moviesRefreshed", moviesRefreshed).
		Int("seriesRefreshed", seriesRefreshed).
		Msg("Scheduled metadata refresh completed")

	return lastErr
}

// refreshMovies refreshes metadata for all movies.
func (t *MetadataRefreshTask) refreshMovies(ctx context.Context) (int, error) {
	movieList, err := t.movieService.List(ctx, movies.ListMoviesOptions{})
	if err != nil {
		t.logger.Error().Err(err).Msg("Failed to list movies")
		return 0, err
	}

	if len(movieList) == 0 {
		t.logger.Info().Msg("No movies in library, skipping movie metadata refresh")
		return 0, nil
	}

	t.logger.Info().Int("count", len(movieList)).Msg("Refreshing movie metadata")

	refreshed := 0
	var lastErr error

	for _, movie := range movieList {
		_, err := t.libraryManager.RefreshMovieMetadata(ctx, movie.ID)
		if err != nil {
			if err == librarymanager.ErrNoMetadataProvider {
				t.logger.Warn().Msg("No metadata provider configured, stopping movie refresh")
				return refreshed, err
			}
			t.logger.Warn().Err(err).Int64("movieId", movie.ID).Str("title", movie.Title).Msg("Failed to refresh movie metadata")
			lastErr = err
			continue
		}

		refreshed++
		t.logger.Debug().Int64("movieId", movie.ID).Str("title", movie.Title).Msg("Refreshed movie metadata")
	}

	t.logger.Info().Int("refreshed", refreshed).Int("total", len(movieList)).Msg("Movie metadata refresh completed")

	return refreshed, lastErr
}

// refreshSeries refreshes metadata for all series.
func (t *MetadataRefreshTask) refreshSeries(ctx context.Context) (int, error) {
	seriesList, err := t.tvService.ListSeries(ctx, tv.ListSeriesOptions{})
	if err != nil {
		t.logger.Error().Err(err).Msg("Failed to list series")
		return 0, err
	}

	if len(seriesList) == 0 {
		t.logger.Info().Msg("No series in library, skipping series metadata refresh")
		return 0, nil
	}

	t.logger.Info().Int("count", len(seriesList)).Msg("Refreshing series metadata")

	refreshed := 0
	var lastErr error

	for _, series := range seriesList {
		_, err := t.libraryManager.RefreshSeriesMetadata(ctx, series.ID)
		if err != nil {
			if err == librarymanager.ErrNoMetadataProvider {
				t.logger.Warn().Msg("No metadata provider configured, stopping series refresh")
				return refreshed, err
			}
			t.logger.Warn().Err(err).Int64("seriesId", series.ID).Str("title", series.Title).Msg("Failed to refresh series metadata")
			lastErr = err
			continue
		}

		refreshed++
		t.logger.Debug().Int64("seriesId", series.ID).Str("title", series.Title).Msg("Refreshed series metadata")
	}

	t.logger.Info().Int("refreshed", refreshed).Int("total", len(seriesList)).Msg("Series metadata refresh completed")

	return refreshed, lastErr
}

// RegisterMetadataRefreshTask registers the metadata refresh task with the scheduler.
func RegisterMetadataRefreshTask(sched *scheduler.Scheduler, lm *librarymanager.Service, ms *movies.Service, ts *tv.Service, logger zerolog.Logger) error {
	task := NewMetadataRefreshTask(lm, ms, ts, logger)

	return sched.RegisterTask(scheduler.TaskConfig{
		ID:          "metadata-refresh",
		Name:        "Metadata Refresh",
		Description: "Refreshes metadata for all movies and series in the library",
		Cron:        "30 23 * * *", // 11:30 PM daily
		RunOnStart:  false,
		Func:        task.Run,
	})
}
