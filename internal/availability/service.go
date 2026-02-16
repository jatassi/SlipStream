package availability

import (
	"context"
	"database/sql"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/database/sqlc"
)

// Service handles media availability tracking.
type Service struct {
	db      *sql.DB
	queries *sqlc.Queries
	logger  *zerolog.Logger
}

// NewService creates a new availability service.
func NewService(db *sql.DB, logger *zerolog.Logger) *Service {
	subLogger := logger.With().Str("component", "availability").Logger()
	return &Service{
		db:      db,
		queries: sqlc.New(db),
		logger:  &subLogger,
	}
}

// SetDB updates the database connection used by this service.
func (s *Service) SetDB(db *sql.DB) {
	s.db = db
	s.queries = sqlc.New(db)
}

// RefreshAll transitions unreleased movies and episodes to missing once their release/air date has passed.
func (s *Service) RefreshAll(ctx context.Context) error {
	s.logger.Info().Msg("Starting status refresh for all media")

	moviesUpdated, err := s.RefreshMovies(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to refresh movie status")
		return err
	}

	episodesUpdated, err := s.RefreshEpisodes(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to refresh episode status")
		return err
	}

	s.logger.Info().
		Int64("movies", moviesUpdated).
		Int64("episodes", episodesUpdated).
		Msg("Status refresh completed")

	return nil
}

// RefreshMovies transitions unreleased movies to missing where their release date has passed.
func (s *Service) RefreshMovies(ctx context.Context) (int64, error) {
	result, err := s.queries.UpdateUnreleasedMoviesToMissing(ctx)
	if err != nil {
		return 0, err
	}
	rowsAffected, _ := result.RowsAffected()
	return rowsAffected, nil
}

// RefreshEpisodes transitions unreleased episodes to missing where their air date has passed.
// Uses two queries: precise datetime comparison for episodes with actual air times,
// and date-only comparison for episodes without air times (air time fallback).
func (s *Service) RefreshEpisodes(ctx context.Context) (int64, error) {
	// Episodes with actual air time: use datetime comparison
	result1, err := s.queries.UpdateUnreleasedEpisodesToMissing(ctx)
	if err != nil {
		return 0, err
	}
	rows1, _ := result1.RowsAffected()

	// Episodes with date-only air date: use date comparison (same as movies)
	result2, err := s.queries.UpdateUnreleasedEpisodesToMissingDateOnly(ctx)
	if err != nil {
		return rows1, err
	}
	rows2, _ := result2.RowsAffected()

	return rows1 + rows2, nil
}
