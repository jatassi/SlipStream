package availability

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/database/sqlc"
)

// Service handles media availability tracking.
type Service struct {
	db      *sql.DB
	queries *sqlc.Queries
	logger  zerolog.Logger
}

// NewService creates a new availability service.
func NewService(db *sql.DB, logger zerolog.Logger) *Service {
	return &Service{
		db:      db,
		queries: sqlc.New(db),
		logger:  logger.With().Str("component", "availability").Logger(),
	}
}

// RefreshAll updates availability for all media types.
// Order matters: episodes -> seasons -> series, and movies independently.
func (s *Service) RefreshAll(ctx context.Context) error {
	s.logger.Info().Msg("Starting availability refresh for all media")

	// 1. Update movies by release_date
	moviesUpdated, err := s.RefreshMovies(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to refresh movie availability")
		return err
	}

	// 2. Update episodes by air_date
	episodesUpdated, err := s.RefreshEpisodes(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to refresh episode availability")
		return err
	}

	// 3. Update seasons from episodes
	seasonsUpdated, err := s.RefreshSeasons(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to refresh season availability")
		return err
	}

	// 4. Update series from seasons
	seriesUpdated, err := s.RefreshSeries(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to refresh series availability")
		return err
	}

	// 5. Update availability status text for all media
	if err := s.RefreshAllAvailabilityStatus(ctx); err != nil {
		s.logger.Error().Err(err).Msg("Failed to refresh availability status text")
		return err
	}

	s.logger.Info().
		Int64("movies", moviesUpdated).
		Int64("episodes", episodesUpdated).
		Int64("seasons", seasonsUpdated).
		Int64("series", seriesUpdated).
		Msg("Availability refresh completed")

	return nil
}

// RefreshMovies updates released flag for all movies based on release_date.
func (s *Service) RefreshMovies(ctx context.Context) (int64, error) {
	result, err := s.queries.UpdateMoviesReleasedByDate(ctx)
	if err != nil {
		return 0, err
	}
	rowsAffected, _ := result.RowsAffected()
	return rowsAffected, nil
}

// RefreshEpisodes updates released flag for all episodes based on air_date.
func (s *Service) RefreshEpisodes(ctx context.Context) (int64, error) {
	result, err := s.queries.UpdateEpisodesReleasedByDate(ctx)
	if err != nil {
		return 0, err
	}
	rowsAffected, _ := result.RowsAffected()
	return rowsAffected, nil
}

// RefreshSeasons updates released flag for all seasons based on episode availability.
func (s *Service) RefreshSeasons(ctx context.Context) (int64, error) {
	result, err := s.queries.UpdateAllSeasonsReleased(ctx)
	if err != nil {
		return 0, err
	}
	rowsAffected, _ := result.RowsAffected()
	return rowsAffected, nil
}

// RefreshSeries updates released flag for all series based on season availability.
func (s *Service) RefreshSeries(ctx context.Context) (int64, error) {
	result, err := s.queries.UpdateAllSeriesReleased(ctx)
	if err != nil {
		return 0, err
	}
	rowsAffected, _ := result.RowsAffected()
	return rowsAffected, nil
}

// SetMovieAvailability sets availability for a single movie based on its release date.
func (s *Service) SetMovieAvailability(ctx context.Context, movieID int64, releaseDate *time.Time) error {
	released := int64(0)
	if releaseDate != nil && !releaseDate.After(time.Now()) {
		released = 1
	}
	return s.queries.UpdateMovieReleased(ctx, sqlc.UpdateMovieReleasedParams{
		Released: released,
		ID:       movieID,
	})
}

// SetEpisodeAvailability sets availability for a single episode based on its air date.
func (s *Service) SetEpisodeAvailability(ctx context.Context, episodeID int64, airDate *time.Time) error {
	released := int64(0)
	if airDate != nil && !airDate.After(time.Now()) {
		released = 1
	}
	return s.queries.UpdateEpisodeReleased(ctx, sqlc.UpdateEpisodeReleasedParams{
		Released: released,
		ID:       episodeID,
	})
}

// RecalculateSeasonAvailability recalculates availability for a specific season.
func (s *Service) RecalculateSeasonAvailability(ctx context.Context, seasonID int64) error {
	return s.queries.UpdateSeasonReleasedFromEpisodes(ctx, seasonID)
}

// RecalculateSeriesAvailability recalculates availability for a specific series.
func (s *Service) RecalculateSeriesAvailability(ctx context.Context, seriesID int64) error {
	return s.queries.UpdateSeriesReleasedFromSeasons(ctx, seriesID)
}

// RecalculateSeriesFullAvailability recalculates availability for all levels of a series.
// This updates episodes, then seasons, then the series itself.
func (s *Service) RecalculateSeriesFullAvailability(ctx context.Context, seriesID int64) error {
	// Get all seasons for this series
	seasons, err := s.queries.GetSeasonsBySeriesID(ctx, seriesID)
	if err != nil {
		return err
	}

	// Update each season's availability based on its episodes
	for _, season := range seasons {
		if err := s.queries.UpdateSeasonReleasedFromEpisodes(ctx, season.ID); err != nil {
			s.logger.Error().Err(err).Int64("seasonID", season.ID).Msg("Failed to update season availability")
		}
	}

	// Update the series availability based on its seasons
	return s.queries.UpdateSeriesReleasedFromSeasons(ctx, seriesID)
}

// IsReleased checks if a date is in the past (media is released).
func IsReleased(date *time.Time) bool {
	if date == nil {
		return false
	}
	return !date.After(time.Now())
}

