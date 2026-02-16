package slots

import (
	"context"
	"database/sql"
	"errors"

	"github.com/slipstream/slipstream/internal/database/sqlc"
)

const (
	statusUnreleased = "unreleased"
	mediaTypeMovie   = "movie"
	mediaTypeEpisode = "episode"
)

// SlotStatus represents the status of a slot for a specific media item.
type SlotStatus struct {
	SlotID           int64  `json:"slotId"`
	SlotNumber       int    `json:"slotNumber"`
	SlotName         string `json:"slotName"`
	Monitored        bool   `json:"monitored"`
	Status           string `json:"status"`
	FileID           *int64 `json:"fileId,omitempty"`
	CurrentQuality   string `json:"currentQuality,omitempty"`
	CurrentQualityID *int64 `json:"currentQualityId,omitempty"`
	ProfileCutoff    int    `json:"profileCutoff"`
}

// MediaStatus represents the aggregated status of a movie or episode.
type MediaStatus struct {
	MediaType      string       `json:"mediaType"`
	MediaID        int64        `json:"mediaId"`
	Status         string       `json:"status"`
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
// buildSlotStatus creates a SlotStatus from a slot and optional assignment
func (s *Service) buildSlotStatus(ctx context.Context, slot *Slot, assignment *MovieSlotAssignmentRow, mediaStatus *MediaStatus) SlotStatus {
	slotStatus := SlotStatus{
		SlotID:     slot.ID,
		SlotNumber: slot.SlotNumber,
		SlotName:   slot.Name,
		Monitored:  true,
		Status:     "missing",
	}

	if slot.QualityProfile != nil {
		slotStatus.ProfileCutoff = slot.QualityProfile.Cutoff
	}

	if assignment == nil {
		mediaStatus.EmptySlots++
		mediaStatus.MonitoredSlots++
		return slotStatus
	}

	slotStatus.Monitored = assignment.Monitored
	slotStatus.Status = assignment.Status

	if fileID, ok := assignment.FileID.(sql.NullInt64); ok && fileID.Valid {
		slotStatus.FileID = &fileID.Int64
		mediaStatus.FilledSlots++

		fileQuality := s.getMovieFileQuality(ctx, fileID.Int64)
		if fileQuality != nil {
			slotStatus.CurrentQuality = fileQuality.Quality
			slotStatus.CurrentQualityID = &fileQuality.QualityID
		}
	} else {
		mediaStatus.EmptySlots++
	}

	if slotStatus.Monitored {
		mediaStatus.MonitoredSlots++
	}

	return slotStatus
}

// buildEpisodeSlotStatus creates a SlotStatus from a slot and optional assignment for episodes
func (s *Service) buildEpisodeSlotStatus(ctx context.Context, slot *Slot, assignment *EpisodeSlotAssignmentRow, mediaStatus *MediaStatus) SlotStatus {
	slotStatus := SlotStatus{
		SlotID:     slot.ID,
		SlotNumber: slot.SlotNumber,
		SlotName:   slot.Name,
		Monitored:  true,
		Status:     "missing",
	}

	if slot.QualityProfile != nil {
		slotStatus.ProfileCutoff = slot.QualityProfile.Cutoff
	}

	if assignment == nil {
		mediaStatus.EmptySlots++
		mediaStatus.MonitoredSlots++
		return slotStatus
	}

	slotStatus.Monitored = assignment.Monitored
	slotStatus.Status = assignment.Status

	if fileID, ok := assignment.FileID.(sql.NullInt64); ok && fileID.Valid {
		slotStatus.FileID = &fileID.Int64
		mediaStatus.FilledSlots++

		fileQuality := s.getEpisodeFileQuality(ctx, fileID.Int64)
		if fileQuality != nil {
			slotStatus.CurrentQuality = fileQuality.Quality
			slotStatus.CurrentQualityID = &fileQuality.QualityID
		}
	} else {
		mediaStatus.EmptySlots++
	}

	if slotStatus.Monitored {
		mediaStatus.MonitoredSlots++
	}

	return slotStatus
}

func (s *Service) GetMovieStatus(ctx context.Context, movieID int64) (*MediaStatus, error) {
	slots, err := s.ListEnabled(ctx)
	if err != nil {
		return nil, err
	}

	if len(slots) == 0 {
		// If no slots enabled, return basic status
		return &MediaStatus{
			MediaType: mediaTypeMovie,
			MediaID:   movieID,
		}, nil
	}

	status := &MediaStatus{
		MediaType:    mediaTypeMovie,
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
		var assignmentPtr *MovieSlotAssignmentRow
		if assignment, ok := assignmentMap[slot.ID]; ok {
			assignmentPtr = &assignment
		}
		slotStatus := s.buildSlotStatus(ctx, slot, assignmentPtr, status)
		status.SlotStatuses = append(status.SlotStatuses, slotStatus)
	}

	// Compute aggregate status from slot statuses
	status.Status = s.computeAggregateStatus(status.SlotStatuses)

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
			MediaType: mediaTypeEpisode,
			MediaID:   episodeID,
		}, nil
	}

	status := &MediaStatus{
		MediaType:    mediaTypeEpisode,
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
		var assignmentPtr *EpisodeSlotAssignmentRow
		if assignment, ok := assignmentMap[slot.ID]; ok {
			assignmentPtr = &assignment
		}
		slotStatus := s.buildEpisodeSlotStatus(ctx, slot, assignmentPtr, status)
		status.SlotStatuses = append(status.SlotStatuses, slotStatus)
	}

	status.Status = s.computeAggregateStatus(status.SlotStatuses)

	return status, nil
}

