package slots

import (
	"context"
	"database/sql"
	"errors"
	"sort"

	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/library/quality"
	"github.com/slipstream/slipstream/internal/library/scanner"
)

var (
	ErrNoMatchingSlot     = errors.New("no matching slot for release")
	ErrRejectImport       = errors.New("import rejected")
	ErrBelowAllProfiles   = errors.New("release below all profile requirements")
	ErrMixedSlotState     = errors.New("cannot import: some slots filled, some empty")
	ErrAllSlotsFilled     = errors.New("cannot import: all slots filled")
	ErrSlotSelectionRequired = errors.New("slot selection required: multiple slots match")
)

// SlotAssignment represents a potential slot assignment for a release.
type SlotAssignment struct {
	SlotID       int64   `json:"slotId"`
	SlotNumber   int     `json:"slotNumber"`
	SlotName     string  `json:"slotName"`
	MatchScore   float64 `json:"matchScore"`
	IsUpgrade    bool    `json:"isUpgrade"`
	IsNewFill    bool    `json:"isNewFill"`
	NeedsUpgrade bool    `json:"needsUpgrade"` // True if file accepted but below cutoff
	Confidence   float64 `json:"confidence"`   // How confident in the assignment (0-1)

	// File info if slot is currently filled
	CurrentFileID *int64  `json:"currentFileId,omitempty"`
	CurrentQuality string `json:"currentQuality,omitempty"`
}

// SlotEvaluation contains the results of evaluating a release against all slots.
type SlotEvaluation struct {
	Assignments       []SlotAssignment `json:"assignments"`
	RecommendedSlotID int64            `json:"recommendedSlotId"`
	RequiresSelection bool             `json:"requiresSelection"` // True if user should choose
	MatchingCount     int              `json:"matchingCount"`     // Number of slots that match
}

// EvaluateRelease evaluates a parsed release against all enabled slot profiles.
// Req 5.1.1: Evaluate release against all enabled slot profiles
// Req 5.1.2: Calculate match score for each slot's profile
func (s *Service) EvaluateRelease(ctx context.Context, parsed *scanner.ParsedMedia, mediaType string, mediaID int64) (*SlotEvaluation, error) {
	slots, err := s.ListEnabledWithProfiles(ctx)
	if err != nil {
		return nil, err
	}

	if len(slots) == 0 {
		return nil, ErrNoMatchingSlot
	}

	releaseAttrs := parsed.ToReleaseAttributes()
	assignments := make([]SlotAssignment, 0, len(slots))

	for _, slot := range slots {
		if slot.QualityProfileID == nil {
			continue
		}

		profile, err := s.qualityService.Get(ctx, *slot.QualityProfileID)
		if err != nil {
			s.logger.Warn().Err(err).Int64("slotId", slot.ID).Msg("Failed to get profile for slot")
			continue
		}

		// Calculate match result
		matchResult := quality.MatchProfileAttributes(releaseAttrs, profile)

		// Calculate quality score based on resolution and source
		qualityScore := s.calculateQualityScore(parsed, profile)

		// Total score combines quality matching and attribute bonuses
		totalScore := qualityScore + matchResult.TotalScore

		// Get current file info for this slot
		currentFile := s.getCurrentSlotFile(ctx, mediaType, mediaID, slot.ID)

		assignment := SlotAssignment{
			SlotID:     slot.ID,
			SlotNumber: slot.SlotNumber,
			SlotName:   slot.Name,
			MatchScore: totalScore,
			IsNewFill:  currentFile == nil,
			Confidence: 0.0,
		}

		if currentFile != nil {
			assignment.CurrentFileID = currentFile.FileID
			assignment.CurrentQuality = currentFile.Quality
			assignment.IsUpgrade = totalScore > currentFile.QualityScore
		}

		// Only include slots where the release passes attribute requirements
		if matchResult.AllMatch {
			assignment.Confidence = 1.0
		} else {
			// Partial match - lower confidence
			assignment.Confidence = 0.5
		}

		assignments = append(assignments, assignment)
	}

	if len(assignments) == 0 {
		return nil, ErrNoMatchingSlot
	}

	// Sort by score descending, then by empty slots, then by slot number
	sort.Slice(assignments, func(i, j int) bool {
		if assignments[i].MatchScore != assignments[j].MatchScore {
			return assignments[i].MatchScore > assignments[j].MatchScore
		}
		// Prefer empty slots on tie
		if assignments[i].IsNewFill != assignments[j].IsNewFill {
			return assignments[i].IsNewFill
		}
		return assignments[i].SlotNumber < assignments[j].SlotNumber
	})

	// Count matching slots (high confidence)
	matchingCount := 0
	for _, a := range assignments {
		if a.Confidence >= 1.0 {
			matchingCount++
		}
	}

	return &SlotEvaluation{
		Assignments:       assignments,
		RecommendedSlotID: assignments[0].SlotID,
		RequiresSelection: matchingCount > 1 && assignments[0].MatchScore == assignments[1].MatchScore,
		MatchingCount:     matchingCount,
	}, nil
}

