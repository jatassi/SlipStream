package api

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/library/librarymanager"
	"github.com/slipstream/slipstream/internal/library/movies"
	"github.com/slipstream/slipstream/internal/library/tv"
	"github.com/slipstream/slipstream/internal/portal/requests"
)

// portalMediaProvisionerAdapter implements requests.MediaProvisioner to find or create media in library.
type portalMediaProvisionerAdapter struct {
	queries        *sqlc.Queries
	movieService   *movies.Service
	tvService      *tv.Service
	libraryManager *librarymanager.Service
	logger         *zerolog.Logger
}

func (a *portalMediaProvisionerAdapter) EnsureMovieInLibrary(ctx context.Context, input *requests.MediaProvisionInput) (int64, error) {
	// First, try to find existing movie by TMDB ID
	existing, err := a.movieService.GetByTmdbID(ctx, int(input.TmdbID))
	if err == nil && existing != nil {
		a.logger.Debug().Int64("tmdbID", input.TmdbID).Int64("movieID", existing.ID).Msg("found existing movie in library")
		return existing.ID, nil
	}

	// Movie not in library - get default settings and create it
	rootFolderID, qualityProfileID, err := a.getDefaultSettings(ctx, mediaTypeMovie)
	if err != nil {
		return 0, fmt.Errorf("failed to get default settings: %w", err)
	}

	// Use user's quality profile if provided
	if input.QualityProfileID != nil {
		qualityProfileID = *input.QualityProfileID
		a.logger.Debug().Int64("qualityProfileID", qualityProfileID).Msg("using user's assigned quality profile for movie")
	}

	movie, err := a.movieService.Create(ctx, &movies.CreateMovieInput{
		Title:            input.Title,
		Year:             input.Year,
		TmdbID:           int(input.TmdbID),
		RootFolderID:     rootFolderID,
		QualityProfileID: qualityProfileID,
		Monitored:        true,
		AddedBy:          input.AddedBy,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to create movie: %w", err)
	}

	a.logger.Info().Int64("tmdbID", input.TmdbID).Int64("movieID", movie.ID).Str("title", input.Title).Msg("created movie in library from request")

	// Fetch metadata including artwork and details
	if a.libraryManager != nil {
		if _, err := a.libraryManager.RefreshMovieMetadata(ctx, movie.ID); err != nil {
			a.logger.Warn().Err(err).Int64("movieID", movie.ID).Msg("failed to refresh movie metadata, movie created without full details")
		} else {
			a.logger.Info().Int64("movieID", movie.ID).Msg("fetched movie metadata and artwork")
		}
	}

	return movie.ID, nil
}

func (a *portalMediaProvisionerAdapter) EnsureSeriesInLibrary(ctx context.Context, input *requests.MediaProvisionInput) (int64, error) {
	existing, err := a.tvService.GetSeriesByTvdbID(ctx, int(input.TvdbID))
	if err == nil && existing != nil {
		return a.applyMonitoringToExistingSeries(ctx, existing.ID, input)
	}

	// Series not in library - get default settings and create it
	rootFolderID, qualityProfileID, err := a.getDefaultSettings(ctx, "tv")
	if err != nil {
		return 0, fmt.Errorf("failed to get default settings: %w", err)
	}

	// Use user's quality profile if provided
	if input.QualityProfileID != nil {
		qualityProfileID = *input.QualityProfileID
		a.logger.Debug().Int64("qualityProfileID", qualityProfileID).Msg("using user's assigned quality profile for series")
	}

	series, err := a.tvService.CreateSeries(ctx, &tv.CreateSeriesInput{
		Title:            input.Title,
		TvdbID:           int(input.TvdbID),
		RootFolderID:     rootFolderID,
		QualityProfileID: qualityProfileID,
		Monitored:        true,
		AddedBy:          input.AddedBy,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to create series: %w", err)
	}

	a.logger.Info().Int64("tvdbID", input.TvdbID).Int64("seriesID", series.ID).Str("title", input.Title).Msg("created series in library from request")

	// Fetch metadata including seasons and episodes
	if a.libraryManager != nil {
		if _, err := a.libraryManager.RefreshSeriesMetadata(ctx, series.ID); err != nil {
			a.logger.Warn().Err(err).Int64("seriesID", series.ID).Msg("failed to refresh series metadata, series created without episodes")
		} else {
			a.logger.Info().Int64("seriesID", series.ID).Msg("fetched series metadata with seasons and episodes")
		}
	}

	if err := a.applyPortalRequestMonitoring(ctx, series.ID, input); err != nil {
		a.logger.Warn().Err(err).Int64("seriesID", series.ID).Msg("failed to apply portal request monitoring")
	}

	return series.ID, nil
}

// applyPortalRequestMonitoring handles all monitoring scenarios for new series from portal requests.
func (a *portalMediaProvisionerAdapter) applyPortalRequestMonitoring(ctx context.Context, seriesID int64, input *requests.MediaProvisionInput) error {
	if len(input.RequestedSeasons) > 0 {
		if err := a.applyRequestedSeasonsMonitoring(ctx, seriesID, input.RequestedSeasons); err != nil {
			return err
		}
		if input.MonitorFuture {
			a.applyMonitorFuture(ctx, seriesID)
		}
	} else if input.MonitorFuture {
		if err := a.tvService.BulkMonitor(ctx, seriesID, tv.BulkMonitorInput{
			MonitorType:     tv.MonitorTypeFuture,
			IncludeSpecials: false,
		}); err != nil {
			return fmt.Errorf("failed to apply monitor future: %w", err)
		}
	}

	// Always unmonitor specials (Season 0) as a safety net
	a.unmonitorSpecials(ctx, seriesID)

	return nil
}

func (a *portalMediaProvisionerAdapter) unmonitorSpecials(ctx context.Context, seriesID int64) {
	if _, err := a.tvService.UpdateSeasonMonitored(ctx, seriesID, 0, false); err != nil {
		if !errors.Is(err, tv.ErrSeasonNotFound) {
			a.logger.Warn().Err(err).Int64("seriesID", seriesID).Msg("failed to unmonitor specials")
		}
	}
}

func (a *portalMediaProvisionerAdapter) applyMonitoringToExistingSeries(ctx context.Context, seriesID int64, input *requests.MediaProvisionInput) (int64, error) {
	a.logger.Debug().Int64("tvdbID", input.TvdbID).Int64("seriesID", seriesID).Msg("found existing series in library")
	if len(input.RequestedSeasons) > 0 {
		a.applyRequestedSeasonsMonitoringAdditive(ctx, seriesID, input.RequestedSeasons)
	}
	if input.MonitorFuture {
		a.applyMonitorFuture(ctx, seriesID)
	}
	return seriesID, nil
}

// applyRequestedSeasonsMonitoring unmonitors all seasons except the requested ones.
func (a *portalMediaProvisionerAdapter) applyRequestedSeasonsMonitoring(ctx context.Context, seriesID int64, requestedSeasons []int64) error {
	// Get all seasons for the series
	seasons, err := a.tvService.ListSeasons(ctx, seriesID)
	if err != nil {
		return fmt.Errorf("failed to get seasons: %w", err)
	}

	// Build a set of requested season numbers for quick lookup
	requestedSet := make(map[int64]bool)
	for _, sn := range requestedSeasons {
		requestedSet[sn] = true
	}

	// Update monitoring for each season
	for i := range seasons {
		season := &seasons[i]
		shouldMonitor := requestedSet[int64(season.SeasonNumber)]

		if season.Monitored != shouldMonitor {
			if _, err := a.tvService.UpdateSeasonMonitored(ctx, seriesID, season.SeasonNumber, shouldMonitor); err != nil {
				a.logger.Warn().
					Err(err).
					Int64("seriesID", seriesID).
					Int("seasonNumber", season.SeasonNumber).
					Bool("monitored", shouldMonitor).
					Msg("failed to update season monitoring")
			} else {
				a.logger.Debug().
					Int64("seriesID", seriesID).
					Int("seasonNumber", season.SeasonNumber).
					Bool("monitored", shouldMonitor).
					Msg("updated season monitoring")
			}
		}
	}

	a.logger.Info().
		Int64("seriesID", seriesID).
		Interface("requestedSeasons", requestedSeasons).
		Msg("applied requested seasons monitoring")

	return nil
}

func (a *portalMediaProvisionerAdapter) applyRequestedSeasonsMonitoringAdditive(ctx context.Context, seriesID int64, requestedSeasons []int64) {
	for _, sn := range requestedSeasons {
		if _, err := a.tvService.UpdateSeasonMonitored(ctx, seriesID, int(sn), true); err != nil {
			a.logger.Warn().Err(err).Int64("seriesID", seriesID).Int64("seasonNumber", sn).Msg("failed to monitor requested season")
		}
	}
}

func (a *portalMediaProvisionerAdapter) applyMonitorFuture(ctx context.Context, seriesID int64) {
	if err := a.queries.UpdateFutureEpisodesMonitored(ctx, sqlc.UpdateFutureEpisodesMonitoredParams{
		Monitored: 1,
		SeriesID:  seriesID,
	}); err != nil {
		a.logger.Warn().Err(err).Int64("seriesID", seriesID).Msg("failed to monitor future episodes")
	}
	if err := a.queries.UpdateFutureSeasonsMonitored(ctx, sqlc.UpdateFutureSeasonsMonitoredParams{
		Monitored:  1,
		SeriesID:   seriesID,
		SeriesID_2: seriesID,
	}); err != nil {
		a.logger.Warn().Err(err).Int64("seriesID", seriesID).Msg("failed to monitor future seasons")
	}
}

func (a *portalMediaProvisionerAdapter) getDefaultSettings(ctx context.Context, mediaType string) (rootFolderID, qualityProfileID int64, err error) {
	rootFolderID = a.resolveRootFolderID(ctx, mediaType)
	if rootFolderID == 0 {
		return 0, 0, fmt.Errorf("no %s root folder configured - please configure a root folder for %s content", mediaType, mediaType)
	}

	profiles, err := a.queries.ListQualityProfiles(ctx)
	if err != nil || len(profiles) == 0 {
		return 0, 0, errors.New("no quality profile configured")
	}
	qualityProfileID = profiles[0].ID

	return rootFolderID, qualityProfileID, nil
}

func (a *portalMediaProvisionerAdapter) resolveRootFolderID(ctx context.Context, mediaType string) int64 {
	if id := a.getMediaTypeSpecificRootFolder(ctx, mediaType); id != 0 {
		return id
	}
	if id := a.getGenericRootFolder(ctx, mediaType); id != 0 {
		return id
	}
	return a.getFirstAvailableRootFolder(ctx, mediaType)
}

func (a *portalMediaProvisionerAdapter) getMediaTypeSpecificRootFolder(ctx context.Context, mediaType string) int64 {
	settingKey := "requests_default_root_folder_id"
	switch mediaType {
	case "tv":
		settingKey = "requests_default_tv_root_folder_id"
	case mediaTypeMovie:
		settingKey = "requests_default_movie_root_folder_id"
	}

	setting, err := a.queries.GetSetting(ctx, settingKey)
	if err != nil || setting.Value == "" {
		return 0
	}
	v, parseErr := strconv.ParseInt(setting.Value, 10, 64)
	if parseErr != nil {
		return 0
	}
	return v
}

func (a *portalMediaProvisionerAdapter) getGenericRootFolder(ctx context.Context, mediaType string) int64 {
	setting, err := a.queries.GetSetting(ctx, "requests_default_root_folder_id")
	if err != nil || setting.Value == "" {
		return 0
	}
	v, parseErr := strconv.ParseInt(setting.Value, 10, 64)
	if parseErr != nil {
		return 0
	}
	rf, rfErr := a.queries.GetRootFolder(ctx, v)
	if rfErr != nil || rf.MediaType != mediaType {
		return 0
	}
	return v
}

func (a *portalMediaProvisionerAdapter) getFirstAvailableRootFolder(ctx context.Context, mediaType string) int64 {
	rootFolders, err := a.queries.ListRootFoldersByMediaType(ctx, mediaType)
	if err != nil || len(rootFolders) == 0 {
		return 0
	}
	return rootFolders[0].ID
}

func (a *portalMediaProvisionerAdapter) SetDB(db *sql.DB) {
	a.queries = sqlc.New(db)
}
