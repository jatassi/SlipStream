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

// ImportDecisionCleaner clears cached import decisions when quality profiles change.
type ImportDecisionCleaner interface {
	ClearDecisionsForProfile(ctx context.Context, profileID int64) error
}

// Service provides quality profile operations.
type Service struct {
	queries               *sqlc.Queries
	logger                zerolog.Logger
	importDecisionCleaner ImportDecisionCleaner
}

// NewService creates a new quality profile service.
func NewService(db *sql.DB, logger zerolog.Logger) *Service {
	return &Service{
		queries: sqlc.New(db),
		logger:  logger.With().Str("component", "quality").Logger(),
	}
}

// SetImportDecisionCleaner sets the cleaner that invalidates import decisions when profiles change.
func (s *Service) SetImportDecisionCleaner(c ImportDecisionCleaner) {
	s.importDecisionCleaner = c
}

// SetDB updates the database connection used by this service.
// This is called when switching between production and development databases.
func (s *Service) SetDB(db *sql.DB) {
	s.queries = sqlc.New(db)
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

	hdrJSON, err := SerializeAttributeSettings(input.HDRSettings)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize HDR settings: %w", err)
	}

	videoCodecJSON, err := SerializeAttributeSettings(input.VideoCodecSettings)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize video codec settings: %w", err)
	}

	audioCodecJSON, err := SerializeAttributeSettings(input.AudioCodecSettings)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize audio codec settings: %w", err)
	}

	audioChannelJSON, err := SerializeAttributeSettings(input.AudioChannelSettings)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize audio channel settings: %w", err)
	}

	// Default to true if not specified
	upgradesEnabled := int64(1)
	if input.UpgradesEnabled != nil && !*input.UpgradesEnabled {
		upgradesEnabled = 0
	}

	allowAutoApprove := int64(0)
	if input.AllowAutoApprove {
		allowAutoApprove = 1
	}

	row, err := s.queries.CreateQualityProfile(ctx, sqlc.CreateQualityProfileParams{
		Name:                 input.Name,
		Cutoff:               int64(input.Cutoff),
		Items:                itemsJSON,
		HdrSettings:          hdrJSON,
		VideoCodecSettings:   videoCodecJSON,
		AudioCodecSettings:   audioCodecJSON,
		AudioChannelSettings: audioChannelJSON,
		UpgradesEnabled:      upgradesEnabled,
		AllowAutoApprove:     allowAutoApprove,
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

	hdrJSON, err := SerializeAttributeSettings(input.HDRSettings)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize HDR settings: %w", err)
	}

	videoCodecJSON, err := SerializeAttributeSettings(input.VideoCodecSettings)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize video codec settings: %w", err)
	}

	audioCodecJSON, err := SerializeAttributeSettings(input.AudioCodecSettings)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize audio codec settings: %w", err)
	}

	audioChannelJSON, err := SerializeAttributeSettings(input.AudioChannelSettings)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize audio channel settings: %w", err)
	}

	upgradesEnabled := int64(0)
	if input.UpgradesEnabled {
		upgradesEnabled = 1
	}

	allowAutoApprove := int64(0)
	if input.AllowAutoApprove {
		allowAutoApprove = 1
	}

	row, err := s.queries.UpdateQualityProfile(ctx, sqlc.UpdateQualityProfileParams{
		ID:                   id,
		Name:                 input.Name,
		Cutoff:               int64(input.Cutoff),
		Items:                itemsJSON,
		HdrSettings:          hdrJSON,
		VideoCodecSettings:   videoCodecJSON,
		AudioCodecSettings:   audioCodecJSON,
		AudioChannelSettings: audioChannelJSON,
		UpgradesEnabled:      upgradesEnabled,
		AllowAutoApprove:     allowAutoApprove,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrProfileNotFound
		}
		return nil, fmt.Errorf("failed to update quality profile: %w", err)
	}

	s.logger.Info().Int64("id", id).Str("name", input.Name).Msg("Updated quality profile")

	// Invalidate import decisions that used this profile — files may now be eligible for import
	if s.importDecisionCleaner != nil {
		if err := s.importDecisionCleaner.ClearDecisionsForProfile(ctx, id); err != nil {
			s.logger.Warn().Err(err).Int64("profileId", id).Msg("Failed to clear import decisions for profile")
		}
	}

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
		upgradesEnabled := p.UpgradesEnabled
		_, err := s.Create(ctx, CreateProfileInput{
			Name:            p.Name,
			Cutoff:          p.Cutoff,
			UpgradesEnabled: &upgradesEnabled,
			Items:           p.Items,
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

	hdrSettings, err := DeserializeAttributeSettings(row.HdrSettings)
	if err != nil {
		s.logger.Warn().Err(err).Int64("id", row.ID).Msg("Failed to deserialize HDR settings, using defaults")
		hdrSettings = DefaultAttributeSettings()
	}

	videoCodecSettings, err := DeserializeAttributeSettings(row.VideoCodecSettings)
	if err != nil {
		s.logger.Warn().Err(err).Int64("id", row.ID).Msg("Failed to deserialize video codec settings, using defaults")
		videoCodecSettings = DefaultAttributeSettings()
	}

	audioCodecSettings, err := DeserializeAttributeSettings(row.AudioCodecSettings)
	if err != nil {
		s.logger.Warn().Err(err).Int64("id", row.ID).Msg("Failed to deserialize audio codec settings, using defaults")
		audioCodecSettings = DefaultAttributeSettings()
	}

	audioChannelSettings, err := DeserializeAttributeSettings(row.AudioChannelSettings)
	if err != nil {
		s.logger.Warn().Err(err).Int64("id", row.ID).Msg("Failed to deserialize audio channel settings, using defaults")
		audioChannelSettings = DefaultAttributeSettings()
	}

	p := &Profile{
		ID:                   row.ID,
		Name:                 row.Name,
		Cutoff:               int(row.Cutoff),
		UpgradesEnabled:      row.UpgradesEnabled == 1,
		AllowAutoApprove:     row.AllowAutoApprove == 1,
		Items:                items,
		HDRSettings:          hdrSettings,
		VideoCodecSettings:   videoCodecSettings,
		AudioCodecSettings:   audioCodecSettings,
		AudioChannelSettings: audioChannelSettings,
	}

	if row.CreatedAt.Valid {
		p.CreatedAt = row.CreatedAt.Time
	}
	if row.UpdatedAt.Valid {
		p.UpdatedAt = row.UpdatedAt.Time
	}

	return p, nil
}

// RecalculateStatusForProfile recalculates status for all movies and episodes using a profile.
// Called when a profile's cutoff or upgrades_enabled changes.
func (s *Service) RecalculateStatusForProfile(ctx context.Context, profileID int64) (int, error) {
	profile, err := s.Get(ctx, profileID)
	if err != nil {
		return 0, err
	}

	updated := 0

	// Recalculate movies
	movieRows, err := s.queries.ListMoviesWithFilesForProfile(ctx, sql.NullInt64{Int64: profileID, Valid: true})
	if err != nil {
		return 0, fmt.Errorf("failed to list movies for profile: %w", err)
	}

	for _, row := range movieRows {
		if !row.CurrentQualityID.Valid {
			continue
		}
		newStatus := profile.StatusForQuality(int(row.CurrentQualityID.Int64))
		if newStatus != row.Status {
			if err := s.queries.UpdateMovieStatus(ctx, sqlc.UpdateMovieStatusParams{
				ID:     row.ID,
				Status: newStatus,
			}); err != nil {
				s.logger.Warn().Err(err).Int64("movieId", row.ID).Msg("Failed to recalculate movie status")
				continue
			}
			updated++
		}
	}

	// Recalculate episodes
	episodeRows, err := s.queries.ListEpisodesWithFilesForProfile(ctx, sql.NullInt64{Int64: profileID, Valid: true})
	if err != nil {
		return updated, fmt.Errorf("failed to list episodes for profile: %w", err)
	}

	for _, row := range episodeRows {
		if !row.CurrentQualityID.Valid {
			continue
		}
		newStatus := profile.StatusForQuality(int(row.CurrentQualityID.Int64))
		if newStatus != row.Status {
			if err := s.queries.UpdateEpisodeStatus(ctx, sqlc.UpdateEpisodeStatusParams{
				ID:     row.ID,
				Status: newStatus,
			}); err != nil {
				s.logger.Warn().Err(err).Int64("episodeId", row.ID).Msg("Failed to recalculate episode status")
				continue
			}
			updated++
		}
	}

	s.logger.Info().
		Int64("profileId", profileID).
		Int("updated", updated).
		Msg("Recalculated media status for profile change")

	return updated, nil
}
