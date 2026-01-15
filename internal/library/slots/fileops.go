package slots

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/slipstream/slipstream/internal/database/sqlc"
)

// SlotFileInfo represents a file assigned to a slot.
type SlotFileInfo struct {
	MediaType  string `json:"mediaType"`
	FileID     int64  `json:"fileId"`
	FilePath   string `json:"filePath"`
	MediaID    int64  `json:"mediaId"`
	MediaTitle string `json:"mediaTitle"`
}

// DisableSlotAction represents the action to take when disabling a slot with files.
type DisableSlotAction string

const (
	// DisableActionDelete deletes all files in the slot.
	DisableActionDelete DisableSlotAction = "delete"
	// DisableActionKeep keeps files but removes slot assignment.
	DisableActionKeep DisableSlotAction = "keep"
	// DisableActionCancel aborts the disable operation.
	DisableActionCancel DisableSlotAction = "cancel"
)

// DisableSlotRequest is the request for disabling a slot.
type DisableSlotRequest struct {
	SlotID int64             `json:"slotId"`
	Action DisableSlotAction `json:"action"`
}

// DisableSlotResult is the result of checking if a slot can be disabled.
type DisableSlotResult struct {
	RequiresPrompt bool           `json:"requiresPrompt"`
	FilesCount     int            `json:"filesCount"`
	Files          []SlotFileInfo `json:"files,omitempty"`
}

// OnFileDeleted handles the file deletion event by clearing slot assignments.
// Req 12.1.1: Deleting file from slot does NOT trigger automatic search
// Req 12.1.2: Slot becomes empty; waits for next scheduled search
func (s *Service) OnFileDeleted(ctx context.Context, mediaType string, fileID int64) error {
	switch mediaType {
	case "movie":
		// Clear from slot assignment table
		if err := s.queries.ClearMovieSlotFileByFileID(ctx, sql.NullInt64{Int64: fileID, Valid: true}); err != nil {
			s.logger.Warn().Err(err).Int64("fileId", fileID).Msg("Failed to clear movie slot assignment")
		}
		// Clear slot_id from file table (if file hasn't been deleted yet)
		if err := s.queries.ClearMovieFileSlot(ctx, fileID); err != nil && err != sql.ErrNoRows {
			s.logger.Warn().Err(err).Int64("fileId", fileID).Msg("Failed to clear movie file slot_id")
		}
	case "episode":
		if err := s.queries.ClearEpisodeSlotFileByFileID(ctx, sql.NullInt64{Int64: fileID, Valid: true}); err != nil {
			s.logger.Warn().Err(err).Int64("fileId", fileID).Msg("Failed to clear episode slot assignment")
		}
		if err := s.queries.ClearEpisodeFileSlot(ctx, fileID); err != nil && err != sql.ErrNoRows {
			s.logger.Warn().Err(err).Int64("fileId", fileID).Msg("Failed to clear episode file slot_id")
		}
	}

	s.logger.Debug().Str("mediaType", mediaType).Int64("fileId", fileID).Msg("Cleared slot assignment for deleted file")
	return nil
}

// CheckDisableSlot checks if a slot can be disabled and returns info about files.
// Req 12.2.1: When user disables slot with files, prompt for action
func (s *Service) CheckDisableSlot(ctx context.Context, slotID int64) (*DisableSlotResult, error) {
	files, err := s.ListFilesInSlot(ctx, slotID)
	if err != nil {
		return nil, err
	}

	return &DisableSlotResult{
		RequiresPrompt: len(files) > 0,
		FilesCount:     len(files),
		Files:          files,
	}, nil
}

// DisableSlotWithAction disables a slot with the specified action for existing files.
// Req 12.2.2: Options: delete files, keep unassigned, or cancel
func (s *Service) DisableSlotWithAction(ctx context.Context, req DisableSlotRequest) error {
	if req.Action == DisableActionCancel {
		return nil
	}

	switch req.Action {
	case DisableActionDelete:
		// Delete all files in this slot
		files, err := s.ListFilesInSlot(ctx, req.SlotID)
		if err != nil {
			return err
		}

		// Req 12.2.2: Delete files from disk and database
		for _, file := range files {
			if s.fileDeleter != nil {
				if err := s.fileDeleter.DeleteFile(ctx, file.MediaType, file.FileID); err != nil {
					s.logger.Warn().
						Err(err).
						Int64("fileId", file.FileID).
						Str("path", file.FilePath).
						Msg("Failed to delete file during slot disable")
				} else {
					s.logger.Info().
						Int64("fileId", file.FileID).
						Str("path", file.FilePath).
						Msg("Deleted file during slot disable")
				}
			} else {
				s.logger.Warn().
					Int64("fileId", file.FileID).
					Str("path", file.FilePath).
					Msg("No file deleter configured, cannot delete file during slot disable")
			}
		}

		// Clear all slot assignments for this slot
		if err := s.ClearSlotAssignments(ctx, req.SlotID); err != nil {
			return err
		}

	case DisableActionKeep:
		// Just clear slot assignments, keep files unassigned
		if err := s.ClearSlotAssignments(ctx, req.SlotID); err != nil {
			return err
		}
	}

	// Now disable the slot
	_, err := s.queries.UpdateVersionSlotEnabled(ctx, sqlc.UpdateVersionSlotEnabledParams{
		ID:      req.SlotID,
		Enabled: 0,
	})
	if err != nil {
		return err
	}

	s.logger.Info().
		Int64("slotId", req.SlotID).
		Str("action", string(req.Action)).
		Msg("Disabled slot with action")

	return nil
}

