package availability

import (
	"context"
	"database/sql"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/module"
)

// Service handles media availability tracking.
type Service struct {
	db       *sql.DB
	queries  *sqlc.Queries
	logger   *zerolog.Logger
	registry *module.Registry
}

// NewService creates a new availability service.
func NewService(db *sql.DB, registry *module.Registry, logger *zerolog.Logger) *Service {
	subLogger := logger.With().Str("component", "availability").Logger()
	return &Service{
		db:       db,
		queries:  sqlc.New(db),
		registry: registry,
		logger:   &subLogger,
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

	totalTransitioned := 0
	for _, mod := range s.registry.Enabled() {
		resolver, ok := mod.(module.ReleaseDateResolver)
		if !ok {
			continue
		}
		count, err := resolver.CheckReleaseDateTransitions(ctx)
		if err != nil {
			s.logger.Error().Err(err).Str("module", string(mod.ID())).Msg("Failed to check release date transitions")
			continue
		}
		totalTransitioned += count
	}
	s.logger.Info().Int("transitioned", totalTransitioned).Msg("Status refresh completed via modules")
	return nil
}
