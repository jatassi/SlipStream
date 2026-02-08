package slots

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/library/quality"
)

var (
	ErrSlotNotFound           = errors.New("slot not found")
	ErrSlotNoQualityProfile   = errors.New("slot has no quality profile configured")
	ErrInvalidSlot            = errors.New("invalid slot configuration")
	ErrSlotHasFiles           = errors.New("slot has files assigned")
	ErrDryRunRequired         = errors.New("dry-run preview is required before enabling multi-version")
	ErrProfilesNotExclusive   = errors.New("assigned profiles are not mutually exclusive")
	ErrMissingProfileForSlot  = errors.New("enabled slot must have a quality profile assigned")
)

// FileDeleter deletes files from disk and database.
// Used when disabling a slot with the "delete" action.
type FileDeleter interface {
	DeleteFile(ctx context.Context, mediaType string, fileID int64) error
}

// RootFolderProvider provides root folder lookup operations.
type RootFolderProvider interface {
	Get(ctx context.Context, id int64) (*RootFolder, error)
}

// RootFolder represents a root folder for slot root folder lookups.
type RootFolder struct {
	ID        int64
	Path      string
	Name      string
	MediaType string
}

// Service provides version slot operations.
type Service struct {
	queries            *sqlc.Queries
	qualityService     *quality.Service
	rootFolderProvider RootFolderProvider
	logger             zerolog.Logger
	fileDeleter        FileDeleter
}

// SetFileDeleter sets the file deleter for slot disable operations.
// Req 12.2.2: Delete files when disabling a slot with delete action.
func (s *Service) SetFileDeleter(deleter FileDeleter) {
	s.fileDeleter = deleter
}

// SetRootFolderProvider sets the root folder provider for slot root folder lookups.
func (s *Service) SetRootFolderProvider(provider RootFolderProvider) {
	s.rootFolderProvider = provider
}

// NewService creates a new slot service.
func NewService(db *sql.DB, qualityService *quality.Service, logger zerolog.Logger) *Service {
	return &Service{
		queries:        sqlc.New(db),
		qualityService: qualityService,
		logger:         logger.With().Str("component", "slots").Logger(),
	}
}

// SetDB updates the database connection used by this service.
// This is called when switching between production and development databases.
func (s *Service) SetDB(db *sql.DB) {
	s.queries = sqlc.New(db)
}

// GetSettings retrieves the multi-version settings.
func (s *Service) GetSettings(ctx context.Context) (*MultiVersionSettings, error) {
	row, err := s.queries.GetMultiVersionSettings(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get multi-version settings: %w", err)
	}
	return s.rowToSettings(row), nil
}

// UpdateSettings updates the multi-version settings.
func (s *Service) UpdateSettings(ctx context.Context, input UpdateMultiVersionSettingsInput) (*MultiVersionSettings, error) {
	current, err := s.GetSettings(ctx)
	if err != nil {
		return nil, err
	}

	// Req 1.2.3: Enabling requires completing dry-run first
	if input.Enabled && !current.Enabled && !current.DryRunCompleted {
		return nil, ErrDryRunRequired
	}

	// If enabling, validate slot configuration
	if input.Enabled && !current.Enabled {
		if err := s.ValidateSlotConfiguration(ctx); err != nil {
			return nil, err
		}
	}

	row, err := s.queries.SetMultiVersionEnabled(ctx, boolToInt64(input.Enabled))
	if err != nil {
		return nil, fmt.Errorf("failed to update multi-version settings: %w", err)
	}

	s.logger.Info().Bool("enabled", input.Enabled).Msg("Updated multi-version settings")

	// Clear all import decisions — mode change invalidates all previous evaluations
	if err := s.queries.CleanupAllImportDecisions(ctx); err != nil {
		s.logger.Warn().Err(err).Msg("Failed to clear import decisions after multi-version toggle")
	}

	return s.rowToSettings(row), nil
}