// DetermineTargetSlot determines the best slot for a release.
// Req 5.1.3: Assign to the slot with the best quality match
// Req 5.1.4: If scores equal, assign to whichever slot is empty (or first if both empty)
// Req 5.1.5: Mutual exclusivity should prevent true ties in normal operation
func (s *Service) DetermineTargetSlot(ctx context.Context, parsed *scanner.ParsedMedia, mediaType string, mediaID int64) (*SlotAssignment, error) {
	eval, err := s.EvaluateRelease(ctx, parsed, mediaType, mediaID)
	if err != nil {
		return nil, err
	}

	if len(eval.Assignments) == 0 {
		return nil, ErrNoMatchingSlot
	}

	// First assignment is the best match
	return &eval.Assignments[0], nil
}

// HandleBelowProfileImport handles the case where a file doesn't fully match any profile.
// Req 5.3.1: All slots empty - accept to closest-matching slot, show "upgrade needed"
// Req 5.3.2: Some slots filled, some empty - reject import
// Req 5.3.3: All slots filled - reject import
func (s *Service) HandleBelowProfileImport(ctx context.Context, parsed *scanner.ParsedMedia, mediaType string, mediaID int64) (*SlotAssignment, error) {
	slots, err := s.ListEnabled(ctx)
	if err != nil {
		return nil, err
	}

	if len(slots) == 0 {
		return nil, ErrNoMatchingSlot
	}

	// Check slot fill status
	var emptySlots, filledSlots int
	for _, slot := range slots {
		file := s.getCurrentSlotFile(ctx, mediaType, mediaID, slot.ID)
		if file == nil {
			emptySlots++
		} else {
			filledSlots++
		}
	}

	// Req 5.3.3: All slots filled - reject
	if emptySlots == 0 && filledSlots > 0 {
		return nil, ErrAllSlotsFilled
	}

	// Req 5.3.2: Some slots filled, some empty - reject
	if emptySlots > 0 && filledSlots > 0 {
		return nil, ErrMixedSlotState
	}

	// Req 5.3.1: All slots empty - accept to Slot 1 with "upgrade needed" flag
	slot, err := s.GetByNumber(ctx, 1)
	if err != nil {
		return nil, err
	}

	return &SlotAssignment{
		SlotID:       slot.ID,
		SlotNumber:   slot.SlotNumber,
		SlotName:     slot.Name,
		MatchScore:   0,
		IsNewFill:    true,
		NeedsUpgrade: true,
		Confidence:   0.25,
	}, nil
}

// currentSlotFileInfo holds info about a file currently assigned to a slot.
type currentSlotFileInfo struct {
	FileID       *int64
	Quality      string
	QualityScore float64
}

