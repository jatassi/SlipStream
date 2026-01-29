package availability

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strconv"
	"strings"
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

// SetDB updates the database connection used by this service.
func (s *Service) SetDB(db *sql.DB) {
	s.db = db
	s.queries = sqlc.New(db)
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

// calculateSeriesAvailabilityStatus determines the badge text for a series based on downloaded seasons.
func (s *Service) calculateSeriesAvailabilityStatus(data *sqlc.ListAllSeriesForAvailabilityRow) string {
	return formatSeriesAvailability(data.TotalSeasons, data.AvailableSeasons)
}

// calculateSeriesAvailabilityStatusFromGetRow is same logic but for GetSeriesAvailabilityData result.
func (s *Service) calculateSeriesAvailabilityStatusFromGetRow(data *sqlc.GetSeriesAvailabilityDataRow) string {
	return formatSeriesAvailability(data.TotalSeasons, data.AvailableSeasons)
}

// formatSeriesAvailability formats the availability status based on total and available seasons.
func formatSeriesAvailability(totalSeasons int64, availableSeasons string) string {
	if totalSeasons == 0 {
		return "Unavailable"
	}

	if availableSeasons == "" {
		return "Unavailable"
	}

	seasons := parseSeasonNumbers(availableSeasons)
	if len(seasons) == 0 {
		return "Unavailable"
	}

	if int64(len(seasons)) == totalSeasons {
		return "Available"
	}

	return formatSeasonRanges(seasons)
}

// parseSeasonNumbers parses a comma-separated string of season numbers into a sorted slice.
func parseSeasonNumbers(s string) []int {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	seasons := make([]int, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if n, err := strconv.Atoi(p); err == nil && n > 0 {
			seasons = append(seasons, n)
		}
	}
	sort.Ints(seasons)
	return seasons
}

// formatSeasonRanges formats a sorted slice of season numbers into a readable string.
// Examples:
//   - [3] → "Season 3 Available"
//   - [1,2,3] → "Seasons 1-3 Available"
//   - [1,2,3,5] → "Seasons 1-3, 5 Available"
//   - [1,2,4,5] → "Seasons 1-2, 4-5 Available"
//   - [1,3,5] → "Seasons 1, 3, 5 Available"
func formatSeasonRanges(seasons []int) string {
	if len(seasons) == 0 {
		return "Unavailable"
	}

	if len(seasons) == 1 {
		return fmt.Sprintf("Season %d Available", seasons[0])
	}

	// Build ranges
	type seasonRange struct {
		start, end int
	}
	var ranges []seasonRange
	start := seasons[0]
	end := seasons[0]

	for i := 1; i < len(seasons); i++ {
		if seasons[i] == end+1 {
			end = seasons[i]
		} else {
			ranges = append(ranges, seasonRange{start, end})
			start = seasons[i]
			end = seasons[i]
		}
	}
	ranges = append(ranges, seasonRange{start, end})

	// Format ranges
	var parts []string
	for _, r := range ranges {
		if r.start == r.end {
			parts = append(parts, strconv.Itoa(r.start))
		} else {
			parts = append(parts, fmt.Sprintf("%d-%d", r.start, r.end))
		}
	}

	return fmt.Sprintf("Seasons %s Available", strings.Join(parts, ", "))
}
