package provisioner

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

const mediaTypeMovie = "movie"

// Service implements requests.MediaProvisioner to find or create media in library.
type Service struct {
	queries        *sqlc.Queries
	movieService   *movies.Service
	tvService      *tv.Service
	libraryManager *librarymanager.Service
	logger         *zerolog.Logger
}

func NewService(
	queries *sqlc.Queries,
	movieService *movies.Service,
	tvService *tv.Service,
	libraryManager *librarymanager.Service,
	logger *zerolog.Logger,
) *Service {
	return &Service{
		queries:        queries,
		movieService:   movieService,
		tvService:      tvService,
		libraryManager: libraryManager,
		logger:         logger,
	}
}

func (s *Service) SetDB(db *sql.DB) {
	s.queries = sqlc.New(db)
}

func (s *Service) EnsureMovieInLibrary(ctx context.Context, input *requests.MediaProvisionInput) (int64, error) {
	// First, try to find existing movie by TMDB ID
	existing, err := s.movieService.GetByTmdbID(ctx, int(input.TmdbID))
	if err == nil && existing != nil {
		s.logger.Debug().Int64("tmdbID", input.TmdbID).Int64("movieID", existing.ID).Msg("found existing movie in library")
		return existing.ID, nil
	}

	// Movie not in library - get default settings and create it
	rootFolderID, qualityProfileID, err := s.getDefaultSettings(ctx, mediaTypeMovie)
	if err != nil {
		return 0, fmt.Errorf("failed to get default settings: %w", err)
	}

	// Use user's quality profile if provided
	if input.QualityProfileID != nil {
		qualityProfileID = *input.QualityProfileID
		s.logger.Debug().Int64("qualityProfileID", qualityProfileID).Msg("using user's assigned quality profile for movie")
	}

	movie, err := s.movieService.Create(ctx, &movies.CreateMovieInput{
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

	s.logger.Info().Int64("tmdbID", input.TmdbID).Int64("movieID", movie.ID).Str("title", input.Title).Msg("created movie in library from request")

	// Fetch metadata including artwork and details
	if s.libraryManager != nil {
		if _, err := s.libraryManager.RefreshMovieMetadata(ctx, movie.ID); err != nil {
			s.logger.Warn().Err(err).Int64("movieID", movie.ID).Msg("failed to refresh movie metadata, movie created without full details")
		} else {
			s.logger.Info().Int64("movieID", movie.ID).Msg("fetched movie metadata and artwork")
		}
	}

	return movie.ID, nil
}

func (s *Service) EnsureSeriesInLibrary(ctx context.Context, input *requests.MediaProvisionInput) (int64, error) {
	existing, err := s.tvService.GetSeriesByTvdbID(ctx, int(input.TvdbID))
	if err == nil && existing != nil {
		return s.applyMonitoringToExistingSeries(ctx, existing.ID, input)
	}

	// Series not in library - get default settings and create it
	rootFolderID, qualityProfileID, err := s.getDefaultSettings(ctx, "tv")
	if err != nil {
		return 0, fmt.Errorf("failed to get default settings: %w", err)
	}

	// Use user's quality profile if provided
	if input.QualityProfileID != nil {
		qualityProfileID = *input.QualityProfileID
		s.logger.Debug().Int64("qualityProfileID", qualityProfileID).Msg("using user's assigned quality profile for series")
	}

	series, err := s.tvService.CreateSeries(ctx, &tv.CreateSeriesInput{
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

	s.logger.Info().Int64("tvdbID", input.TvdbID).Int64("seriesID", series.ID).Str("title", input.Title).Msg("created series in library from request")

	// Fetch metadata including seasons and episodes
	if s.libraryManager != nil {
		if _, err := s.libraryManager.RefreshSeriesMetadata(ctx, series.ID); err != nil {
			s.logger.Warn().Err(err).Int64("seriesID", series.ID).Msg("failed to refresh series metadata, series created without episodes")
		} else {
			s.logger.Info().Int64("seriesID", series.ID).Msg("fetched series metadata with seasons and episodes")
		}
	}

	if err := s.applyPortalRequestMonitoring(ctx, series.ID, input); err != nil {
		s.logger.Warn().Err(err).Int64("seriesID", series.ID).Msg("failed to apply portal request monitoring")
	}

	return series.ID, nil
}

// applyPortalRequestMonitoring handles all monitoring scenarios for new series from portal requests.
func (s *Service) applyPortalRequestMonitoring(ctx context.Context, seriesID int64, input *requests.MediaProvisionInput) error {
	if len(input.RequestedSeasons) > 0 {
		if err := s.applyRequestedSeasonsMonitoring(ctx, seriesID, input.RequestedSeasons); err != nil {
			return err
		}
		if input.MonitorFuture {
			s.applyMonitorFuture(ctx, seriesID)
		}
	} else if input.MonitorFuture {
		if err := s.tvService.BulkMonitor(ctx, seriesID, tv.BulkMonitorInput{
			MonitorType:     tv.MonitorTypeFuture,
			IncludeSpecials: false,
		}); err != nil {
			return fmt.Errorf("failed to apply monitor future: %w", err)
		}
	}

	// Always unmonitor specials (Season 0) as a safety net
	s.unmonitorSpecials(ctx, seriesID)

	return nil
}

func (s *Service) unmonitorSpecials(ctx context.Context, seriesID int64) {
	if _, err := s.tvService.UpdateSeasonMonitored(ctx, seriesID, 0, false); err != nil {
		if !errors.Is(err, tv.ErrSeasonNotFound) {
			s.logger.Warn().Err(err).Int64("seriesID", seriesID).Msg("failed to unmonitor specials")
		}
	}
}

func (s *Service) applyMonitoringToExistingSeries(ctx context.Context, seriesID int64, input *requests.MediaProvisionInput) (int64, error) {
	s.logger.Debug().Int64("tvdbID", input.TvdbID).Int64("seriesID", seriesID).Msg("found existing series in library")
	if len(input.RequestedSeasons) > 0 {
		s.applyRequestedSeasonsMonitoringAdditive(ctx, seriesID, input.RequestedSeasons)
	}
	if input.MonitorFuture {
		s.applyMonitorFuture(ctx, seriesID)
	}
	return seriesID, nil
}

// applyRequestedSeasonsMonitoring unmonitors all seasons except the requested ones.
func (s *Service) applyRequestedSeasonsMonitoring(ctx context.Context, seriesID int64, requestedSeasons []int64) error {
	// Get all seasons for the series
	seasons, err := s.tvService.ListSeasons(ctx, seriesID)
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
			if _, err := s.tvService.UpdateSeasonMonitored(ctx, seriesID, season.SeasonNumber, shouldMonitor); err != nil {
				s.logger.Warn().
					Err(err).
					Int64("seriesID", seriesID).
					Int("seasonNumber", season.SeasonNumber).
					Bool("monitored", shouldMonitor).
					Msg("failed to update season monitoring")
			} else {
				s.logger.Debug().
					Int64("seriesID", seriesID).
					Int("seasonNumber", season.SeasonNumber).
					Bool("monitored", shouldMonitor).
					Msg("updated season monitoring")
			}
		}
	}

	s.logger.Info().
		Int64("seriesID", seriesID).
		Interface("requestedSeasons", requestedSeasons).
		Msg("applied requested seasons monitoring")

	return nil
}

func (s *Service) applyRequestedSeasonsMonitoringAdditive(ctx context.Context, seriesID int64, requestedSeasons []int64) {
	for _, sn := range requestedSeasons {
		if _, err := s.tvService.UpdateSeasonMonitored(ctx, seriesID, int(sn), true); err != nil {
			s.logger.Warn().Err(err).Int64("seriesID", seriesID).Int64("seasonNumber", sn).Msg("failed to monitor requested season")
		}
	}
}

func (s *Service) applyMonitorFuture(ctx context.Context, seriesID int64) {
	if err := s.queries.UpdateFutureEpisodesMonitored(ctx, sqlc.UpdateFutureEpisodesMonitoredParams{
		Monitored: true,
		SeriesID:  seriesID,
	}); err != nil {
		s.logger.Warn().Err(err).Int64("seriesID", seriesID).Msg("failed to monitor future episodes")
	}
	if err := s.queries.UpdateFutureSeasonsMonitored(ctx, sqlc.UpdateFutureSeasonsMonitoredParams{
		Monitored:  true,
		SeriesID:   seriesID,
		SeriesID_2: seriesID,
	}); err != nil {
		s.logger.Warn().Err(err).Int64("seriesID", seriesID).Msg("failed to monitor future seasons")
	}
}

func (s *Service) getDefaultSettings(ctx context.Context, mediaType string) (rootFolderID, qualityProfileID int64, err error) {
	rootFolderID = s.resolveRootFolderID(ctx, mediaType)
	if rootFolderID == 0 {
		return 0, 0, fmt.Errorf("no %s root folder configured - please configure a root folder for %s content", mediaType, mediaType)
	}

	profiles, err := s.queries.ListQualityProfiles(ctx)
	if err != nil || len(profiles) == 0 {
		return 0, 0, errors.New("no quality profile configured")
	}
	qualityProfileID = profiles[0].ID

	return rootFolderID, qualityProfileID, nil
}

func (s *Service) resolveRootFolderID(ctx context.Context, mediaType string) int64 {
	if id := s.getMediaTypeSpecificRootFolder(ctx, mediaType); id != 0 {
		return id
	}
	if id := s.getGenericRootFolder(ctx, mediaType); id != 0 {
		return id
	}
	return s.getFirstAvailableRootFolder(ctx, mediaType)
}

func (s *Service) getMediaTypeSpecificRootFolder(ctx context.Context, mediaType string) int64 {
	settingKey := "requests_default_root_folder_id"
	switch mediaType {
	case "tv":
		settingKey = "requests_default_tv_root_folder_id"
	case mediaTypeMovie:
		settingKey = "requests_default_movie_root_folder_id"
	}

	setting, err := s.queries.GetSetting(ctx, settingKey)
	if err != nil || setting.Value == "" {
		return 0
	}
	v, parseErr := strconv.ParseInt(setting.Value, 10, 64)
	if parseErr != nil {
		return 0
	}
	return v
}

func (s *Service) getGenericRootFolder(ctx context.Context, mediaType string) int64 {
	setting, err := s.queries.GetSetting(ctx, "requests_default_root_folder_id")
	if err != nil || setting.Value == "" {
		return 0
	}
	v, parseErr := strconv.ParseInt(setting.Value, 10, 64)
	if parseErr != nil {
		return 0
	}
	rf, rfErr := s.queries.GetRootFolder(ctx, v)
	if rfErr != nil || rf.MediaType != mediaType {
		return 0
	}
	return v
}

func (s *Service) getFirstAvailableRootFolder(ctx context.Context, mediaType string) int64 {
	rootFolders, err := s.queries.ListRootFoldersByMediaType(ctx, mediaType)
	if err != nil || len(rootFolders) == 0 {
		return 0
	}
	return rootFolders[0].ID
}
