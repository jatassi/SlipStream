package importer //nolint:revive // package name is established; renaming would be disruptive

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
	PrimarySlotID   *int64           `json:"primarySlotId,omitempty"`   // Most common slot
	PrimarySlotName string           `json:"primarySlotName,omitempty"` // Name of most common slot
	IsMixedSlots    bool             `json:"isMixedSlots"`              // True if episodes go to different slots
	SlotCounts      map[int64]int    `json:"slotCounts"`                // Count of episodes per slot
	SlotNames       map[int64]string `json:"slotNames"`                 // Names for display
}

// AnalyzeSeasonPack scans a download folder and identifies individual episodes.
func (s *Service) AnalyzeSeasonPack(ctx context.Context, downloadPath string, seriesID *int64) (*SeasonPackAnalysis, error) {
	result := &SeasonPackAnalysis{
		DownloadPath: downloadPath,
		SeriesID:     seriesID,
		Files:        make([]SeasonPackFile, 0),
	}

	files, err := s.findVideoFiles(downloadPath)
	if err != nil {
		return nil, err
	}

	result.TotalFiles = len(files)

	for _, filePath := range files {
		packFile := s.analyzeSeasonPackFile(ctx, filePath, seriesID)
		s.updateAnalysisCounters(result, &packFile)
		result.Files = append(result.Files, packFile)
	}

	return result, nil
}

func (s *Service) analyzeSeasonPackFile(ctx context.Context, filePath string, seriesID *int64) SeasonPackFile {
	packFile := SeasonPackFile{
		Path:     filePath,
		Filename: filepath.Base(filePath),
	}

	s.setFileSize(&packFile)
	s.parseSeasonPackFile(&packFile)
	s.checkFileReadiness(ctx, &packFile)
	s.matchFileToLibrary(ctx, &packFile, seriesID)

	return packFile
}

func (s *Service) setFileSize(packFile *SeasonPackFile) {
	stat, err := os.Stat(packFile.Path)
	if err == nil {
		packFile.Size = stat.Size()
	}
}

func (s *Service) parseSeasonPackFile(packFile *SeasonPackFile) {
	parsed := scanner.ParsePath(packFile.Path)
	packFile.ParsedInfo = parsed

	if parsed.Season > 0 {
		packFile.SeasonNumber = parsed.Season
	}
	if parsed.Episode > 0 {
		packFile.EpisodeNum = parsed.Episode
	}
	if parsed.EndEpisode > 0 && parsed.EndEpisode > parsed.Episode {
		packFile.EndEpisode = parsed.EndEpisode
	}
}

func (s *Service) checkFileReadiness(ctx context.Context, packFile *SeasonPackFile) {
	completion := s.CheckFileCompletion(ctx, packFile.Path)
	packFile.IsReady = completion.Status == CompletionReady

	if packFile.ParsedInfo.Season > 0 && packFile.SeasonNumber == 0 {
		packFile.SeasonNumber = packFile.ParsedInfo.Season
	}
}

func (s *Service) matchFileToLibrary(ctx context.Context, packFile *SeasonPackFile, seriesID *int64) {
	if seriesID == nil || packFile.EpisodeNum <= 0 {
		return
	}

	episode, err := s.tv.GetEpisodeByNumber(ctx, *seriesID, packFile.SeasonNumber, packFile.EpisodeNum)
	if err == nil && episode != nil {
		packFile.EpisodeID = &episode.ID
		packFile.SeriesID = seriesID
		packFile.IsMatched = true
		return
	}

	s.createMissingEpisode(ctx, packFile, seriesID)
}

func (s *Service) createMissingEpisode(ctx context.Context, packFile *SeasonPackFile, seriesID *int64) {
	title := packFile.ParsedInfo.Title
	if title == "" {
		title = fmt.Sprintf("Episode %d", packFile.EpisodeNum)
	}

	newEpisode, err := s.tv.CreateEpisode(ctx, *seriesID, packFile.SeasonNumber, packFile.EpisodeNum, title)
	if err != nil {
		s.logger.Warn().
			Err(err).
			Int64("seriesId", *seriesID).
			Int("season", packFile.SeasonNumber).
			Int("episode", packFile.EpisodeNum).
			Msg("Failed to create missing episode")
		return
	}

	if newEpisode != nil {
		packFile.EpisodeID = &newEpisode.ID
		packFile.SeriesID = seriesID
		packFile.IsMatched = true
		s.logger.Info().
			Int64("seriesId", *seriesID).
			Int("season", packFile.SeasonNumber).
			Int("episode", packFile.EpisodeNum).
			Str("filename", packFile.Filename).
			Msg("Created missing episode from season pack file")
	}
}