// SetSlotMonitored sets the monitored status for a slot assignment.
// Req 1.1.6: Each slot has its own independent monitored status per movie/episode
// Req 8.1.1: Each slot has its own monitored toggle per movie/episode
// Req 8.1.2: A slot can be monitored independently
func (s *Service) SetSlotMonitored(ctx context.Context, mediaType string, mediaID, slotID int64, monitored bool) error {
	monitoredVal := int64(0)
	if monitored {
		monitoredVal = 1
	}

	switch mediaType {
	case mediaTypeMovie:
		// Check if assignment exists
		_, err := s.queries.GetMovieSlotAssignment(ctx, sqlc.GetMovieSlotAssignmentParams{
			MovieID: mediaID,
			SlotID:  slotID,
		})
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				_, err = s.queries.CreateMovieSlotAssignment(ctx, sqlc.CreateMovieSlotAssignmentParams{
					MovieID:   mediaID,
					SlotID:    slotID,
					FileID:    sql.NullInt64{},
					Monitored: monitoredVal,
					Status:    "missing",
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
	case mediaTypeEpisode:
		_, err := s.queries.GetEpisodeSlotAssignment(ctx, sqlc.GetEpisodeSlotAssignmentParams{
			EpisodeID: mediaID,
			SlotID:    slotID,
		})
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				_, err = s.queries.CreateEpisodeSlotAssignment(ctx, sqlc.CreateEpisodeSlotAssignmentParams{
					EpisodeID: mediaID,
					SlotID:    slotID,
					FileID:    sql.NullInt64{},
					Monitored: monitoredVal,
					Status:    "missing",
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
// initializeMovieSlot creates a slot assignment for a movie if it doesn't exist
func (s *Service) initializeMovieSlot(ctx context.Context, movieID, slotID int64) {
	_, err := s.queries.GetMovieSlotAssignment(ctx, sqlc.GetMovieSlotAssignmentParams{
		MovieID: movieID,
		SlotID:  slotID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		_, err = s.queries.CreateMovieSlotAssignment(ctx, sqlc.CreateMovieSlotAssignmentParams{
			MovieID:   movieID,
			SlotID:    slotID,
			FileID:    sql.NullInt64{},
			Monitored: 1,
			Status:    "missing",
		})
		if err != nil {
			s.logger.Warn().Err(err).Int64("movieId", movieID).Int64("slotId", slotID).Msg("Failed to create movie slot assignment")
		}
	}
}

// initializeEpisodeSlot creates a slot assignment for an episode if it doesn't exist
func (s *Service) initializeEpisodeSlot(ctx context.Context, episodeID, slotID int64) {
	_, err := s.queries.GetEpisodeSlotAssignment(ctx, sqlc.GetEpisodeSlotAssignmentParams{
		EpisodeID: episodeID,
		SlotID:    slotID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		_, err = s.queries.CreateEpisodeSlotAssignment(ctx, sqlc.CreateEpisodeSlotAssignmentParams{
			EpisodeID: episodeID,
			SlotID:    slotID,
			FileID:    sql.NullInt64{},
			Monitored: 1,
			Status:    "missing",
		})
		if err != nil {
			s.logger.Warn().Err(err).Int64("episodeId", episodeID).Int64("slotId", slotID).Msg("Failed to create episode slot assignment")
		}
	}
}

func (s *Service) InitializeSlotAssignments(ctx context.Context, mediaType string, mediaID int64) error {
	slots, err := s.ListEnabled(ctx)
	if err != nil {
		return err
	}

	for _, slot := range slots {
		switch mediaType {
		case mediaTypeMovie:
			s.initializeMovieSlot(ctx, mediaID, slot.ID)
		case mediaTypeEpisode:
			s.initializeEpisodeSlot(ctx, mediaID, slot.ID)
		}
	}
	return nil
}

// computeAggregateStatus derives the overall media status from individual slot statuses.
// Priority order: downloading > failed > missing > upgradable > available > unreleased.
var statusPriority = map[string]int{
	"downloading":    6,
	"failed":         5,
	"missing":        4,
	"upgradable":     3,
	"available":      2,
	statusUnreleased: 1,
}

func (s *Service) computeAggregateStatus(slotStatuses []SlotStatus) string {
	highestPriority := 0
	highestStatus := statusUnreleased

	for _, ss := range slotStatuses {
		if !ss.Monitored {
			continue
		}
		if priority := statusPriority[ss.Status]; priority > highestPriority {
			highestPriority = priority
			highestStatus = ss.Status
		}
	}

	return highestStatus
}