// IsMultiVersionEnabled returns whether multi-version is enabled.
// Req 1.2.2: When disabled, system behaves as single-version
func (s *Service) IsMultiVersionEnabled(ctx context.Context) bool {
	settings, err := s.GetSettings(ctx)
	if err != nil {
		s.logger.Warn().Err(err).Msg("Failed to get multi-version settings, assuming disabled")
		return false
	}
	return settings.Enabled
}

// Get retrieves a slot by ID.
func (s *Service) Get(ctx context.Context, id int64) (*Slot, error) {
	row, err := s.queries.GetVersionSlot(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSlotNotFound
		}
		return nil, fmt.Errorf("failed to get slot: %w", err)
	}
	return s.rowToSlot(row), nil
}

// GetByNumber retrieves a slot by slot number (1, 2, or 3).
func (s *Service) GetByNumber(ctx context.Context, slotNumber int) (*Slot, error) {
	row, err := s.queries.GetVersionSlotByNumber(ctx, int64(slotNumber))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSlotNotFound
		}
		return nil, fmt.Errorf("failed to get slot: %w", err)
	}
	return s.rowToSlot(row), nil
}

// List returns all slots.
func (s *Service) List(ctx context.Context) ([]*Slot, error) {
	rows, err := s.queries.ListVersionSlots(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list slots: %w", err)
	}

	slots := make([]*Slot, 0, len(rows))
	for _, row := range rows {
		slot := s.rowToSlot(row)
		// Load profile info if assigned
		if row.QualityProfileID.Valid {
			profile, err := s.qualityService.Get(ctx, row.QualityProfileID.Int64)
			if err == nil {
				slot.QualityProfile = &SlotProfile{
					ID:              profile.ID,
					Name:            profile.Name,
					Cutoff:          profile.Cutoff,
					UpgradesEnabled: profile.UpgradesEnabled,
				}
			}
		}
		slots = append(slots, slot)
	}
	return slots, nil
}

// ListEnabled returns only enabled slots.
func (s *Service) ListEnabled(ctx context.Context) ([]*Slot, error) {
	rows, err := s.queries.ListEnabledVersionSlots(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list enabled slots: %w", err)
	}

	slots := make([]*Slot, 0, len(rows))
	for _, row := range rows {
		slot := s.rowToSlot(row)
		if row.QualityProfileID.Valid {
			profile, err := s.qualityService.Get(ctx, row.QualityProfileID.Int64)
			if err == nil {
				slot.QualityProfile = &SlotProfile{
					ID:              profile.ID,
					Name:            profile.Name,
					Cutoff:          profile.Cutoff,
					UpgradesEnabled: profile.UpgradesEnabled,
				}
			}
		}
		slots = append(slots, slot)
	}
	return slots, nil
}

// GetEffectiveSlots returns slots based on multi-version mode.
// When disabled, returns only Slot 1 for legacy behavior.
func (s *Service) GetEffectiveSlots(ctx context.Context) ([]*Slot, error) {
	if !s.IsMultiVersionEnabled(ctx) {
		slot, err := s.GetByNumber(ctx, 1)
		if err != nil {
			return nil, err
		}
		return []*Slot{slot}, nil
	}
	return s.ListEnabled(ctx)
}

// Update updates a slot.
func (s *Service) Update(ctx context.Context, id int64, input UpdateSlotInput) (*Slot, error) {
	if input.Name == "" {
		return nil, ErrInvalidSlot
	}

	var profileID sql.NullInt64
	if input.QualityProfileID != nil {
		profileID = sql.NullInt64{Int64: *input.QualityProfileID, Valid: true}
	}

	var movieRootFolderID sql.NullInt64
	if input.MovieRootFolderID != nil {
		movieRootFolderID = sql.NullInt64{Int64: *input.MovieRootFolderID, Valid: true}
	}

	var tvRootFolderID sql.NullInt64
	if input.TVRootFolderID != nil {
		tvRootFolderID = sql.NullInt64{Int64: *input.TVRootFolderID, Valid: true}
	}

	row, err := s.queries.UpdateVersionSlot(ctx, sqlc.UpdateVersionSlotParams{
		ID:                id,
		Name:              input.Name,
		Enabled:           boolToInt64(input.Enabled),
		QualityProfileID:  profileID,
		DisplayOrder:      int64(input.DisplayOrder),
		MovieRootFolderID: movieRootFolderID,
		TvRootFolderID:    tvRootFolderID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSlotNotFound
		}
		return nil, fmt.Errorf("failed to update slot: %w", err)
	}

	s.logger.Info().Int64("id", id).Str("name", input.Name).Msg("Updated slot")

	// Clear all import decisions — slot config change may affect evaluations
	if err := s.queries.CleanupAllImportDecisions(ctx); err != nil {
		s.logger.Warn().Err(err).Msg("Failed to clear import decisions after slot update")
	}

	return s.rowToSlot(row), nil
}