// getCurrentSlotFile gets info about the current file in a slot for a media item.
func (s *Service) getCurrentSlotFile(ctx context.Context, mediaType string, mediaID int64, slotID int64) *currentSlotFileInfo {
	switch mediaType {
	case "movie":
		assignment, err := s.queries.GetMovieSlotAssignment(ctx, sqlc.GetMovieSlotAssignmentParams{
			MovieID: mediaID,
			SlotID:  slotID,
		})
		if err != nil {
			return nil
		}
		if !assignment.FileID.Valid {
			return nil
		}
		info := &currentSlotFileInfo{
			FileID: &assignment.FileID.Int64,
		}
		if fq := s.getMovieFileQuality(ctx, assignment.FileID.Int64); fq != nil {
			info.Quality = fq.Quality
		}
		return info
	case "episode":
		assignment, err := s.queries.GetEpisodeSlotAssignment(ctx, sqlc.GetEpisodeSlotAssignmentParams{
			EpisodeID: mediaID,
			SlotID:    slotID,
		})
		if err != nil {
			return nil
		}
		if !assignment.FileID.Valid {
			return nil
		}
		info := &currentSlotFileInfo{
			FileID: &assignment.FileID.Int64,
		}
		if fq := s.getEpisodeFileQuality(ctx, assignment.FileID.Int64); fq != nil {
			info.Quality = fq.Quality
		}
		return info
	}
	return nil
}

// calculateQualityScore calculates a quality score based on resolution and source.
func (s *Service) calculateQualityScore(parsed *scanner.ParsedMedia, profile *quality.Profile) float64 {
	var score float64

	// Resolution scoring (major factor)
	switch parsed.Quality {
	case "2160p":
		score += 40
	case "1080p":
		score += 30
	case "720p":
		score += 20
	case "480p":
		score += 10
	}

	// Source scoring
	switch parsed.Source {
	case "Remux":
		score += 10
	case "BluRay":
		score += 8
	case "WEB-DL":
		score += 6
	case "WEBRip":
		score += 5
	case "HDTV":
		score += 4
	case "DVDRip":
		score += 2
	case "SDTV":
		score += 1
	}

	return score
}