func (s *Service) updateAnalysisCounters(result *SeasonPackAnalysis, packFile *SeasonPackFile) {
	if packFile.IsReady {
		result.ReadyFiles++
	}
	if packFile.IsMatched {
		result.MatchedFiles++
	} else {
		result.UnmatchedFiles++
	}
	if packFile.SeasonNumber > 0 && result.SeasonNumber == 0 {
		result.SeasonNumber = packFile.SeasonNumber
	}
}

// CreateQueueMediaForSeasonPack creates queue_media entries for each file in a season pack.
// Req 16.2.3: Each episode from pack individually assessed with its own slot assignment.
func (s *Service) CreateQueueMediaForSeasonPack(ctx context.Context, mappingID int64, analysis *SeasonPackAnalysis) ([]*sqlc.QueueMedium, error) {
	entries := make([]*sqlc.QueueMedium, 0, len(analysis.Files))

	for i := range analysis.Files {
		file := &analysis.Files[i]
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
	entries, err := s.downloader.GetQueueMediaByDownloadMapping(ctx, mappingID)
	if err != nil {
		return 0, err
	}

	var imported int
	for _, entry := range entries {
		if !s.isReadyForImport(entry) {
			continue
		}

		if err := s.downloader.UpdateQueueMediaStatus(ctx, entry.ID, downloader.QueueMediaStatusImporting); err != nil {
			s.logger.Warn().Err(err).Int64("id", entry.ID).Msg("Failed to update queue media status")
			continue
		}

		if s.queueSeasonPackEntry(ctx, mappingID, entry) {
			imported++
		}
	}

	return imported, nil
}

func (s *Service) isReadyForImport(entry *sqlc.QueueMedium) bool {
	if entry.FileStatus != string(downloader.QueueMediaStatusReady) {
		return false
	}
	return entry.FilePath.Valid && entry.FilePath.String != ""
}

func (s *Service) queueSeasonPackEntry(ctx context.Context, mappingID int64, entry *sqlc.QueueMedium) bool {
	mapping, err := s.getDownloadMappingByID(ctx, mappingID)
	if err != nil {
		s.logger.Warn().Err(err).Int64("mappingId", mappingID).Msg("Failed to get download mapping")
		return false
	}

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
	if entry.TargetSlotID.Valid {
		job.TargetSlotID = &entry.TargetSlotID.Int64
	}

	if err := s.QueueImport(job); err != nil {
		s.logger.Warn().Err(err).Str("path", entry.FilePath.String).Msg("Failed to queue season pack file")
		_ = s.downloader.UpdateQueueMediaStatusWithError(ctx, entry.ID, downloader.QueueMediaStatusFailed, err.Error())
		return false
	}

	return true
}

// UpdateSeasonPackFileStatuses checks completion status and updates queue_media entries.
func (s *Service) UpdateSeasonPackFileStatuses(ctx context.Context, mappingID int64) error {
	entries, err := s.downloader.GetQueueMediaByDownloadMapping(ctx, mappingID)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		s.updateSingleFileStatus(ctx, entry)
	}

	return nil
}

func (s *Service) updateSingleFileStatus(ctx context.Context, entry *sqlc.QueueMedium) {
	if !entry.FilePath.Valid || entry.FilePath.String == "" {
		return
	}

	if isTerminalFileStatus(entry.FileStatus) {
		return
	}

	newStatus := s.mapCompletionToQueueStatus(ctx, entry.FilePath.String)
	if string(newStatus) != entry.FileStatus {
		if err := s.downloader.UpdateQueueMediaStatus(ctx, entry.ID, newStatus); err != nil {
			s.logger.Warn().Err(err).Int64("id", entry.ID).Msg("Failed to update queue media status")
		}
	}
}

func isTerminalFileStatus(status string) bool {
	return status == string(downloader.QueueMediaStatusImported) ||
		status == string(downloader.QueueMediaStatusFailed) ||
		status == string(downloader.QueueMediaStatusImporting)
}