// SetEnabled enables or disables a slot.
func (s *Service) SetEnabled(ctx context.Context, id int64, enabled bool) (*Slot, error) {
	// If disabling, check if slot has files
	if !enabled {
		count, err := s.GetSlotFileCount(ctx, id)
		if err != nil {
			return nil, err
		}
		if count > 0 {
			return nil, ErrSlotHasFiles
		}
	}

	row, err := s.queries.UpdateVersionSlotEnabled(ctx, sqlc.UpdateVersionSlotEnabledParams{
		ID:      id,
		Enabled: boolToInt64(enabled),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSlotNotFound
		}
		return nil, fmt.Errorf("failed to update slot enabled: %w", err)
	}

	s.logger.Info().Int64("id", id).Bool("enabled", enabled).Msg("Updated slot enabled status")

	// Clear all import decisions — slot availability change may affect evaluations
	if err := s.queries.CleanupAllImportDecisions(ctx); err != nil {
		s.logger.Warn().Err(err).Msg("Failed to clear import decisions after slot enable/disable")
	}

	return s.rowToSlot(row), nil
}

// SetProfile sets the quality profile for a slot.
func (s *Service) SetProfile(ctx context.Context, id int64, profileID *int64) (*Slot, error) {
	var nullProfileID sql.NullInt64
	if profileID != nil {
		nullProfileID = sql.NullInt64{Int64: *profileID, Valid: true}
	}

	row, err := s.queries.UpdateVersionSlotProfile(ctx, sqlc.UpdateVersionSlotProfileParams{
		ID:               id,
		QualityProfileID: nullProfileID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSlotNotFound
		}
		return nil, fmt.Errorf("failed to update slot profile: %w", err)
	}

	s.logger.Info().Int64("id", id).Interface("profileId", profileID).Msg("Updated slot profile")
	return s.rowToSlot(row), nil
}

// SetRootFolders sets the root folders for a slot.
// Req 22.1.1-22.1.2: Each slot can have a dedicated root folder per media type.
func (s *Service) SetRootFolders(ctx context.Context, slotID int64, movieRootFolderID, tvRootFolderID *int64) (*Slot, error) {
	var movieID sql.NullInt64
	if movieRootFolderID != nil {
		movieID = sql.NullInt64{Int64: *movieRootFolderID, Valid: true}
	}

	var tvID sql.NullInt64
	if tvRootFolderID != nil {
		tvID = sql.NullInt64{Int64: *tvRootFolderID, Valid: true}
	}

	row, err := s.queries.UpdateVersionSlotRootFolders(ctx, sqlc.UpdateVersionSlotRootFoldersParams{
		ID:                slotID,
		MovieRootFolderID: movieID,
		TvRootFolderID:    tvID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSlotNotFound
		}
		return nil, fmt.Errorf("failed to update slot root folders: %w", err)
	}

	s.logger.Info().
		Int64("id", slotID).
		Interface("movieRootFolderId", movieRootFolderID).
		Interface("tvRootFolderId", tvRootFolderID).
		Msg("Updated slot root folders")
	return s.rowToSlot(row), nil
}