// ListEnabledWithProfiles returns enabled slots with their full profile information.
func (s *Service) ListEnabledWithProfiles(ctx context.Context) ([]*SlotWithProfile, error) {
	rows, err := s.queries.ListEnabledVersionSlotsWithProfiles(ctx)
	if err != nil {
		return nil, err
	}

	slots := make([]*SlotWithProfile, 0, len(rows))
	for _, row := range rows {
		slot := &SlotWithProfile{
			Slot: Slot{
				ID:           row.ID,
				SlotNumber:   int(row.SlotNumber),
				Name:         row.Name,
				Enabled:      row.Enabled != 0,
				DisplayOrder: int(row.DisplayOrder),
			},
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

		// Profile settings for matching
		if row.ProfileItems.Valid {
			slot.ProfileItems = row.ProfileItems.String
		}
		if row.ProfileHdrSettings.Valid {
			slot.ProfileHDRSettings = row.ProfileHdrSettings.String
		}
		if row.ProfileVideoCodecSettings.Valid {
			slot.ProfileVideoCodecSettings = row.ProfileVideoCodecSettings.String
		}
		if row.ProfileAudioCodecSettings.Valid {
			slot.ProfileAudioCodecSettings = row.ProfileAudioCodecSettings.String
		}
		if row.ProfileAudioChannelSettings.Valid {
			slot.ProfileAudioChannelSettings = row.ProfileAudioChannelSettings.String
		}

		slots = append(slots, slot)
	}

	return slots, nil
}

// AssignFileToSlot assigns a file to a specific slot for a media item.
func (s *Service) AssignFileToSlot(ctx context.Context, mediaType string, mediaID int64, slotID int64, fileID int64) error {
	switch mediaType {
	case "movie":
		_, err := s.queries.UpsertMovieSlotAssignment(ctx, sqlc.UpsertMovieSlotAssignmentParams{
			MovieID:   mediaID,
			SlotID:    slotID,
			FileID:    sql.NullInt64{Int64: fileID, Valid: true},
			Monitored: 1,
		})
		if err != nil {
			return err
		}
		// Also update the movie_files table slot_id
		_, err = s.queries.UpdateMovieFileSlot(ctx, sqlc.UpdateMovieFileSlotParams{
			SlotID: sql.NullInt64{Int64: slotID, Valid: true},
			ID:     fileID,
		})
		return err
	case "episode":
		_, err := s.queries.UpsertEpisodeSlotAssignment(ctx, sqlc.UpsertEpisodeSlotAssignmentParams{
			EpisodeID: mediaID,
			SlotID:    slotID,
			FileID:    sql.NullInt64{Int64: fileID, Valid: true},
			Monitored: 1,
		})
		if err != nil {
			return err
		}
		// Also update the episode_files table slot_id
		_, err = s.queries.UpdateEpisodeFileSlot(ctx, sqlc.UpdateEpisodeFileSlotParams{
			SlotID: sql.NullInt64{Int64: slotID, Valid: true},
			ID:     fileID,
		})
		return err
	}
	return nil
}

// UnassignFileFromSlot removes a file from a slot.
func (s *Service) UnassignFileFromSlot(ctx context.Context, mediaType string, mediaID int64, slotID int64) error {
	switch mediaType {
	case "movie":
		return s.queries.ClearMovieSlotFile(ctx, sqlc.ClearMovieSlotFileParams{
			MovieID: mediaID,
			SlotID:  slotID,
		})
	case "episode":
		return s.queries.ClearEpisodeSlotFile(ctx, sqlc.ClearEpisodeSlotFileParams{
			EpisodeID: mediaID,
			SlotID:    slotID,
		})
	}
	return nil
}

// GetMovieSlotAssignments gets all slot assignments for a movie.
func (s *Service) GetMovieSlotAssignments(ctx context.Context, movieID int64) ([]MovieSlotAssignmentRow, error) {
	rows, err := s.queries.ListMovieSlotAssignments(ctx, movieID)
	if err != nil {
		return nil, err
	}

	result := make([]MovieSlotAssignmentRow, 0, len(rows))
	for _, row := range rows {
		result = append(result, MovieSlotAssignmentRow{
			ID:               row.ID,
			MovieID:          row.MovieID,
			SlotID:           row.SlotID,
			FileID:           row.FileID,
			Monitored:        row.Monitored != 0,
			Status:           row.Status,
			SlotName:         row.SlotName,
			SlotNumber:       int(row.SlotNumber),
			QualityProfileID: row.QualityProfileID,
		})
	}

	return result, nil
}

// GetEpisodeSlotAssignments gets all slot assignments for an episode.
func (s *Service) GetEpisodeSlotAssignments(ctx context.Context, episodeID int64) ([]EpisodeSlotAssignmentRow, error) {
	rows, err := s.queries.ListEpisodeSlotAssignments(ctx, episodeID)
	if err != nil {
		return nil, err
	}

	result := make([]EpisodeSlotAssignmentRow, 0, len(rows))
	for _, row := range rows {
		result = append(result, EpisodeSlotAssignmentRow{
			ID:               row.ID,
			EpisodeID:        row.EpisodeID,
			SlotID:           row.SlotID,
			FileID:           row.FileID,
			Monitored:        row.Monitored != 0,
			Status:           row.Status,
			SlotName:         row.SlotName,
			SlotNumber:       int(row.SlotNumber),
			QualityProfileID: row.QualityProfileID,
		})
	}

	return result, nil
}

// MovieSlotAssignmentRow represents a movie slot assignment with slot info.
type MovieSlotAssignmentRow struct {
	ID               int64
	MovieID          int64
	SlotID           int64
	FileID           interface{} // sql.NullInt64
	Monitored        bool
	Status           string
	SlotName         string
	SlotNumber       int
	QualityProfileID interface{} // sql.NullInt64
}

// EpisodeSlotAssignmentRow represents an episode slot assignment with slot info.
type EpisodeSlotAssignmentRow struct {
	ID               int64
	EpisodeID        int64
	SlotID           int64
	FileID           interface{} // sql.NullInt64
	Monitored        bool
	Status           string
	SlotName         string
	SlotNumber       int
	QualityProfileID interface{} // sql.NullInt64
}
