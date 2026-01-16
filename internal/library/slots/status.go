package slots

import (
	"context"
	"database/sql"

	"github.com/slipstream/slipstream/internal/database/sqlc"
)

// SlotStatus represents the status of a slot for a specific media item.
type SlotStatus struct {
	SlotID           int64   `json:"slotId"`
	SlotNumber       int     `json:"slotNumber"`
	SlotName         string  `json:"slotName"`
	Monitored        bool    `json:"monitored"`
	HasFile          bool    `json:"hasFile"`
	FileID           *int64  `json:"fileId,omitempty"`
	CurrentQuality   string  `json:"currentQuality,omitempty"`
	CurrentQualityID *int64  `json:"currentQualityId,omitempty"`
	ProfileCutoff    int     `json:"profileCutoff"`
	NeedsUpgrade     bool    `json:"needsUpgrade"` // Req 6.2.2: file below cutoff
	IsMissing        bool    `json:"isMissing"`    // Req 6.1.1: monitored slot is empty
}

// MediaStatus represents the aggregated status of a movie or episode.
type MediaStatus struct {
	MediaType      string       `json:"mediaType"`
	MediaID        int64        `json:"mediaId"`
	IsMissing      bool         `json:"isMissing"`      // Req 6.1.1: ANY monitored slot empty
	NeedsUpgrade   bool         `json:"needsUpgrade"`   // Any slot needs upgrade
	SlotStatuses   []SlotStatus `json:"slotStatuses"`
	FilledSlots    int          `json:"filledSlots"`
	EmptySlots     int          `json:"emptySlots"`
	MonitoredSlots int          `json:"monitoredSlots"`
}

// GetMovieStatus returns the full slot status for a movie.
// Req 6.1.1: Movie is "missing" if ANY monitored slot is empty
// Req 6.1.2: Unmonitored empty slots do not affect missing status
// Req 6.2.1: Each slot independently tracks upgrade eligibility
// Req 6.2.2: Slot is "upgrade needed" if file below cutoff
func (s *Service) GetMovieStatus(ctx context.Context, movieID int64) (*MediaStatus, error) {
	slots, err := s.ListEnabled(ctx)
	if err != nil {
		return nil, err
	}

	if len(slots) == 0 {
		// If no slots enabled, return basic status
		return &MediaStatus{
			MediaType: "movie",
			MediaID:   movieID,
		}, nil
	}

	status := &MediaStatus{
		MediaType:    "movie",
		MediaID:      movieID,
		SlotStatuses: make([]SlotStatus, 0, len(slots)),
	}

	// Get all slot assignments for this movie
	assignments, err := s.GetMovieSlotAssignments(ctx, movieID)
	if err != nil {
		return nil, err
	}

	// Create a map for quick lookup
	assignmentMap := make(map[int64]MovieSlotAssignmentRow)
	for _, a := range assignments {
		assignmentMap[a.SlotID] = a
	}

	for _, slot := range slots {
		slotStatus := SlotStatus{
			SlotID:     slot.ID,
			SlotNumber: slot.SlotNumber,
			SlotName:   slot.Name,
			Monitored:  true, // Default to monitored
		}

		// Get profile cutoff if available
		if slot.QualityProfile != nil {
			slotStatus.ProfileCutoff = slot.QualityProfile.Cutoff
		}

		// Check if we have an assignment for this slot
		if assignment, ok := assignmentMap[slot.ID]; ok {
			slotStatus.Monitored = assignment.Monitored

			// Check if slot has a file
			if fileID, ok := assignment.FileID.(sql.NullInt64); ok && fileID.Valid {
				slotStatus.HasFile = true
				slotStatus.FileID = &fileID.Int64
				status.FilledSlots++

				// Check upgrade status using quality from file
				fileQuality := s.getMovieFileQuality(ctx, fileID.Int64)
				if fileQuality != nil {
					slotStatus.CurrentQuality = fileQuality.Quality
					slotStatus.CurrentQualityID = &fileQuality.QualityID
					// Req 6.2.2: Check if below cutoff (only if upgrades enabled)
					if slot.QualityProfile != nil && slot.QualityProfile.UpgradesEnabled {
						slotStatus.NeedsUpgrade = fileQuality.QualityID < int64(slotStatus.ProfileCutoff)
						if slotStatus.NeedsUpgrade {
							status.NeedsUpgrade = true
						}
					}
				}
			} else {
				status.EmptySlots++
			}
		} else {
			// No assignment exists - slot is empty and uses default monitoring
			status.EmptySlots++
		}

		// Req 6.1.1: Check if this contributes to missing status
		if slotStatus.Monitored && !slotStatus.HasFile {
			slotStatus.IsMissing = true
			status.IsMissing = true
		}

		if slotStatus.Monitored {
			status.MonitoredSlots++
		}

		status.SlotStatuses = append(status.SlotStatuses, slotStatus)
	}

	return status, nil
}