// RefreshAllAvailabilityStatus updates the availability_status text for all movies and series.
func (s *Service) RefreshAllAvailabilityStatus(ctx context.Context) error {
	// Update all movies availability status (simple: Available/Unreleased)
	if _, err := s.queries.UpdateAllMoviesAvailabilityStatus(ctx); err != nil {
		s.logger.Error().Err(err).Msg("Failed to update movie availability status")
		return err
	}

	// Update all series availability status (requires calculation)
	seriesData, err := s.queries.ListAllSeriesForAvailability(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to list series for availability status update")
		return err
	}

	for _, data := range seriesData {
		status := s.calculateSeriesAvailabilityStatus(data)
		if err := s.queries.UpdateSeriesAvailabilityStatus(ctx, sqlc.UpdateSeriesAvailabilityStatusParams{
			AvailabilityStatus: status,
			ID:                 data.ID,
		}); err != nil {
			s.logger.Warn().Err(err).Int64("seriesID", data.ID).Msg("Failed to update series availability status")
		}
	}

	return nil
}

// UpdateSeriesAvailabilityStatus updates the availability_status for a single series.
func (s *Service) UpdateSeriesAvailabilityStatus(ctx context.Context, seriesID int64) error {
	data, err := s.queries.GetSeriesAvailabilityData(ctx, seriesID)
	if err != nil {
		return err
	}

	status := s.calculateSeriesAvailabilityStatusFromGetRow(data)
	return s.queries.UpdateSeriesAvailabilityStatus(ctx, sqlc.UpdateSeriesAvailabilityStatusParams{
		AvailabilityStatus: status,
		ID:                 seriesID,
	})
}

// UpdateMovieAvailabilityStatus updates the availability_status for a single movie.
func (s *Service) UpdateMovieAvailabilityStatus(ctx context.Context, movieID int64, released bool) error {
	status := "Unreleased"
	if released {
		status = "Available"
	}
	return s.queries.UpdateMovieAvailabilityStatus(ctx, sqlc.UpdateMovieAvailabilityStatusParams{
		AvailabilityStatus: status,
		ID:                 movieID,
	})
}

// calculateSeriesAvailabilityStatus determines the badge text for a series.
func (s *Service) calculateSeriesAvailabilityStatus(data *sqlc.ListAllSeriesForAvailabilityRow) string {
	// If fully released
	if data.Released == 1 {
		return "Available"
	}

	totalSeasons := data.TotalSeasons
	releasedSeasons := data.ReleasedSeasons

	// No seasons yet
	if totalSeasons == 0 {
		return "Unreleased"
	}

	// No seasons released
	if releasedSeasons == 0 {
		// Check if any episodes have aired in unreleased seasons (currently airing)
		if data.AiredEpsInUnreleasedSeasons > 0 {
			// Get the first unreleased season number
			if data.FirstUnreleasedSeason != nil {
				if seasonNum, ok := data.FirstUnreleasedSeason.(int64); ok {
					return fmt.Sprintf("Season %d Airing", seasonNum)
				}
			}
			return "Season 1 Airing"
		}
		return "Unreleased"
	}

	// Some seasons released, check if currently airing
	if data.AiredEpsInUnreleasedSeasons > 0 {
		if data.FirstUnreleasedSeason != nil {
			if seasonNum, ok := data.FirstUnreleasedSeason.(int64); ok {
				return fmt.Sprintf("Season %d Airing", seasonNum)
			}
		}
	}

	// Partial availability (some complete seasons but not all)
	if releasedSeasons < totalSeasons {
		if releasedSeasons == 1 {
			return "Season 1 Available"
		}
		return fmt.Sprintf("Seasons 1-%d Available", releasedSeasons)
	}

	// All seasons released (shouldn't reach here if data.Released is correct)
	return "Available"
}

// calculateSeriesAvailabilityStatusFromGetRow is same logic but for GetSeriesAvailabilityData result.
func (s *Service) calculateSeriesAvailabilityStatusFromGetRow(data *sqlc.GetSeriesAvailabilityDataRow) string {
	// If fully released
	if data.Released == 1 {
		return "Available"
	}

	totalSeasons := data.TotalSeasons
	releasedSeasons := data.ReleasedSeasons

	// No seasons yet
	if totalSeasons == 0 {
		return "Unreleased"
	}

	// No seasons released
	if releasedSeasons == 0 {
		// Check if any episodes have aired in unreleased seasons (currently airing)
		if data.AiredEpsInUnreleasedSeasons > 0 {
			// Get the first unreleased season number
			if data.FirstUnreleasedSeason != nil {
				if seasonNum, ok := data.FirstUnreleasedSeason.(int64); ok {
					return fmt.Sprintf("Season %d Airing", seasonNum)
				}
			}
			return "Season 1 Airing"
		}
		return "Unreleased"
	}

	// Some seasons released, check if currently airing
	if data.AiredEpsInUnreleasedSeasons > 0 {
		if data.FirstUnreleasedSeason != nil {
			if seasonNum, ok := data.FirstUnreleasedSeason.(int64); ok {
				return fmt.Sprintf("Season %d Airing", seasonNum)
			}
		}
	}

	// Partial availability (some complete seasons but not all)
	if releasedSeasons < totalSeasons {
		if releasedSeasons == 1 {
			return "Season 1 Available"
		}
		return fmt.Sprintf("Seasons 1-%d Available", releasedSeasons)
	}

	// All seasons released (shouldn't reach here if data.Released is correct)
	return "Available"
}
