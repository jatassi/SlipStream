package quality

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/database/sqlc"
)

var (
	ErrProfileNotFound = errors.New("quality profile not found")
	ErrProfileInUse    = errors.New("quality profile is in use")
	ErrInvalidProfile  = errors.New("invalid quality profile")
)

// Service provides quality profile operations.
type Service struct {
	queries *sqlc.Queries
	logger  zerolog.Logger
}

// NewService creates a new quality profile service.
func NewService(db *sql.DB, logger zerolog.Logger) *Service {
	return &Service{
		queries: sqlc.New(db),
		logger:  logger.With().Str("component", "quality").Logger(),
	}
}

// Get retrieves a quality profile by ID.
func (s *Service) Get(ctx context.Context, id int64) (*Profile, error) {
	row, err := s.queries.GetQualityProfile(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrProfileNotFound
		}
		return nil, fmt.Errorf("failed to get quality profile: %w", err)
	}
	return s.rowToProfile(row)
}

// GetByName retrieves a quality profile by name.
func (s *Service) GetByName(ctx context.Context, name string) (*Profile, error) {
	row, err := s.queries.GetQualityProfileByName(ctx, name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrProfileNotFound
		}
		return nil, fmt.Errorf("failed to get quality profile: %w", err)
	}
	return s.rowToProfile(row)
}

// List returns all quality profiles.
func (s *Service) List(ctx context.Context) ([]*Profile, error) {
	rows, err := s.queries.ListQualityProfiles(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list quality profiles: %w", err)
	}

	profiles := make([]*Profile, 0, len(rows))
	for _, row := range rows {
		p, err := s.rowToProfile(row)
		if err != nil {
			s.logger.Warn().Err(err).Int64("id", row.ID).Msg("Failed to parse quality profile")
			continue
		}
		profiles = append(profiles, p)
	}
	return profiles, nil
}

// Create creates a new quality profile.
func (s *Service) Create(ctx context.Context, input CreateProfileInput) (*Profile, error) {
	if input.Name == "" {
		return nil, ErrInvalidProfile
	}

	itemsJSON, err := SerializeItems(input.Items)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize items: %w", err)
	}

	row, err := s.queries.CreateQualityProfile(ctx, sqlc.CreateQualityProfileParams{
		Name:   input.Name,
		Cutoff: int64(input.Cutoff),
		Items:  itemsJSON,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create quality profile: %w", err)
	}

	s.logger.Info().Int64("id", row.ID).Str("name", input.Name).Msg("Created quality profile")
	return s.rowToProfile(row)
}

// Update updates an existing quality profile.
func (s *Service) Update(ctx context.Context, id int64, input UpdateProfileInput) (*Profile, error) {
	if input.Name == "" {
		return nil, ErrInvalidProfile
	}

	itemsJSON, err := SerializeItems(input.Items)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize items: %w", err)
	}

	row, err := s.queries.UpdateQualityProfile(ctx, sqlc.UpdateQualityProfileParams{
		ID:     id,
		Name:   input.Name,
		Cutoff: int64(input.Cutoff),
		Items:  itemsJSON,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrProfileNotFound
		}
		return nil, fmt.Errorf("failed to update quality profile: %w", err)
	}

	s.logger.Info().Int64("id", id).Str("name", input.Name).Msg("Updated quality profile")
	return s.rowToProfile(row)
}

// Delete deletes a quality profile.
func (s *Service) Delete(ctx context.Context, id int64) error {
	// Check if profile is in use
	movieCount, err := s.queries.CountMoviesUsingProfile(ctx, sql.NullInt64{Int64: id, Valid: true})
	if err != nil {
		return fmt.Errorf("failed to check movie usage: %w", err)
	}
	if movieCount > 0 {
		return ErrProfileInUse
	}

	seriesCount, err := s.queries.CountSeriesUsingProfile(ctx, sql.NullInt64{Int64: id, Valid: true})
	if err != nil {
		return fmt.Errorf("failed to check series usage: %w", err)
	}
	if seriesCount > 0 {
		return ErrProfileInUse
	}

	if err := s.queries.DeleteQualityProfile(ctx, id); err != nil {
		return fmt.Errorf("failed to delete quality profile: %w", err)
	}

	s.logger.Info().Int64("id", id).Msg("Deleted quality profile")
	return nil
}

// GetQualities returns the list of predefined qualities.
func (s *Service) GetQualities() []Quality {
	return PredefinedQualities
}

// EnsureDefaults creates default profiles if none exist.
func (s *Service) EnsureDefaults(ctx context.Context) error {
	profiles, err := s.List(ctx)
	if err != nil {
		return err
	}

	if len(profiles) > 0 {
		return nil // Already have profiles
	}

	// Create default profiles
	defaults := []Profile{
		DefaultProfile(),
		HD1080pProfile(),
		Ultra4KProfile(),
	}

	for _, p := range defaults {
		_, err := s.Create(ctx, CreateProfileInput{
			Name:   p.Name,
			Cutoff: p.Cutoff,
			Items:  p.Items,
		})
		if err != nil {
			s.logger.Warn().Err(err).Str("name", p.Name).Msg("Failed to create default profile")
		}
	}

	s.logger.Info().Msg("Created default quality profiles")
	return nil
}

// rowToProfile converts a database row to a Profile.
func (s *Service) rowToProfile(row *sqlc.QualityProfile) (*Profile, error) {
	items, err := DeserializeItems(row.Items)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize items: %w", err)
	}

	p := &Profile{
		ID:     row.ID,
		Name:   row.Name,
		Cutoff: int(row.Cutoff),
		Items:  items,
	}

	if row.CreatedAt.Valid {
		p.CreatedAt = row.CreatedAt.Time
	}
	if row.UpdatedAt.Valid {
		p.UpdatedAt = row.UpdatedAt.Time
	}

	return p, nil
}