// GetEpisodeStatus returns the full slot status for an episode.
func (s *Service) GetEpisodeStatus(ctx context.Context, episodeID int64) (*MediaStatus, error) {
	slots, err := s.ListEnabled(ctx)
	if err != nil {
		return nil, err
	}

	if len(slots) == 0 {
		return &MediaStatus{
			MediaType: "episode",
			MediaID:   episodeID,
		}, nil
	}

	status := &MediaStatus{
		MediaType:    "episode",
		MediaID:      episodeID,
		SlotStatuses: make([]SlotStatus, 0, len(slots)),
	}

	assignments, err := s.GetEpisodeSlotAssignments(ctx, episodeID)
	if err != nil {
		return nil, err
	}

	assignmentMap := make(map[int64]EpisodeSlotAssignmentRow)
	for _, a := range assignments {
		assignmentMap[a.SlotID] = a
	}

	for _, slot := range slots {
		slotStatus := SlotStatus{
			SlotID:     slot.ID,
			SlotNumber: slot.SlotNumber,
			SlotName:   slot.Name,
			Monitored:  true,
		}

		if slot.QualityProfile != nil {
			slotStatus.ProfileCutoff = slot.QualityProfile.Cutoff
		}

		if assignment, ok := assignmentMap[slot.ID]; ok {
			slotStatus.Monitored = assignment.Monitored

			if fileID, ok := assignment.FileID.(sql.NullInt64); ok && fileID.Valid {
				slotStatus.HasFile = true
				slotStatus.FileID = &fileID.Int64
				status.FilledSlots++

				fileQuality := s.getEpisodeFileQuality(ctx, fileID.Int64)
				if fileQuality != nil {
					slotStatus.CurrentQuality = fileQuality.Quality
					slotStatus.CurrentQualityID = &fileQuality.QualityID
					// Check if below cutoff (only if upgrades enabled)
					if slot.QualityProfile != nil && slot.QualityProfile.UpgradesEnabled {
						slotStatus.NeedsUpgrade = fileQuality.QualityID < int64(slotStatus.ProfileCutoff)
						if slotStatus.NeedsUpgrade {
							status.NeedsUpgrade = true
						}
					}
				}
			} else {
				status.EmptySlots++
			}
		} else {
			status.EmptySlots++
		}

		if slotStatus.Monitored && !slotStatus.HasFile {
			slotStatus.IsMissing = true
			status.IsMissing = true
		}

		if slotStatus.Monitored {
			status.MonitoredSlots++
		}

		status.SlotStatuses = append(status.SlotStatuses, slotStatus)
	}

	return status, nil
}