// GetRootFolderForSlot returns the appropriate root folder path for a slot and media type.
// Returns empty string if no slot root folder is set (caller should fall back to media item root folder).
// Req 22.2.1-22.2.3: Check target slot's root folder first, fall back if not set.
func (s *Service) GetRootFolderForSlot(ctx context.Context, slotID int64, mediaType string) (string, error) {
	if s.rootFolderProvider == nil {
		return "", nil
	}

	slot, err := s.Get(ctx, slotID)
	if err != nil {
		return "", err
	}

	var rootFolderID *int64
	switch mediaType {
	case "movie":
		rootFolderID = slot.MovieRootFolderID
	case "episode", "tv":
		rootFolderID = slot.TVRootFolderID
	default:
		return "", nil
	}

	if rootFolderID == nil {
		return "", nil
	}

	rf, err := s.rootFolderProvider.Get(ctx, *rootFolderID)
	if err != nil {
		s.logger.Warn().Err(err).Int64("rootFolderId", *rootFolderID).Msg("Failed to get slot root folder")
		return "", nil
	}

	return rf.Path, nil
}

// GetSlotFileCount returns the total number of files assigned to a slot.
func (s *Service) GetSlotFileCount(ctx context.Context, slotID int64) (int64, error) {
	nullSlotID := sql.NullInt64{Int64: slotID, Valid: true}
	count, err := s.queries.CountAllFilesInSlot(ctx, sqlc.CountAllFilesInSlotParams{
		SlotID:   nullSlotID,
		SlotID_2: nullSlotID,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to count files in slot: %w", err)
	}
	return count, nil
}

// GetSlotFileID returns the file ID assigned to a specific slot for a media item.
// Returns 0 if no file is assigned.
func (s *Service) GetSlotFileID(ctx context.Context, mediaType string, mediaID, slotID int64) (int64, error) {
	if mediaType == "movie" {
		rows, err := s.queries.ListMovieSlotAssignments(ctx, mediaID)
		if err != nil {
			return 0, err
		}
		for _, row := range rows {
			if row.SlotID == slotID && row.FileID.Valid {
				return row.FileID.Int64, nil
			}
		}
	} else if mediaType == "episode" {
		rows, err := s.queries.ListEpisodeSlotAssignments(ctx, mediaID)
		if err != nil {
			return 0, err
		}
		for _, row := range rows {
			if row.SlotID == slotID && row.FileID.Valid {
				return row.FileID.Int64, nil
			}
		}
	}
	return 0, nil
}

// ValidationResult holds the result of slot configuration validation.
type ValidationResult struct {
	Valid     bool
	Errors    []string
	Conflicts []SlotConflict
}

// SlotConflict represents a conflict between two slots' profiles.
type SlotConflict struct {
	SlotAName string           `json:"slotAName"`
	SlotBName string           `json:"slotBName"`
	Issues    []AttributeIssue `json:"issues"`
}

// AttributeIssue describes a specific attribute overlap.
type AttributeIssue struct {
	Attribute string `json:"attribute"`
	Message   string `json:"message"`
}

// ValidateSlotConfiguration validates the current slot configuration.
// Called before enabling multi-version mode.
func (s *Service) ValidateSlotConfiguration(ctx context.Context) error {
	result := s.ValidateSlotConfigurationFull(ctx)
	if !result.Valid {
		if len(result.Errors) > 0 {
			return fmt.Errorf("%s", result.Errors[0])
		}
		return ErrInvalidSlot
	}
	return nil
}

// ValidateSlotConfigurationFull validates the current slot configuration
// and returns all validation errors.
func (s *Service) ValidateSlotConfigurationFull(ctx context.Context) ValidationResult {
	var errors []string

	slots, err := s.ListEnabled(ctx)
	if err != nil {
		return ValidationResult{Valid: false, Errors: []string{fmt.Sprintf("Failed to list slots: %v", err)}}
	}

	if len(slots) < 1 {
		errors = append(errors, "At least one slot must be enabled")
	}

	// Check each enabled slot has a profile and collect profiles for exclusivity check
	var profiles []*quality.Profile
	var slotsWithProfiles []*Slot
	for _, slot := range slots {
		if slot.QualityProfileID == nil {
			errors = append(errors, fmt.Sprintf("Slot %q does not have a quality profile assigned", slot.Name))
			continue
		}
		profile, err := s.qualityService.Get(ctx, *slot.QualityProfileID)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Failed to get profile for slot %q: %v", slot.Name, err))
			continue
		}
		profiles = append(profiles, profile)
		slotsWithProfiles = append(slotsWithProfiles, slot)
	}

	// Check mutual exclusivity between all enabled slots with profiles
	var conflicts []SlotConflict
	if len(profiles) >= 2 {
		slotConfigs := make([]quality.SlotConfig, len(profiles))
		for i, p := range profiles {
			slotConfigs[i] = quality.SlotConfig{
				SlotNumber: slotsWithProfiles[i].SlotNumber,
				SlotName:   slotsWithProfiles[i].Name,
				Enabled:    true,
				Profile:    p,
			}
		}
		exclusivityErrs, valid := quality.ValidateSlotExclusivity(slotConfigs)
		if !valid {
			s.logger.Warn().Interface("errors", exclusivityErrs).Msg("Slot profiles are not mutually exclusive")
			for _, e := range exclusivityErrs {
				// Convert quality.AttributeIssue to slots.AttributeIssue
				var issues []AttributeIssue
				for _, issue := range e.Issues {
					issues = append(issues, AttributeIssue{
						Attribute: issue.Attribute,
						Message:   issue.Message,
					})
				}
				conflicts = append(conflicts, SlotConflict{
					SlotAName: e.SlotAName,
					SlotBName: e.SlotBName,
					Issues:    issues,
				})
				errors = append(errors, fmt.Sprintf("Profile conflict between %s and %s", e.SlotAName, e.SlotBName))
			}
		}
	}

	return ValidationResult{
		Valid:     len(errors) == 0,
		Errors:    errors,
		Conflicts: conflicts,
	}
}

