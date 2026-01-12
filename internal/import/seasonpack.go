package importer

import (
	"context"
	"os"
	"path/filepath"

	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/downloader"
	"github.com/slipstream/slipstream/internal/library/scanner"
)

// SeasonPackFile represents a file within a season pack download.
type SeasonPackFile struct {
	Path         string              `json:"path"`
	Filename     string              `json:"filename"`
	Size         int64               `json:"size"`
	ParsedInfo   *scanner.ParsedMedia `json:"parsedInfo,omitempty"`
	EpisodeID    *int64              `json:"episodeId,omitempty"`
	SeriesID     *int64              `json:"seriesId,omitempty"`
	SeasonNumber int                 `json:"seasonNumber,omitempty"`
	EpisodeNum   int                 `json:"episodeNumber,omitempty"`
	EndEpisode   int                 `json:"endEpisode,omitempty"` // For multi-episode files
	IsReady      bool                `json:"isReady"`
	IsMatched    bool                `json:"isMatched"`
	Error        string              `json:"error,omitempty"`
}

// SeasonPackAnalysis contains the result of analyzing a season pack.
type SeasonPackAnalysis struct {
	DownloadPath   string            `json:"downloadPath"`
	SeriesID       *int64            `json:"seriesId,omitempty"`
	SeasonNumber   int               `json:"seasonNumber,omitempty"`
	TotalFiles     int               `json:"totalFiles"`
	ReadyFiles     int               `json:"readyFiles"`
	MatchedFiles   int               `json:"matchedFiles"`
	UnmatchedFiles int               `json:"unmatchedFiles"`
	Files          []SeasonPackFile  `json:"files"`
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

		// Parse the filename
		parsed := scanner.ParseFilename(packFile.Filename)
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
		}

		entry, err := s.downloader.CreateQueueMedia(ctx, input)
		if err != nil {
			s.logger.Warn().Err(err).Str("path", file.Path).Msg("Failed to create queue media entry")
			continue
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
	}

	if m.MovieID.Valid {
		mapping.MovieID = &m.MovieID.Int64
		mapping.MediaType = "movie"
	}

	if m.SeriesID.Valid {
		mapping.SeriesID = &m.SeriesID.Int64
		mapping.MediaType = "episode"
	}

	if m.SeasonNumber.Valid {
		sn := int(m.SeasonNumber.Int64)
		mapping.SeasonNumber = &sn
	}

	if m.EpisodeID.Valid {
		mapping.EpisodeID = &m.EpisodeID.Int64
	}

	return mapping
}

// IsSeasonPack determines if a download is a season pack.
func (s *Service) IsSeasonPack(mapping *DownloadMapping) bool {
	// A download is a season pack if:
	// 1. It's marked as a season pack in the mapping
	// 2. Or it has a series ID and season number but no specific episode ID
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
		parsed := scanner.ParseFilename(filepath.Base(file))
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