func (s *Service) mapCompletionToQueueStatus(ctx context.Context, path string) downloader.QueueMediaStatus {
	completion := s.CheckFileCompletion(ctx, path)
	switch completion.Status {
	case CompletionReady:
		return downloader.QueueMediaStatusReady
	case CompletionNotFound:
		return downloader.QueueMediaStatusFailed
	default:
		return downloader.QueueMediaStatusPending
	}
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

	s.populateNullableFields(mapping, m)
	mapping.MediaType = determineMappingMediaType(mapping)
	return mapping
}

func (s *Service) populateNullableFields(mapping *DownloadMapping, m *sqlc.DownloadMapping) {
	if m.TargetSlotID.Valid {
		id := m.TargetSlotID.Int64
		mapping.TargetSlotID = &id
	}
	if m.MovieID.Valid {
		mapping.MovieID = &m.MovieID.Int64
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
}

func determineMappingMediaType(m *DownloadMapping) string {
	switch {
	case m.MovieID != nil:
		return mediaTypeMovie
	case m.IsCompleteSeries:
		return mediaSeries
	case m.IsSeasonPack || (m.SeasonNumber != nil && m.EpisodeID == nil):
		return mediaSeason
	case m.SeriesID != nil:
		return mediaTypeEpisode
	default:
		return ""
	}
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
func (s *Service) DetectSeasonPackFromPath(ctx context.Context, downloadPath string) (isSeasonPack bool, fileCount int, err error) {
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
	isSeasonPack = episodeCount > 1

	return isSeasonPack, len(files), nil
}

// EvaluateSeasonPackSlots evaluates slots for each file in a season pack analysis.
// Req 16.2.1: Season pack assigned to the slot it best matches
// Req 16.2.2: May result in mixed slots across seasons (acceptable)
// Req 16.2.3: Each episode from pack individually assessed
func (s *Service) EvaluateSeasonPackSlots(ctx context.Context, analysis *SeasonPackAnalysis) error {
	if s.slots == nil || !s.slots.IsMultiVersionEnabled(ctx) {
		return nil
	}

	tracker := newSlotTracker()

	for i := range analysis.Files {
		file := &analysis.Files[i]
		if file.ParsedInfo == nil || file.EpisodeID == nil {
			continue
		}
		s.evaluateFileSlot(ctx, file, tracker)
	}

	analysis.SlotSummary = tracker.buildSummary()
	return nil
}

type slotTracker struct {
	counts        map[int64]int
	names         map[int64]string
	firstSlotID   *int64
	hasMixedSlots bool
}

func newSlotTracker() *slotTracker {
	return &slotTracker{
		counts: make(map[int64]int),
		names:  make(map[int64]string),
	}
}

func (t *slotTracker) record(slotID int64, slotName string) {
	t.counts[slotID]++
	t.names[slotID] = slotName

	if t.firstSlotID == nil {
		t.firstSlotID = &slotID
	} else if *t.firstSlotID != slotID {
		t.hasMixedSlots = true
	}
}

func (t *slotTracker) buildSummary() *SeasonPackSlotSummary {
	if len(t.counts) == 0 {
		return nil
	}

	var primarySlotID int64
	maxCount := 0
	for slotID, count := range t.counts {
		if count > maxCount {
			maxCount = count
			primarySlotID = slotID
		}
	}

	return &SeasonPackSlotSummary{
		PrimarySlotID:   &primarySlotID,
		PrimarySlotName: t.names[primarySlotID],
		IsMixedSlots:    t.hasMixedSlots,
		SlotCounts:      t.counts,
		SlotNames:       t.names,
	}
}

func (s *Service) evaluateFileSlot(ctx context.Context, file *SeasonPackFile, tracker *slotTracker) {
	eval, err := s.slots.EvaluateRelease(ctx, file.ParsedInfo, "episode", *file.EpisodeID)
	if err != nil {
		s.logger.Debug().Err(err).Str("file", file.Filename).Msg("Failed to evaluate slots for season pack file")
		return
	}

	if len(eval.Assignments) == 0 {
		return
	}

	best := eval.Assignments[0]
	file.TargetSlotID = &best.SlotID
	file.TargetSlotName = best.SlotName
	file.SlotMatchScore = best.MatchScore
	file.IsSlotUpgrade = best.IsUpgrade
	file.IsSlotNewFill = best.IsNewFill

	tracker.record(best.SlotID, best.SlotName)
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

	for i := range analysis.Files {
		file := &analysis.Files[i]
		if file.EpisodeID != nil && file.TargetSlotID != nil {
			slots[*file.EpisodeID] = *file.TargetSlotID
		}
	}

	return slots
}