// ListFilesInSlot returns all files assigned to a slot.
func (s *Service) ListFilesInSlot(ctx context.Context, slotID int64) ([]SlotFileInfo, error) {
	rows, err := s.queries.ListFilesAssignedToSlot(ctx, sqlc.ListFilesAssignedToSlotParams{
		SlotID:   sql.NullInt64{Int64: slotID, Valid: true},
		SlotID_2: sql.NullInt64{Int64: slotID, Valid: true},
	})
	if err != nil {
		return nil, err
	}

	files := make([]SlotFileInfo, 0, len(rows))
	for _, row := range rows {
		files = append(files, SlotFileInfo{
			MediaType:  row.MediaType,
			FileID:     row.FileID,
			FilePath:   row.FilePath,
			MediaID:    row.MediaID,
			MediaTitle: row.MediaTitle,
		})
	}
	return files, nil
}

// ClearSlotAssignments clears all file assignments for a slot.
func (s *Service) ClearSlotAssignments(ctx context.Context, slotID int64) error {
	// Clear slot_id from movie_files
	if err := s.queries.ClearAllMovieFileSlotsForSlot(ctx, sql.NullInt64{Int64: slotID, Valid: true}); err != nil {
		return err
	}

	// Clear slot_id from episode_files
	if err := s.queries.ClearAllEpisodeFileSlotsForSlot(ctx, sql.NullInt64{Int64: slotID, Valid: true}); err != nil {
		return err
	}

	// Clear file_id from movie_slot_assignments
	if err := s.queries.ClearAllMovieSlotAssignmentsForSlot(ctx, slotID); err != nil {
		return err
	}

	// Clear file_id from episode_slot_assignments
	if err := s.queries.ClearAllEpisodeSlotAssignmentsForSlot(ctx, slotID); err != nil {
		return err
	}

	return nil
}

// GetFilesNeedingReviewCount returns the count of files that couldn't be assigned to slots.
// This is used during migration and scanning.
func (s *Service) GetFilesNeedingReviewCount(ctx context.Context) (int64, error) {
	// Count movie files without slot assignments
	movieCount, err := s.queries.CountMovieFilesWithoutSlot(ctx)
	if err != nil {
		return 0, err
	}

	// Count episode files without slot assignments
	episodeCount, err := s.queries.CountEpisodeFilesWithoutSlot(ctx)
	if err != nil {
		return 0, err
	}

	return movieCount + episodeCount, nil
}

// ReviewQueueItem represents a file in the review queue (no slot assigned).
type ReviewQueueItem struct {
	MediaType  string `json:"mediaType"`
	FileID     int64  `json:"fileId"`
	FilePath   string `json:"filePath"`
	MediaID    int64  `json:"mediaId"`
	MediaTitle string `json:"mediaTitle"`
	Quality    string `json:"quality,omitempty"`
	Size       int64  `json:"size"`
}

// ReviewQueueStats represents statistics about the review queue.
type ReviewQueueStats struct {
	TotalFiles   int64 `json:"totalFiles"`
	MovieFiles   int64 `json:"movieFiles"`
	EpisodeFiles int64 `json:"episodeFiles"`
}

// GetReviewQueueStats returns statistics about files in the review queue.
// Req 13.1.3: Extra files (more than slot count) queued for user review
func (s *Service) GetReviewQueueStats(ctx context.Context) (*ReviewQueueStats, error) {
	movieCount, err := s.queries.CountMovieFilesWithoutSlot(ctx)
	if err != nil {
		return nil, err
	}

	episodeCount, err := s.queries.CountEpisodeFilesWithoutSlot(ctx)
	if err != nil {
		return nil, err
	}

	return &ReviewQueueStats{
		TotalFiles:   movieCount + episodeCount,
		MovieFiles:   movieCount,
		EpisodeFiles: episodeCount,
	}, nil
}

// ListReviewQueueItems returns files that are in the review queue (no slot assigned).
func (s *Service) ListReviewQueueItems(ctx context.Context) ([]ReviewQueueItem, error) {
	var items []ReviewQueueItem

	// Get movie files without slots
	movieRows, err := s.queries.ListMovieFilesWithoutSlot(ctx)
	if err != nil {
		return nil, err
	}
	for _, row := range movieRows {
		items = append(items, ReviewQueueItem{
			MediaType:  "movie",
			FileID:     row.ID,
			FilePath:   row.Path,
			MediaID:    row.MovieID,
			MediaTitle: row.MovieTitle,
			Quality:    row.Quality.String,
			Size:       row.Size,
		})
	}

	// Get episode files without slots
	episodeRows, err := s.queries.ListEpisodeFilesWithoutSlot(ctx)
	if err != nil {
		return nil, err
	}
	for _, row := range episodeRows {
		items = append(items, ReviewQueueItem{
			MediaType:  "episode",
			FileID:     row.ID,
			FilePath:   row.Path,
			MediaID:    row.EpisodeID,
			MediaTitle: row.SeriesTitle + " S" + fmt.Sprintf("%02d", row.SeasonNumber) + "E" + fmt.Sprintf("%02d", row.EpisodeNumber),
			Quality:    row.Quality.String,
			Size:       row.Size,
		})
	}

	return items, nil
}
