package importer

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/downloader"
	"github.com/slipstream/slipstream/internal/library/scanner"
)

// SeasonPackFile represents a file within a season pack download.
type SeasonPackFile struct {
	Path         string               `json:"path"`
	Filename     string               `json:"filename"`
	Size         int64                `json:"size"`
	ParsedInfo   *scanner.ParsedMedia `json:"parsedInfo,omitempty"`
	EpisodeID    *int64               `json:"episodeId,omitempty"`
	SeriesID     *int64               `json:"seriesId,omitempty"`
	SeasonNumber int                  `json:"seasonNumber,omitempty"`
	EpisodeNum   int                  `json:"episodeNumber,omitempty"`
	EndEpisode   int                  `json:"endEpisode,omitempty"` // For multi-episode files
	IsReady      bool                 `json:"isReady"`
	IsMatched    bool                 `json:"isMatched"`
	Error        string               `json:"error,omitempty"`

	// Req 16.2.1, 16.2.3: Per-episode slot evaluation
	TargetSlotID   *int64  `json:"targetSlotId,omitempty"`
	TargetSlotName string  `json:"targetSlotName,omitempty"`
	SlotMatchScore float64 `json:"slotMatchScore,omitempty"`
	IsSlotUpgrade  bool    `json:"isSlotUpgrade,omitempty"`
	IsSlotNewFill  bool    `json:"isSlotNewFill,omitempty"`
}

// SeasonPackAnalysis contains the result of analyzing a season pack.
type SeasonPackAnalysis struct {
	DownloadPath   string           `json:"downloadPath"`
	SeriesID       *int64           `json:"seriesId,omitempty"`
	SeasonNumber   int              `json:"seasonNumber,omitempty"`
	TotalFiles     int              `json:"totalFiles"`
	ReadyFiles     int              `json:"readyFiles"`
	MatchedFiles   int              `json:"matchedFiles"`
	UnmatchedFiles int              `json:"unmatchedFiles"`
	Files          []SeasonPackFile `json:"files"`

	// Req 16.2.1, 16.2.2: Season pack slot summary
	SlotSummary *SeasonPackSlotSummary `json:"slotSummary,omitempty"`
}

// SeasonPackSlotSummary summarizes slot assignments for a season pack.
// Req 16.2.2: May result in mixed slots across seasons (acceptable)
type SeasonPackSlotSummary struct {
	PrimarySlotID   *int64         `json:"primarySlotId,omitempty"`   // Most common slot
	PrimarySlotName string         `json:"primarySlotName,omitempty"` // Name of most common slot
	IsMixedSlots    bool           `json:"isMixedSlots"`              // True if episodes go to different slots
	SlotCounts      map[int64]int  `json:"slotCounts"`                // Count of episodes per slot
	SlotNames       map[int64]string `json:"slotNames"`               // Names for display
}