// SetDryRunCompleted marks the dry-run as completed.
func (s *Service) SetDryRunCompleted(ctx context.Context, completed bool) error {
	_, err := s.queries.SetDryRunCompleted(ctx, boolToInt64(completed))
	if err != nil {
		return fmt.Errorf("failed to set dry-run completed: %w", err)
	}
	s.logger.Info().Bool("completed", completed).Msg("Set dry-run completed")
	return nil
}

// rowToSlot converts a database row to a Slot.
func (s *Service) rowToSlot(row *sqlc.VersionSlot) *Slot {
	slot := &Slot{
		ID:           row.ID,
		SlotNumber:   int(row.SlotNumber),
		Name:         row.Name,
		Enabled:      row.Enabled != 0,
		DisplayOrder: int(row.DisplayOrder),
	}

	if row.QualityProfileID.Valid {
		slot.QualityProfileID = &row.QualityProfileID.Int64
	}
	if row.MovieRootFolderID.Valid {
		slot.MovieRootFolderID = &row.MovieRootFolderID.Int64
	}
	if row.TvRootFolderID.Valid {
		slot.TVRootFolderID = &row.TvRootFolderID.Int64
	}
	if row.CreatedAt.Valid {
		slot.CreatedAt = row.CreatedAt.Time
	}
	if row.UpdatedAt.Valid {
		slot.UpdatedAt = row.UpdatedAt.Time
	}

	return slot
}

// rowToSettings converts a database row to MultiVersionSettings.
func (s *Service) rowToSettings(row *sqlc.MultiVersionSetting) *MultiVersionSettings {
	settings := &MultiVersionSettings{
		Enabled:         row.Enabled != 0,
		DryRunCompleted: row.DryRunCompleted != 0,
	}

	if row.LastMigrationAt.Valid {
		settings.LastMigrationAt = &row.LastMigrationAt.Time
	}
	if row.CreatedAt.Valid {
		settings.CreatedAt = row.CreatedAt.Time
	}
	if row.UpdatedAt.Valid {
		settings.UpdatedAt = row.UpdatedAt.Time
	}

	return settings
}

func boolToInt64(b bool) int64 {
	if b {
		return 1
	}
	return 0
}
