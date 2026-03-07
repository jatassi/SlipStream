package slots

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/library/quality"
	"github.com/slipstream/slipstream/internal/library/rootfolder"
)

var (
	ErrSlotNotFound          = errors.New("slot not found")
	ErrSlotNoQualityProfile  = errors.New("slot has no quality profile configured")
	ErrInvalidSlot           = errors.New("invalid slot configuration")
	ErrSlotHasFiles          = errors.New("slot has files assigned")
	ErrDryRunRequired        = errors.New("dry-run preview is required before enabling multi-version")
	ErrProfilesNotExclusive  = errors.New("assigned profiles are not mutually exclusive")
	ErrMissingProfileForSlot = errors.New("enabled slot must have a quality profile assigned")
)

// FileDeleter deletes files from disk and database.
// Used when disabling a slot with the "delete" action.
type FileDeleter interface {
	DeleteFile(ctx context.Context, mediaType string, fileID int64) error
}

// RootFolderProvider provides root folder lookup operations.
type RootFolderProvider interface {
	Get(ctx context.Context, id int64) (*rootfolder.RootFolder, error)
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

// NewService creates a new slot service.
func NewService(db *sql.DB, qualityService *quality.Service, logger *zerolog.Logger, rootFolderProvider RootFolderProvider) *Service {
	return &Service{
		queries:            sqlc.New(db),
		qualityService:     qualityService,
		logger:             logger.With().Str("component", "slots").Logger(),
		rootFolderProvider: rootFolderProvider,
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

	row, err := s.queries.SetMultiVersionEnabled(ctx, input.Enabled)
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
	slot := s.rowToSlot(row)
	if err := s.enrichSlotWithRootFolders(ctx, slot); err != nil {
		s.logger.Warn().Err(err).Int64("slotId", id).Msg("Failed to enrich slot with root folders")
	}
	return slot, nil
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
	slot := s.rowToSlot(row)
	if err := s.enrichSlotWithRootFolders(ctx, slot); err != nil {
		s.logger.Warn().Err(err).Int("slotNumber", slotNumber).Msg("Failed to enrich slot with root folders")
	}
	return slot, nil
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
		if err := s.enrichSlotWithRootFolders(ctx, slot); err != nil {
			s.logger.Warn().Err(err).Int64("slotId", slot.ID).Msg("Failed to enrich slot with root folders")
		}
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
		if err := s.enrichSlotWithRootFolders(ctx, slot); err != nil {
			s.logger.Warn().Err(err).Int64("slotId", slot.ID).Msg("Failed to enrich slot with root folders")
		}
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

	row, err := s.queries.UpdateVersionSlot(ctx, sqlc.UpdateVersionSlotParams{
		ID:               id,
		Name:             input.Name,
		Enabled:          input.Enabled,
		QualityProfileID: profileID,
		DisplayOrder:     int64(input.DisplayOrder),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSlotNotFound
		}
		return nil, fmt.Errorf("failed to update slot: %w", err)
	}

	// Sync root folders via pivot table
	if input.RootFolders != nil {
		if err := s.syncRootFolders(ctx, id, input.RootFolders); err != nil {
			return nil, fmt.Errorf("failed to sync root folders: %w", err)
		}
	}

	s.logger.Info().Int64("id", id).Str("name", input.Name).Msg("Updated slot")

	// Clear all import decisions — slot config change may affect evaluations
	if err := s.queries.CleanupAllImportDecisions(ctx); err != nil {
		s.logger.Warn().Err(err).Msg("Failed to clear import decisions after slot update")
	}

	slot := s.rowToSlot(row)
	if err := s.enrichSlotWithRootFolders(ctx, slot); err != nil {
		s.logger.Warn().Err(err).Int64("slotId", id).Msg("Failed to enrich slot with root folders")
	}
	return slot, nil
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
		Enabled: enabled,
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

	slot := s.rowToSlot(row)
	if err := s.enrichSlotWithRootFolders(ctx, slot); err != nil {
		s.logger.Warn().Err(err).Int64("slotId", id).Msg("Failed to enrich slot with root folders")
	}
	return slot, nil
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
	slot := s.rowToSlot(row)
	if err := s.enrichSlotWithRootFolders(ctx, slot); err != nil {
		s.logger.Warn().Err(err).Int64("slotId", id).Msg("Failed to enrich slot with root folders")
	}
	return slot, nil
}

// SetRootFolders sets the root folders for a slot.
// Req 22.1.1-22.1.2: Each slot can have a dedicated root folder per media type.
func (s *Service) SetRootFolders(ctx context.Context, slotID int64, rootFolders map[string]*int64) (*Slot, error) {
	// Verify slot exists
	_, err := s.queries.GetVersionSlot(ctx, slotID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSlotNotFound
		}
		return nil, fmt.Errorf("failed to get slot: %w", err)
	}

	if err := s.syncRootFolders(ctx, slotID, rootFolders); err != nil {
		return nil, fmt.Errorf("failed to sync root folders: %w", err)
	}

	s.logger.Info().
		Int64("id", slotID).
		Interface("rootFolders", rootFolders).
		Msg("Updated slot root folders")

	return s.Get(ctx, slotID)
}

// syncRootFolders synchronizes root folder assignments for a slot via the pivot table.
func (s *Service) syncRootFolders(ctx context.Context, slotID int64, rootFolders map[string]*int64) error {
	for moduleType, rootFolderID := range rootFolders {
		if rootFolderID != nil {
			_, err := s.queries.UpsertSlotRootFolder(ctx, sqlc.UpsertSlotRootFolderParams{
				SlotID:       slotID,
				ModuleType:   moduleType,
				RootFolderID: *rootFolderID,
			})
			if err != nil {
				return fmt.Errorf("failed to upsert root folder for module %s: %w", moduleType, err)
			}
		} else {
			err := s.queries.DeleteSlotRootFolder(ctx, sqlc.DeleteSlotRootFolderParams{
				SlotID:     slotID,
				ModuleType: moduleType,
			})
			if err != nil {
				return fmt.Errorf("failed to delete root folder for module %s: %w", moduleType, err)
			}
		}
	}
	return nil
}

// GetRootFolderForSlot returns the appropriate root folder path for a slot and media type.
// Returns empty string if no slot root folder is set (caller should fall back to media item root folder).
// Req 22.2.1-22.2.3: Check target slot's root folder first, fall back if not set.
func (s *Service) GetRootFolderForSlot(ctx context.Context, slotID int64, mediaType string) (string, error) {
	if s.rootFolderProvider == nil {
		return "", nil
	}

	// Map media type to module type
	var moduleType string
	switch mediaType {
	case mediaTypeMovie:
		moduleType = "movie"
	case mediaTypeEpisode, "tv":
		moduleType = "tv"
	default:
		return "", nil
	}

	rootFolderRow, err := s.queries.GetSlotRootFolder(ctx, sqlc.GetSlotRootFolderParams{
		SlotID:     slotID,
		ModuleType: moduleType,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		s.logger.Warn().Err(err).Int64("slotId", slotID).Str("moduleType", moduleType).Msg("Failed to query slot root folder")
		return "", nil
	}

	rf, err := s.rootFolderProvider.Get(ctx, rootFolderRow.RootFolderID)
	if err != nil {
		s.logger.Warn().Err(err).Int64("rootFolderId", rootFolderRow.RootFolderID).Msg("Failed to get slot root folder")
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
// getMovieSlotFileID retrieves the file ID for a movie slot assignment
func (s *Service) getMovieSlotFileID(ctx context.Context, movieID, slotID int64) (int64, error) {
	rows, err := s.queries.ListMovieSlotAssignments(ctx, movieID)
	if err != nil {
		return 0, err
	}
	for _, row := range rows {
		if row.SlotID == slotID && row.FileID.Valid {
			return row.FileID.Int64, nil
		}
	}
	return 0, nil
}

// getEpisodeSlotFileID retrieves the file ID for an episode slot assignment
func (s *Service) getEpisodeSlotFileID(ctx context.Context, episodeID, slotID int64) (int64, error) {
	rows, err := s.queries.ListEpisodeSlotAssignments(ctx, episodeID)
	if err != nil {
		return 0, err
	}
	for _, row := range rows {
		if row.SlotID == slotID && row.FileID.Valid {
			return row.FileID.Int64, nil
		}
	}
	return 0, nil
}

func (s *Service) GetSlotFileID(ctx context.Context, mediaType string, mediaID, slotID int64) (int64, error) {
	switch mediaType {
	case mediaTypeMovie:
		return s.getMovieSlotFileID(ctx, mediaID, slotID)
	case mediaTypeEpisode:
		return s.getEpisodeSlotFileID(ctx, mediaID, slotID)
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
// validateSlotProfiles checks that all slots have valid profiles and returns them
func (s *Service) validateSlotProfiles(ctx context.Context, slots []*Slot) ([]*quality.Profile, []*Slot, []string) {
	var validationErrors []string
	var profiles []*quality.Profile
	var slotsWithProfiles []*Slot

	for _, slot := range slots {
		if slot.QualityProfileID == nil {
			validationErrors = append(validationErrors, fmt.Sprintf("Slot %q does not have a quality profile assigned", slot.Name))
			continue
		}
		profile, err := s.qualityService.Get(ctx, *slot.QualityProfileID)
		if err != nil {
			validationErrors = append(validationErrors, fmt.Sprintf("Failed to get profile for slot %q: %v", slot.Name, err))
			continue
		}
		profiles = append(profiles, profile)
		slotsWithProfiles = append(slotsWithProfiles, slot)
	}

	return profiles, slotsWithProfiles, validationErrors
}

// checkSlotExclusivity validates mutual exclusivity between slot profiles
func (s *Service) checkSlotExclusivity(profiles []*quality.Profile, slotsWithProfiles []*Slot) (conflicts []SlotConflict, errMsgs []string) {
	if len(profiles) < 2 {
		return nil, nil
	}

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
	if valid {
		return nil, nil
	}

	s.logger.Warn().Interface("errors", exclusivityErrs).Msg("Slot profiles are not mutually exclusive")

	conflicts = make([]SlotConflict, 0)
	errMsgs = make([]string, 0)
	for i := range exclusivityErrs {
		e := &exclusivityErrs[i]
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
		errMsgs = append(errMsgs, fmt.Sprintf("Profile conflict between %s and %s", e.SlotAName, e.SlotBName))
	}

	return conflicts, errMsgs
}

func (s *Service) ValidateSlotConfigurationFull(ctx context.Context) ValidationResult {
	var validationErrors []string

	slots, err := s.ListEnabled(ctx)
	if err != nil {
		return ValidationResult{Valid: false, Errors: []string{fmt.Sprintf("Failed to list slots: %v", err)}}
	}

	if len(slots) < 1 {
		validationErrors = append(validationErrors, "At least one slot must be enabled")
	}

	profiles, slotsWithProfiles, profileErrors := s.validateSlotProfiles(ctx, slots)
	validationErrors = append(validationErrors, profileErrors...)

	conflicts, exclusivityErrors := s.checkSlotExclusivity(profiles, slotsWithProfiles)
	validationErrors = append(validationErrors, exclusivityErrors...)

	return ValidationResult{
		Valid:     len(validationErrors) == 0,
		Errors:    validationErrors,
		Conflicts: conflicts,
	}
}

// SetDryRunCompleted marks the dry-run as completed.
func (s *Service) SetDryRunCompleted(ctx context.Context, completed bool) error {
	_, err := s.queries.SetDryRunCompleted(ctx, completed)
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
		Enabled:      row.Enabled,
		DisplayOrder: int(row.DisplayOrder),
		RootFolders:  make(map[string]*int64),
	}

	if row.QualityProfileID.Valid {
		slot.QualityProfileID = &row.QualityProfileID.Int64
	}
	if row.CreatedAt.Valid {
		slot.CreatedAt = row.CreatedAt.Time
	}
	if row.UpdatedAt.Valid {
		slot.UpdatedAt = row.UpdatedAt.Time
	}

	return slot
}

// enrichSlotWithRootFolders queries the pivot table and populates the RootFolders map on a slot.
func (s *Service) enrichSlotWithRootFolders(ctx context.Context, slot *Slot) error {
	rows, err := s.queries.ListSlotRootFolders(ctx, slot.ID)
	if err != nil {
		return fmt.Errorf("failed to list root folders for slot %d: %w", slot.ID, err)
	}

	if slot.RootFolders == nil {
		slot.RootFolders = make(map[string]*int64)
	}

	for _, row := range rows {
		id := row.RootFolderID
		slot.RootFolders[row.ModuleType] = &id
	}

	return nil
}

// rowToSettings converts a database row to MultiVersionSettings.
func (s *Service) rowToSettings(row *sqlc.MultiVersionSetting) *MultiVersionSettings {
	settings := &MultiVersionSettings{
		Enabled:         row.Enabled,
		DryRunCompleted: row.DryRunCompleted,
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