// AnalyzeSeasonPack scans a download folder and identifies individual episodes.
func (s *Service) AnalyzeSeasonPack(ctx context.Context, downloadPath string, seriesID *int64) (*SeasonPackAnalysis, error) {
	result := &SeasonPackAnalysis{
		DownloadPath: downloadPath,
		SeriesID:     seriesID,
		Files:        make([]SeasonPackFile, 0),
	}

	// Find all video files
	files, err := s.findVideoFiles(downloadPath)
	if err != nil {
		return nil, err
	}

	result.TotalFiles = len(files)

	for _, filePath := range files {
		packFile := SeasonPackFile{
			Path:     filePath,
			Filename: filepath.Base(filePath),
		}

		// Get file info
		stat, err := os.Stat(filePath)
		if err == nil {
			packFile.Size = stat.Size()
		}

		// Parse the filename, falling back to parent dir for quality info
		parsed := scanner.ParsePath(filePath)
		packFile.ParsedInfo = parsed

		// Check if file is ready for import
		completion := s.CheckFileCompletion(ctx, filePath)
		packFile.IsReady = completion.Status == CompletionReady

		if packFile.IsReady {
			result.ReadyFiles++
		}

		// Extract episode info from parsed filename
		if parsed.Season > 0 {
			packFile.SeasonNumber = parsed.Season
			if result.SeasonNumber == 0 {
				result.SeasonNumber = parsed.Season
			}
		}

		if parsed.Episode > 0 {
			packFile.EpisodeNum = parsed.Episode
		}

		// Check for multi-episode files (e.g., S01E01-E02)
		if parsed.EndEpisode > 0 && parsed.EndEpisode > parsed.Episode {
			packFile.EndEpisode = parsed.EndEpisode
		}

		// Try to match to library if we have a series ID
		if seriesID != nil && packFile.EpisodeNum > 0 {
			episode, err := s.tv.GetEpisodeByNumber(ctx, *seriesID, packFile.SeasonNumber, packFile.EpisodeNum)
			if err == nil && episode != nil {
				packFile.EpisodeID = &episode.ID
				packFile.SeriesID = seriesID
				packFile.IsMatched = true
				result.MatchedFiles++
			} else {
				// Episode doesn't exist in database - create it from the parsed filename
				// This handles Netflix-style releases where metadata providers only have placeholders
				title := ""
				if parsed.Title != "" {
					title = parsed.Title
				} else {
					title = fmt.Sprintf("Episode %d", packFile.EpisodeNum)
				}

				newEpisode, createErr := s.tv.CreateEpisode(ctx, *seriesID, packFile.SeasonNumber, packFile.EpisodeNum, title)
				if createErr == nil && newEpisode != nil {
					packFile.EpisodeID = &newEpisode.ID
					packFile.SeriesID = seriesID
					packFile.IsMatched = true
					result.MatchedFiles++
					s.logger.Info().
						Int64("seriesId", *seriesID).
						Int("season", packFile.SeasonNumber).
						Int("episode", packFile.EpisodeNum).
						Str("filename", packFile.Filename).
						Msg("Created missing episode from season pack file")
				} else {
					s.logger.Warn().
						Err(createErr).
						Int64("seriesId", *seriesID).
						Int("season", packFile.SeasonNumber).
						Int("episode", packFile.EpisodeNum).
						Msg("Failed to create missing episode")
				}
			}
		}

		if !packFile.IsMatched {
			result.UnmatchedFiles++
		}

		result.Files = append(result.Files, packFile)
	}

	return result, nil
}

