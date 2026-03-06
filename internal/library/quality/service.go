package quality

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"

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

// QualityItemDef mirrors module.QualityItem for registration without importing the module package.
type QualityItemDef struct {
	ID         int
	Name       string
	Source     string
	Resolution int
	Weight     int
}

type moduleQualityRegistry struct {
	mu    sync.RWMutex
	items map[string][]QualityItemDef
}

// Service provides quality profile operations.
type Service struct {
	queries               *sqlc.Queries
	logger                *zerolog.Logger
	importDecisionCleaner ImportDecisionCleaner
	moduleQualities       moduleQualityRegistry
}

// NewService creates a new quality profile service.
func NewService(db *sql.DB, logger *zerolog.Logger) *Service {
	subLogger := logger.With().Str("component", "quality").Logger()
	return &Service{
		queries: sqlc.New(db),
		logger:  &subLogger,
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

// RegisterModuleQualities registers quality items for a module.
func (s *Service) RegisterModuleQualities(moduleType string, items []QualityItemDef) {
	s.moduleQualities.mu.Lock()
	defer s.moduleQualities.mu.Unlock()
	if s.moduleQualities.items == nil {
		s.moduleQualities.items = make(map[string][]QualityItemDef)
	}
	s.moduleQualities.items[moduleType] = items
	s.logger.Info().Str("moduleType", moduleType).Int("qualityCount", len(items)).
		Msg("Registered quality items for module")
}

// GetQualitiesForModule returns quality items for a specific module.
func (s *Service) GetQualitiesForModule(moduleType string) []Quality {
	s.moduleQualities.mu.RLock()
	defer s.moduleQualities.mu.RUnlock()
	items, ok := s.moduleQualities.items[moduleType]
	if !ok || len(items) == 0 {
		return PredefinedQualities
	}
	result := make([]Quality, len(items))
	for i, item := range items {
		result[i] = Quality(item)
	}
	return result
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

// GetByName retrieves a quality profile by name and module type.
func (s *Service) GetByName(ctx context.Context, name, moduleType string) (*Profile, error) {
	row, err := s.queries.GetQualityProfileByName(ctx, sqlc.GetQualityProfileByNameParams{
		Name:       name,
		ModuleType: moduleType,
	})
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

// ListByModule returns quality profiles for a specific module type.
func (s *Service) ListByModule(ctx context.Context, moduleType string) ([]*Profile, error) {
	rows, err := s.queries.ListQualityProfilesByModule(ctx, moduleType)
	if err != nil {
		return nil, fmt.Errorf("failed to list quality profiles for module %s: %w", moduleType, err)
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
func (s *Service) Create(ctx context.Context, input *CreateProfileInput) (*Profile, error) {
	if input.Name == "" {
		return nil, ErrInvalidProfile
	}
	if input.ModuleType == "" {
		return nil, fmt.Errorf("%w: moduleType is required", ErrInvalidProfile)
	}

	serialized, err := s.serializeProfileSettings(input.Items, input.HDRSettings, input.VideoCodecSettings, input.AudioCodecSettings, input.AudioChannelSettings)
	if err != nil {
		return nil, err
	}

	upgradesEnabled := input.UpgradesEnabled == nil || *input.UpgradesEnabled

	upgradeStrategy := string(input.UpgradeStrategy)
	if !IsValidUpgradeStrategy(upgradeStrategy) {
		upgradeStrategy = string(StrategyBalanced)
	}

	row, err := s.queries.CreateQualityProfile(ctx, sqlc.CreateQualityProfileParams{
		Name:                    input.Name,
		ModuleType:              input.ModuleType,
		Cutoff:                  int64(input.Cutoff),
		Items:                   serialized.items,
		HdrSettings:             sql.NullString{String: serialized.hdr, Valid: serialized.hdr != ""},
		VideoCodecSettings:      sql.NullString{String: serialized.videoCodec, Valid: serialized.videoCodec != ""},
		AudioCodecSettings:      sql.NullString{String: serialized.audioCodec, Valid: serialized.audioCodec != ""},
		AudioChannelSettings:    sql.NullString{String: serialized.audioChannel, Valid: serialized.audioChannel != ""},
		UpgradesEnabled:         upgradesEnabled,
		AllowAutoApprove:        input.AllowAutoApprove,
		UpgradeStrategy:         upgradeStrategy,
		CutoffOverridesStrategy: boolToInt64(input.CutoffOverridesStrategy),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create quality profile: %w", err)
	}

	s.logger.Info().Int64("id", row.ID).Str("name", input.Name).Msg("Created quality profile")
	return s.rowToProfile(row)
}

type serializedSettings struct {
	items, hdr, videoCodec, audioCodec, audioChannel string
}

func (s *Service) serializeProfileSettings(items []QualityItem, hdr, videoCodec, audioCodec, audioChannel AttributeSettings) (*serializedSettings, error) {
	itemsJSON, err := SerializeItems(items)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize items: %w", err)
	}
	hdrJSON, err := SerializeAttributeSettings(hdr)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize HDR settings: %w", err)
	}
	videoCodecJSON, err := SerializeAttributeSettings(videoCodec)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize video codec settings: %w", err)
	}
	audioCodecJSON, err := SerializeAttributeSettings(audioCodec)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize audio codec settings: %w", err)
	}
	audioChannelJSON, err := SerializeAttributeSettings(audioChannel)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize audio channel settings: %w", err)
	}
	return &serializedSettings{
		items:        itemsJSON,
		hdr:          hdrJSON,
		videoCodec:   videoCodecJSON,
		audioCodec:   audioCodecJSON,
		audioChannel: audioChannelJSON,
	}, nil
}

func boolToInt64(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

// Update updates an existing quality profile.
func (s *Service) Update(ctx context.Context, id int64, input *UpdateProfileInput) (*Profile, error) {
	if input.Name == "" {
		return nil, ErrInvalidProfile
	}

	serialized, err := s.serializeProfileSettings(input.Items, input.HDRSettings, input.VideoCodecSettings, input.AudioCodecSettings, input.AudioChannelSettings)
	if err != nil {
		return nil, err
	}

	upgradeStrategy := string(input.UpgradeStrategy)
	if !IsValidUpgradeStrategy(upgradeStrategy) {
		upgradeStrategy = string(StrategyBalanced)
	}

	row, err := s.queries.UpdateQualityProfile(ctx, sqlc.UpdateQualityProfileParams{
		ID:                      id,
		Name:                    input.Name,
		Cutoff:                  int64(input.Cutoff),
		Items:                   serialized.items,
		HdrSettings:             sql.NullString{String: serialized.hdr, Valid: serialized.hdr != ""},
		VideoCodecSettings:      sql.NullString{String: serialized.videoCodec, Valid: serialized.videoCodec != ""},
		AudioCodecSettings:      sql.NullString{String: serialized.audioCodec, Valid: serialized.audioCodec != ""},
		AudioChannelSettings:    sql.NullString{String: serialized.audioChannel, Valid: serialized.audioChannel != ""},
		UpgradesEnabled:         input.UpgradesEnabled,
		AllowAutoApprove:        input.AllowAutoApprove,
		UpgradeStrategy:         upgradeStrategy,
		CutoffOverridesStrategy: boolToInt64(input.CutoffOverridesStrategy),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrProfileNotFound
		}
		return nil, fmt.Errorf("failed to update quality profile: %w", err)
	}

	s.logger.Info().Int64("id", id).Str("name", input.Name).Msg("Updated quality profile")

	if s.importDecisionCleaner != nil {
		if err := s.importDecisionCleaner.ClearDecisionsForProfile(ctx, id); err != nil {
			s.logger.Warn().Err(err).Int64("profileId", id).Msg("Failed to clear import decisions for profile")
		}
	}

	return s.rowToProfile(row)
}

// Delete deletes a quality profile.
func (s *Service) Delete(ctx context.Context, id int64) error {
	profile, err := s.Get(ctx, id)
	if err != nil {
		return err
	}

	if err := s.checkProfileUsage(ctx, id, profile.ModuleType); err != nil {
		return err
	}

	if err := s.queries.DeleteQualityProfile(ctx, id); err != nil {
		return fmt.Errorf("failed to delete quality profile: %w", err)
	}

	s.logger.Info().Int64("id", id).Msg("Deleted quality profile")
	return nil
}

// checkProfileUsage verifies that a quality profile is not in use by any media or slots.
func (s *Service) checkProfileUsage(ctx context.Context, id int64, moduleType string) error {
	nullID := sql.NullInt64{Int64: id, Valid: true}

	switch moduleType {
	case "movie":
		count, err := s.queries.CountMoviesUsingProfile(ctx, nullID)
		if err != nil {
			return fmt.Errorf("failed to check movie usage: %w", err)
		}
		if count > 0 {
			return ErrProfileInUse
		}
	case "tv":
		count, err := s.queries.CountSeriesUsingProfile(ctx, nullID)
		if err != nil {
			return fmt.Errorf("failed to check series usage: %w", err)
		}
		if count > 0 {
			return ErrProfileInUse
		}
	}

	slotCount, err := s.queries.CountSlotsUsingProfile(ctx, nullID)
	if err != nil {
		return fmt.Errorf("failed to check slot usage: %w", err)
	}
	if slotCount > 0 {
		return ErrProfileInUse
	}
	return nil
}

// GetQualities returns the list of predefined qualities.
func (s *Service) GetQualities() []Quality {
	return PredefinedQualities
}

// EnsureDefaults creates default profiles if none exist for the given module type.
func (s *Service) EnsureDefaults(ctx context.Context, moduleType string) error {
	count, err := s.queries.CountQualityProfilesByModule(ctx, moduleType)
	if err != nil {
		return err
	}

	if count > 0 {
		return nil // Already have profiles for this module
	}

	// Create default profiles
	defaults := []Profile{
		DefaultProfile(),
		HD1080pProfile(),
		Ultra4KProfile(),
	}

	for i := range defaults {
		p := &defaults[i]
		upgradesEnabled := p.UpgradesEnabled
		_, err := s.Create(ctx, &CreateProfileInput{
			Name:            p.Name,
			ModuleType:      moduleType,
			Cutoff:          p.Cutoff,
			UpgradesEnabled: &upgradesEnabled,
			Items:           p.Items,
		})
		if err != nil {
			s.logger.Warn().Err(err).Str("name", p.Name).Msg("Failed to create default profile")
		}
	}

	s.logger.Info().Str("moduleType", moduleType).Msg("Created default quality profiles")
	return nil
}

// rowToProfile converts a database row to a Profile.
func (s *Service) rowToProfile(row *sqlc.QualityProfile) (*Profile, error) {
	items, err := DeserializeItems(row.Items)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize items: %w", err)
	}

	hdrSettings, err := DeserializeAttributeSettings(row.HdrSettings.String)
	if err != nil {
		s.logger.Warn().Err(err).Int64("id", row.ID).Msg("Failed to deserialize HDR settings, using defaults")
		hdrSettings = DefaultAttributeSettings()
	}

	videoCodecSettings, err := DeserializeAttributeSettings(row.VideoCodecSettings.String)
	if err != nil {
		s.logger.Warn().Err(err).Int64("id", row.ID).Msg("Failed to deserialize video codec settings, using defaults")
		videoCodecSettings = DefaultAttributeSettings()
	}

	audioCodecSettings, err := DeserializeAttributeSettings(row.AudioCodecSettings.String)
	if err != nil {
		s.logger.Warn().Err(err).Int64("id", row.ID).Msg("Failed to deserialize audio codec settings, using defaults")
		audioCodecSettings = DefaultAttributeSettings()
	}

	audioChannelSettings, err := DeserializeAttributeSettings(row.AudioChannelSettings.String)
	if err != nil {
		s.logger.Warn().Err(err).Int64("id", row.ID).Msg("Failed to deserialize audio channel settings, using defaults")
		audioChannelSettings = DefaultAttributeSettings()
	}

	upgradeStrategy := UpgradeStrategy(row.UpgradeStrategy)
	if !IsValidUpgradeStrategy(string(upgradeStrategy)) {
		upgradeStrategy = StrategyBalanced
	}

	p := &Profile{
		ID:                      row.ID,
		Name:                    row.Name,
		ModuleType:              row.ModuleType,
		Cutoff:                  int(row.Cutoff),
		UpgradesEnabled:         row.UpgradesEnabled,
		UpgradeStrategy:         upgradeStrategy,
		CutoffOverridesStrategy: row.CutoffOverridesStrategy == 1,
		AllowAutoApprove:        row.AllowAutoApprove,
		Items:                   items,
		HDRSettings:             hdrSettings,
		VideoCodecSettings:      videoCodecSettings,
		AudioCodecSettings:      audioCodecSettings,
		AudioChannelSettings:    audioChannelSettings,
	}

	if row.CreatedAt.Valid {
		p.CreatedAt = row.CreatedAt.Time
	}
	if row.UpdatedAt.Valid {
		p.UpdatedAt = row.UpdatedAt.Time
	}

	return p, nil
}

// RecalculateStatusForProfile recalculates status for media using a profile, scoped by module type.
// Called when a profile's cutoff or upgrades_enabled changes.
func (s *Service) RecalculateStatusForProfile(ctx context.Context, profileID int64) (int, error) {
	profile, err := s.Get(ctx, profileID)
	if err != nil {
		return 0, err
	}

	switch profile.ModuleType {
	case "movie":
		updated, err := s.recalculateMovieStatuses(ctx, profileID, profile)
		if err != nil {
			return 0, err
		}
		s.logger.Info().Int64("profileId", profileID).Int("updated", updated).
			Msg("Recalculated movie status for profile change")
		return updated, nil
	case "tv":
		updated, err := s.recalculateEpisodeStatuses(ctx, profileID, profile)
		if err != nil {
			return 0, err
		}
		s.logger.Info().Int64("profileId", profileID).Int("updated", updated).
			Msg("Recalculated episode status for profile change")
		return updated, nil
	default:
		movieUpdated, err := s.recalculateMovieStatuses(ctx, profileID, profile)
		if err != nil {
			return 0, err
		}
		episodeUpdated, err := s.recalculateEpisodeStatuses(ctx, profileID, profile)
		if err != nil {
			return movieUpdated, err
		}
		updated := movieUpdated + episodeUpdated
		s.logger.Info().Int64("profileId", profileID).Int("updated", updated).
			Msg("Recalculated media status for profile change")
		return updated, nil
	}
}

func (s *Service) recalculateMovieStatuses(ctx context.Context, profileID int64, profile *Profile) (int, error) {
	movieRows, err := s.queries.ListMoviesWithFilesForProfile(ctx, sql.NullInt64{Int64: profileID, Valid: true})
	if err != nil {
		return 0, fmt.Errorf("failed to list movies for profile: %w", err)
	}

	updated := 0
	for _, row := range movieRows {
		if !row.CurrentQualityID.Valid {
			continue
		}
		newStatus := profile.StatusForQuality(int(row.CurrentQualityID.Int64))
		if newStatus == row.Status {
			continue
		}
		if err := s.queries.UpdateMovieStatus(ctx, sqlc.UpdateMovieStatusParams{
			ID:     row.ID,
			Status: newStatus,
		}); err != nil {
			s.logger.Warn().Err(err).Int64("movieId", row.ID).Msg("Failed to recalculate movie status")
			continue
		}
		updated++
	}
	return updated, nil
}

func (s *Service) recalculateEpisodeStatuses(ctx context.Context, profileID int64, profile *Profile) (int, error) {
	episodeRows, err := s.queries.ListEpisodesWithFilesForProfile(ctx, sql.NullInt64{Int64: profileID, Valid: true})
	if err != nil {
		return 0, fmt.Errorf("failed to list episodes for profile: %w", err)
	}

	updated := 0
	for _, row := range episodeRows {
		if !row.CurrentQualityID.Valid {
			continue
		}
		newStatus := profile.StatusForQuality(int(row.CurrentQualityID.Int64))
		if newStatus == row.Status {
			continue
		}
		if err := s.queries.UpdateEpisodeStatus(ctx, sqlc.UpdateEpisodeStatusParams{
			ID:     row.ID,
			Status: newStatus,
		}); err != nil {
			s.logger.Warn().Err(err).Int64("episodeId", row.ID).Msg("Failed to recalculate episode status")
			continue
		}
		updated++
	}
	return updated, nil
}