// SetSlotMonitored sets the monitored status for a slot assignment.
// Req 1.1.6: Each slot has its own independent monitored status per movie/episode
// Req 8.1.1: Each slot has its own monitored toggle per movie/episode
// Req 8.1.2: A slot can be monitored independently
func (s *Service) SetSlotMonitored(ctx context.Context, mediaType string, mediaID int64, slotID int64, monitored bool) error {
	monitoredVal := int64(0)
	if monitored {
		monitoredVal = 1
	}

	switch mediaType {
	case "movie":
		// Check if assignment exists
		_, err := s.queries.GetMovieSlotAssignment(ctx, sqlc.GetMovieSlotAssignmentParams{
			MovieID: mediaID,
			SlotID:  slotID,
		})
		if err != nil {
			if err == sql.ErrNoRows {
				// Create a new assignment with the specified monitoring status
				_, err = s.queries.CreateMovieSlotAssignment(ctx, sqlc.CreateMovieSlotAssignmentParams{
					MovieID:   mediaID,
					SlotID:    slotID,
					FileID:    sql.NullInt64{},
					Monitored: monitoredVal,
				})
				return err
			}
			return err
		}

		return s.queries.UpdateMovieSlotMonitored(ctx, sqlc.UpdateMovieSlotMonitoredParams{
			Monitored: monitoredVal,
			MovieID:   mediaID,
			SlotID:    slotID,
		})
	case "episode":
		_, err := s.queries.GetEpisodeSlotAssignment(ctx, sqlc.GetEpisodeSlotAssignmentParams{
			EpisodeID: mediaID,
			SlotID:    slotID,
		})
		if err != nil {
			if err == sql.ErrNoRows {
				_, err = s.queries.CreateEpisodeSlotAssignment(ctx, sqlc.CreateEpisodeSlotAssignmentParams{
					EpisodeID: mediaID,
					SlotID:    slotID,
					FileID:    sql.NullInt64{},
					Monitored: monitoredVal,
				})
				return err
			}
			return err
		}

		return s.queries.UpdateEpisodeSlotMonitored(ctx, sqlc.UpdateEpisodeSlotMonitoredParams{
			Monitored: monitoredVal,
			EpisodeID: mediaID,
			SlotID:    slotID,
		})
	}
	return nil
}

// fileQualityInfo holds quality information for a file.
type fileQualityInfo struct {
	Quality   string
	QualityID int64
}

// getMovieFileQuality retrieves quality info for a movie file.
func (s *Service) getMovieFileQuality(ctx context.Context, fileID int64) *fileQualityInfo {
	file, err := s.queries.GetMovieFile(ctx, fileID)
	if err != nil {
		return nil
	}
	if !file.QualityID.Valid {
		return nil
	}
	return &fileQualityInfo{
		Quality:   file.Quality.String,
		QualityID: file.QualityID.Int64,
	}
}

// getEpisodeFileQuality retrieves quality info for an episode file.
func (s *Service) getEpisodeFileQuality(ctx context.Context, fileID int64) *fileQualityInfo {
	file, err := s.queries.GetEpisodeFile(ctx, fileID)
	if err != nil {
		return nil
	}
	if !file.QualityID.Valid {
		return nil
	}
	return &fileQualityInfo{
		Quality:   file.Quality.String,
		QualityID: file.QualityID.Int64,
	}
}

// InitializeSlotAssignments creates slot assignments for a media item if they don't exist.
// This ensures all enabled slots have assignment rows for proper monitoring/status tracking.
func (s *Service) InitializeSlotAssignments(ctx context.Context, mediaType string, mediaID int64) error {
	slots, err := s.ListEnabled(ctx)
	if err != nil {
		return err
	}

	for _, slot := range slots {
		switch mediaType {
		case "movie":
			_, err := s.queries.GetMovieSlotAssignment(ctx, sqlc.GetMovieSlotAssignmentParams{
				MovieID: mediaID,
				SlotID:  slot.ID,
			})
			if err == sql.ErrNoRows {
				_, err = s.queries.CreateMovieSlotAssignment(ctx, sqlc.CreateMovieSlotAssignmentParams{
					MovieID:   mediaID,
					SlotID:    slot.ID,
					FileID:    sql.NullInt64{},
					Monitored: 1, // Default to monitored
				})
				if err != nil {
					s.logger.Warn().Err(err).Int64("movieId", mediaID).Int64("slotId", slot.ID).Msg("Failed to create movie slot assignment")
				}
			}
		case "episode":
			_, err := s.queries.GetEpisodeSlotAssignment(ctx, sqlc.GetEpisodeSlotAssignmentParams{
				EpisodeID: mediaID,
				SlotID:    slot.ID,
			})
			if err == sql.ErrNoRows {
				_, err = s.queries.CreateEpisodeSlotAssignment(ctx, sqlc.CreateEpisodeSlotAssignmentParams{
					EpisodeID: mediaID,
					SlotID:    slot.ID,
					FileID:    sql.NullInt64{},
					Monitored: 1,
				})
				if err != nil {
					s.logger.Warn().Err(err).Int64("episodeId", mediaID).Int64("slotId", slot.ID).Msg("Failed to create episode slot assignment")
				}
			}
		}
	}
	return nil
}