// CreateQueueMediaForSeasonPack creates queue_media entries for each file in a season pack.
// Req 16.2.3: Each episode from pack individually assessed with its own slot assignment.
func (s *Service) CreateQueueMediaForSeasonPack(ctx context.Context, mappingID int64, analysis *SeasonPackAnalysis) ([]*sqlc.QueueMedium, error) {
	entries := make([]*sqlc.QueueMedium, 0, len(analysis.Files))

	for _, file := range analysis.Files {
		if !file.IsMatched {
			continue
		}

		status := downloader.QueueMediaStatusPending
		if file.IsReady {
			status = downloader.QueueMediaStatusReady
		}

		input := downloader.CreateQueueMediaInput{
			DownloadMappingID: mappingID,
			EpisodeID:         file.EpisodeID,
			FilePath:          file.Path,
			FileStatus:        status,
			TargetSlotID:      file.TargetSlotID, // Req 16.2.3: Per-episode slot
		}

		entry, err := s.downloader.CreateQueueMedia(ctx, input)
		if err != nil {
			s.logger.Warn().Err(err).Str("path", file.Path).Msg("Failed to create queue media entry")
			continue
		}

		// Set episode status to downloading when queue_media entry is created
		if file.EpisodeID != nil {
			_ = s.queries.UpdateEpisodeStatusWithDetails(ctx, sqlc.UpdateEpisodeStatusWithDetailsParams{
				Status:           "downloading",
				ActiveDownloadID: sql.NullString{},
				StatusMessage:    sql.NullString{},
				ID:               *file.EpisodeID,
			})
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

// ProcessReadySeasonPackFiles imports only the files that are ready within a season pack.
// This allows individual episode import without waiting for the full pack.
func (s *Service) ProcessReadySeasonPackFiles(ctx context.Context, mappingID int64) (int, error) {
	// Get all queue_media entries for this mapping
	entries, err := s.downloader.GetQueueMediaByDownloadMapping(ctx, mappingID)
	if err != nil {
		return 0, err
	}

	var imported int
	for _, entry := range entries {
		// Skip if not ready
		if entry.FileStatus != string(downloader.QueueMediaStatusReady) {
			continue
		}

		// Skip if already importing/imported
		if entry.FileStatus == string(downloader.QueueMediaStatusImporting) ||
		   entry.FileStatus == string(downloader.QueueMediaStatusImported) {
			continue
		}

		// Skip if no file path
		if !entry.FilePath.Valid || entry.FilePath.String == "" {
			continue
		}

		// Update status to importing
		if err := s.downloader.UpdateQueueMediaStatus(ctx, entry.ID, downloader.QueueMediaStatusImporting); err != nil {
			s.logger.Warn().Err(err).Int64("id", entry.ID).Msg("Failed to update queue media status")
			continue
		}

		// Get the mapping for context
		mapping, err := s.getDownloadMappingByID(ctx, mappingID)
		if err != nil {
			s.logger.Warn().Err(err).Int64("mappingId", mappingID).Msg("Failed to get download mapping")
			continue
		}

		// Create the import job
		job := ImportJob{
			SourcePath:      entry.FilePath.String,
			DownloadMapping: mapping,
			QueueMedia: &QueueMedia{
				ID:                entry.ID,
				DownloadMappingID: entry.DownloadMappingID,
				FilePath:          entry.FilePath.String,
				FileStatus:        entry.FileStatus,
			},
			Manual: false,
		}

		if entry.EpisodeID.Valid {
			job.QueueMedia.EpisodeID = &entry.EpisodeID.Int64
		}

		// Req 16.2.3: Use per-episode slot from queue_media
		if entry.TargetSlotID.Valid {
			job.TargetSlotID = &entry.TargetSlotID.Int64
		}

		// Queue the import
		if err := s.QueueImport(job); err != nil {
			s.logger.Warn().Err(err).Str("path", entry.FilePath.String).Msg("Failed to queue season pack file")
			_ = s.downloader.UpdateQueueMediaStatusWithError(ctx, entry.ID, downloader.QueueMediaStatusFailed, err.Error())
			continue
		}

		imported++
	}

	return imported, nil
}

// UpdateSeasonPackFileStatuses checks completion status and updates queue_media entries.
func (s *Service) UpdateSeasonPackFileStatuses(ctx context.Context, mappingID int64) error {
	entries, err := s.downloader.GetQueueMediaByDownloadMapping(ctx, mappingID)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if !entry.FilePath.Valid || entry.FilePath.String == "" {
			continue
		}

		// Skip already completed/failed entries
		if entry.FileStatus == string(downloader.QueueMediaStatusImported) ||
		   entry.FileStatus == string(downloader.QueueMediaStatusFailed) ||
		   entry.FileStatus == string(downloader.QueueMediaStatusImporting) {
			continue
		}

		// Check completion status
		completion := s.CheckFileCompletion(ctx, entry.FilePath.String)

		var newStatus downloader.QueueMediaStatus
		switch completion.Status {
		case CompletionReady:
			newStatus = downloader.QueueMediaStatusReady
		case CompletionNotFound:
			newStatus = downloader.QueueMediaStatusFailed
		default:
			newStatus = downloader.QueueMediaStatusPending
		}

		// Update if status changed
		if string(newStatus) != entry.FileStatus {
			if err := s.downloader.UpdateQueueMediaStatus(ctx, entry.ID, newStatus); err != nil {
				s.logger.Warn().Err(err).Int64("id", entry.ID).Msg("Failed to update queue media status")
			}
		}
	}

	return nil
}

// getDownloadMappingByID retrieves a download mapping and converts it to our internal type.
func (s *Service) getDownloadMappingByID(ctx context.Context, mappingID int64) (*DownloadMapping, error) {
	// We need to get the mapping from the database
	// Unfortunately, we don't have a direct query for this, so we list and filter
	mappings, err := s.queries.ListActiveDownloadMappings(ctx)
	if err != nil {
		return nil, err
	}

	for _, m := range mappings {
		if m.ID == mappingID {
			return s.convertMapping(m), nil
		}
	}

	return nil, ErrNoMatch
}

// convertMapping converts a sqlc mapping to our internal type.
func (s *Service) convertMapping(m *sqlc.DownloadMapping) *DownloadMapping {
	mapping := &DownloadMapping{
		ID:               m.ID,
		DownloadClientID: m.ClientID,
		DownloadID:       m.DownloadID,
		IsSeasonPack:     m.IsSeasonPack == 1,
		IsCompleteSeries: m.IsCompleteSeries == 1,
		Source:           m.Source,
	}

	// Copy TargetSlotID if present
	if m.TargetSlotID.Valid {
		id := m.TargetSlotID.Int64
		mapping.TargetSlotID = &id
	}

	if m.MovieID.Valid {
		mapping.MovieID = &m.MovieID.Int64
		mapping.MediaType = "movie"
	}

	if m.SeriesID.Valid {
		mapping.SeriesID = &m.SeriesID.Int64
	}

	if m.SeasonNumber.Valid {
		sn := int(m.SeasonNumber.Int64)
		mapping.SeasonNumber = &sn
	}

	if m.EpisodeID.Valid {
		mapping.EpisodeID = &m.EpisodeID.Int64
	}

	// Determine MediaType based on flags and fields
	if mapping.MovieID != nil {
		mapping.MediaType = "movie"
	} else if mapping.IsCompleteSeries {
		mapping.MediaType = "series"
	} else if mapping.IsSeasonPack || (mapping.SeasonNumber != nil && mapping.EpisodeID == nil) {
		mapping.MediaType = "season"
	} else if mapping.SeriesID != nil {
		mapping.MediaType = "episode"
	}

	return mapping
}

// IsSeasonPack determines if a download is a season pack.
func (s *Service) IsSeasonPack(mapping *DownloadMapping) bool {
	// A download is a season pack if:
	// 1. It's explicitly marked as a season pack in the mapping
	// 2. Or it has a series ID and season number but no specific episode ID
	if mapping.IsSeasonPack {
		return true
	}
	if mapping.SeriesID != nil && mapping.SeasonNumber != nil && mapping.EpisodeID == nil {
		return true
	}
	return false
}

// DetectSeasonPackFromPath analyzes a path to detect if it's a season pack.
func (s *Service) DetectSeasonPackFromPath(ctx context.Context, downloadPath string) (bool, int, error) {
	files, err := s.findVideoFiles(downloadPath)
	if err != nil {
		return false, 0, err
	}

	if len(files) <= 1 {
		return false, len(files), nil
	}

	// Parse filenames and look for episode patterns
	episodeCount := 0
	seasonNumber := 0
	for _, file := range files {
		parsed := scanner.ParsePath(file)
		if parsed.Episode > 0 {
			episodeCount++
			if seasonNumber == 0 && parsed.Season > 0 {
				seasonNumber = parsed.Season
			}
		}
	}

	// If we found multiple episode files with episode numbers, it's likely a season pack
	isSeasonPack := episodeCount > 1

	return isSeasonPack, len(files), nil
}

// EvaluateSeasonPackSlots evaluates slots for each file in a season pack analysis.
// Req 16.2.1: Season pack assigned to the slot it best matches
// Req 16.2.2: May result in mixed slots across seasons (acceptable)
// Req 16.2.3: Each episode from pack individually assessed
func (s *Service) EvaluateSeasonPackSlots(ctx context.Context, analysis *SeasonPackAnalysis) error {
	if s.slots == nil {
		return nil // Slots service not configured, skip evaluation
	}

	if !s.slots.IsMultiVersionEnabled(ctx) {
		return nil // Multi-version not enabled, skip
	}

	slotCounts := make(map[int64]int)
	slotNames := make(map[int64]string)
	hasMultipleSlots := false
	var firstSlotID *int64

	for i := range analysis.Files {
		file := &analysis.Files[i]

		// Skip files without parsed info or episode ID
		if file.ParsedInfo == nil || file.EpisodeID == nil {
			continue
		}

		// Req 16.2.3: Individually assess each episode from the pack
		eval, err := s.slots.EvaluateRelease(ctx, file.ParsedInfo, "episode", *file.EpisodeID)
		if err != nil {
			s.logger.Debug().Err(err).Str("file", file.Filename).Msg("Failed to evaluate slots for season pack file")
			continue
		}

		if len(eval.Assignments) == 0 {
			continue
		}

		// Use the best assignment
		best := eval.Assignments[0]
		file.TargetSlotID = &best.SlotID
		file.TargetSlotName = best.SlotName
		file.SlotMatchScore = best.MatchScore
		file.IsSlotUpgrade = best.IsUpgrade
		file.IsSlotNewFill = best.IsNewFill

		// Track slot distribution
		slotCounts[best.SlotID]++
		slotNames[best.SlotID] = best.SlotName

		// Check for mixed slots
		if firstSlotID == nil {
			firstSlotID = &best.SlotID
		} else if *firstSlotID != best.SlotID {
			hasMultipleSlots = true
		}
	}

	// Build slot summary
	if len(slotCounts) > 0 {
		// Find primary (most common) slot
		var primarySlotID int64
		maxCount := 0
		for slotID, count := range slotCounts {
			if count > maxCount {
				maxCount = count
				primarySlotID = slotID
			}
		}

		analysis.SlotSummary = &SeasonPackSlotSummary{
			PrimarySlotID:   &primarySlotID,
			PrimarySlotName: slotNames[primarySlotID],
			IsMixedSlots:    hasMultipleSlots,
			SlotCounts:      slotCounts,
			SlotNames:       slotNames,
		}
	}

	return nil
}

// AnalyzeSeasonPackWithSlots scans and evaluates slots in one operation.
// This combines AnalyzeSeasonPack and EvaluateSeasonPackSlots.
func (s *Service) AnalyzeSeasonPackWithSlots(ctx context.Context, downloadPath string, seriesID *int64) (*SeasonPackAnalysis, error) {
	analysis, err := s.AnalyzeSeasonPack(ctx, downloadPath, seriesID)
	if err != nil {
		return nil, err
	}

	// Evaluate slots for each file
	if err := s.EvaluateSeasonPackSlots(ctx, analysis); err != nil {
		s.logger.Warn().Err(err).Msg("Failed to evaluate slots for season pack")
		// Don't fail the entire operation, just continue without slot info
	}

	return analysis, nil
}

// GetSeasonPackTargetSlots returns the target slot IDs for all matched files in a season pack.
// Returns a map from episode ID to slot ID for use during import.
func (s *Service) GetSeasonPackTargetSlots(analysis *SeasonPackAnalysis) map[int64]int64 {
	slots := make(map[int64]int64)

	for _, file := range analysis.Files {
		if file.EpisodeID != nil && file.TargetSlotID != nil {
			slots[*file.EpisodeID] = *file.TargetSlotID
		}
	}

	return slots
}
