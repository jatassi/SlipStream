package importer

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/slipstream/slipstream/internal/downloader"
	fsmock "github.com/slipstream/slipstream/internal/filesystem/mock"
	"github.com/slipstream/slipstream/internal/import/renamer"
	"github.com/slipstream/slipstream/internal/library/movies"
	"github.com/slipstream/slipstream/internal/library/organizer"
	"github.com/slipstream/slipstream/internal/library/quality"
	"github.com/slipstream/slipstream/internal/library/scanner"
	"github.com/slipstream/slipstream/internal/library/slots"
	"github.com/slipstream/slipstream/internal/library/tv"
	"github.com/slipstream/slipstream/internal/mediainfo"
)

var ErrNotApplicable = errors.New("not applicable")

// ProcessCompletedDownload processes a completed download from the queue.
func (s *Service) ProcessCompletedDownload(ctx context.Context, mapping *DownloadMapping) error {
	s.logger.Debug().
		Int64("mappingId", mapping.ID).
		Str("mediaType", mapping.MediaType).
		Msg("Processing completed download")

	if strings.HasPrefix(mapping.DownloadID, "mock-") {
		return s.processMockImport(ctx, mapping)
	}

	downloadPath, err := s.getDownloadPath(ctx, mapping)
	if err != nil {
		return err
	}

	files, err := s.findVideoFiles(downloadPath)
	if err != nil {
		return fmt.Errorf("failed to find video files: %w", err)
	}

	if len(files) == 0 {
		return fmt.Errorf("no video files found in %s", downloadPath)
	}

	s.queueFilesForImport(files, mapping)
	return nil
}

func (s *Service) getDownloadPath(ctx context.Context, mapping *DownloadMapping) (string, error) {
	client, err := s.downloader.GetClient(ctx, mapping.DownloadClientID)
	if err != nil {
		return "", fmt.Errorf("failed to get download client: %w", err)
	}

	items, err := client.List(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to list downloads: %w", err)
	}

	for i := range items {
		item := &items[i]
		if item.ID == mapping.DownloadID {
			return filepath.Join(item.DownloadDir, item.Name), nil
		}
	}

	return "", fmt.Errorf("could not find download path for ID %s", mapping.DownloadID)
}

func (s *Service) queueFilesForImport(files []string, mapping *DownloadMapping) {
	for _, file := range files {
		job := ImportJob{
			SourcePath:      file,
			DownloadMapping: mapping,
			Manual:          false,
		}
		if err := s.QueueImport(job); err != nil {
			s.logger.Warn().Err(err).Str("file", file).Msg("Failed to queue file for import")
		}
	}
}

// ProcessManualImport processes a manual import with a confirmed match.
func (s *Service) ProcessManualImport(ctx context.Context, sourcePath string, match *LibraryMatch, targetSlotID *int64) (*ImportResult, error) {
	job := ImportJob{
		SourcePath:     sourcePath,
		Manual:         true,
		ConfirmedMatch: match,
		TargetSlotID:   targetSlotID,
	}

	// Process synchronously for manual imports
	return s.processImport(ctx, job)
}

// SlipStreamSubdirs are the subdirectories where SlipStream places downloads.
// Only files in these directories should be scanned for import.
var SlipStreamSubdirs = []string{"SlipStream/Movies", "SlipStream/Series", "SlipStream"}

// ScanForPendingImports scans download folders for files ready to import.
// Only scans the SlipStream subdirectories to avoid importing downloads from other applications.
func (s *Service) ScanForPendingImports(ctx context.Context) error {
	s.logger.Info().Msg("Scanning for pending imports")

	libraryStats := s.loadLibraryFileStats(ctx)

	clients, err := s.downloader.List(ctx)
	if err != nil {
		return fmt.Errorf("failed to list download clients: %w", err)
	}

	for _, client := range clients {
		if !client.Enabled {
			continue
		}
		s.scanClientDownloads(ctx, client, libraryStats)
	}

	return nil
}

func (s *Service) scanClientDownloads(ctx context.Context, client *downloader.DownloadClient, libraryStats []libraryFileStat) {
	dlClient, err := s.downloader.GetClient(ctx, client.ID)
	if err != nil {
		s.logger.Warn().Err(err).Str("client", client.Name).Msg("Failed to get client")
		return
	}

	baseDir, err := dlClient.GetDownloadDir(ctx)
	if err != nil {
		s.logger.Warn().Err(err).Str("client", client.Name).Msg("Failed to get download dir")
		return
	}

	for _, subDir := range SlipStreamSubdirs {
		s.scanSubdirectory(ctx, baseDir, subDir, libraryStats)
	}
}

func (s *Service) scanSubdirectory(ctx context.Context, baseDir, subDir string, libraryStats []libraryFileStat) {
	slipstreamDir := filepath.Join(baseDir, subDir)

	if _, err := os.Stat(slipstreamDir); os.IsNotExist(err) {
		return
	}

	files, err := s.findVideoFiles(slipstreamDir)
	if err != nil {
		s.logger.Warn().Err(err).Str("path", slipstreamDir).Msg("Failed to scan for files")
		return
	}

	for _, file := range files {
		s.processFoundFile(ctx, file, libraryStats)
	}
}

func (s *Service) processFoundFile(ctx context.Context, file string, libraryStats []libraryFileStat) {
	if s.IsProcessing(file) {
		return
	}

	if s.isFileAlreadyImported(ctx, file, libraryStats) {
		return
	}

	job := ImportJob{
		SourcePath: file,
		Manual:     false,
	}

	if err := s.QueueImport(job); err != nil {
		s.logger.Debug().Err(err).Str("file", file).Msg("Failed to queue file")
	}
}

// libraryFileStat holds a pre-loaded file stat for efficient hardlink detection.
type libraryFileStat struct {
	path string
	info os.FileInfo
}

// loadLibraryFileStats loads and stats all library files for hardlink detection.
// Called once per scan cycle to avoid repeated DB queries and stat calls per download file.
func (s *Service) loadLibraryFileStats(ctx context.Context) []libraryFileStat {
	var stats []libraryFileStat

	moviePaths, err := s.queries.ListAllMovieFilePaths(ctx)
	if err != nil {
		s.logger.Warn().Err(err).Msg("Failed to load movie file paths for hardlink detection")
	} else {
		for _, p := range moviePaths {
			if info, err := os.Stat(p); err == nil {
				stats = append(stats, libraryFileStat{path: p, info: info})
			}
		}
	}

	episodePaths, err := s.queries.ListAllEpisodeFilePaths(ctx)
	if err != nil {
		s.logger.Warn().Err(err).Msg("Failed to load episode file paths for hardlink detection")
	} else {
		for _, p := range episodePaths {
			if info, err := os.Stat(p); err == nil {
				stats = append(stats, libraryFileStat{path: p, info: info})
			}
		}
	}

	s.logger.Debug().Int("count", len(stats)).Msg("Loaded library file stats for hardlink detection")
	return stats
}

// isFileAlreadyImported checks if a file has already been imported to the library.
// This prevents re-importing files that remain in the download folder after import (e.g., hardlink mode).
func (s *Service) isFileAlreadyImported(ctx context.Context, path string, libraryStats []libraryFileStat) bool {
	// Check import decisions table — if we previously evaluated and rejected this file, skip it
	decision, err := s.queries.GetImportDecision(ctx, path)
	if err == nil && decision.ID > 0 {
		s.logger.Debug().
			Str("path", path).
			Str("decision", decision.Decision).
			Msg("File has a previous import decision, skipping")
		return true
	}

	// Check if it was imported as a movie file (via original_path)
	nullPath := toNullString(path)
	movieImported, err := s.queries.IsOriginalPathImportedMovie(ctx, nullPath)
	if err == nil && movieImported != 0 {
		return true
	}

	// Check if it was imported as an episode file (via original_path)
	episodeImported, err := s.queries.IsOriginalPathImportedEpisode(ctx, nullPath)
	if err == nil && episodeImported != 0 {
		return true
	}

	// Check if this file is a hardlink to an existing library file using os.SameFile.
	// Uses pre-loaded stats so no DB queries or extra stat calls are needed here.
	return s.isHardlinkToLibraryFile(path, libraryStats)
}

// isHardlinkToLibraryFile checks if the given path is a hardlink to any existing library file.
// Uses pre-loaded library file stats for efficient comparison via os.SameFile.
func (s *Service) isHardlinkToLibraryFile(sourcePath string, libraryStats []libraryFileStat) bool {
	sourceStat, err := os.Stat(sourcePath)
	if err != nil {
		return false
	}

	for _, entry := range libraryStats {
		if os.SameFile(sourceStat, entry.info) {
			s.logger.Debug().
				Str("source", sourcePath).
				Str("libraryFile", entry.path).
				Msg("Source file is hardlink to existing library file")
			return true
		}
	}

	return false
}

// toNullString converts a string to sql.NullString, treating empty strings as null.
func toNullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

// isSameFile checks if two paths point to the same file (e.g., hardlinks).
// This prevents re-importing files that are already in the library.
func (s *Service) isSameFile(path1, path2 string) bool {
	stat1, err1 := os.Stat(path1)
	stat2, err2 := os.Stat(path2)

	if err1 != nil || err2 != nil {
		return false
	}

	return os.SameFile(stat1, stat2)
}

// processImport handles the actual import of a single file.
func (s *Service) processImport(ctx context.Context, job ImportJob) (*ImportResult, error) {
	result := &ImportResult{SourcePath: job.SourcePath}

	settings := s.loadAndApplySettings(ctx)

	if err := s.prepareImport(ctx, job, result, settings); err != nil {
		return result, err
	}

	isMultiVersion := s.slots != nil && s.slots.IsMultiVersionEnabled(ctx)
	targetSlotID, slotUpgradeFile, err := s.processSlotAssignment(ctx, job, result, isMultiVersion)
	if err != nil || result.RequiresSlotSelection {
		return result, err
	}

	if err := s.performFileImport(ctx, job, result); err != nil {
		return result, err
	}

	s.finalizeImport(ctx, job, result, targetSlotID, slotUpgradeFile, isMultiVersion)
	return result, nil
}

func (s *Service) prepareImport(ctx context.Context, job ImportJob, result *ImportResult, settings *ImportSettings) error {
	if err := s.validateFile(ctx, job.SourcePath, settings); err != nil {
		result.Error = err
		return err
	}

	match, err := s.resolveLibraryMatch(ctx, job, settings)
	if err != nil {
		result.Error = err
		return err
	}

	if err := s.populateRootFolder(ctx, match, job.TargetSlotID); err != nil {
		s.logger.Warn().Err(err).Msg("Failed to populate root folder, using match root folder")
	}
	result.Match = match

	if err := s.enforceUpgradePolicy(ctx, job, match); err != nil {
		result.Error = err
		return err
	}

	mediaInfo := &mediainfo.MediaInfo{}
	result.MediaInfo = mediaInfo

	destPath, err := s.computeDestination(ctx, match, mediaInfo, job.SourcePath)
	if err != nil {
		result.Error = err
		return err
	}
	result.DestinationPath = destPath

	if !job.Manual && s.isSameFile(job.SourcePath, destPath) {
		result.Error = ErrFileAlreadyInLibrary
		return result.Error
	}

	return nil
}

func (s *Service) processSlotAssignment(ctx context.Context, job ImportJob, result *ImportResult, isMultiVersion bool) (slotID, upgradeFileID *int64, err error) {
	targetSlotID, err := s.resolveSlotTarget(ctx, job, result.Match, result, isMultiVersion)
	if err != nil {
		return nil, nil, err
	}
	if result.RequiresSlotSelection {
		return nil, nil, nil
	}

	slotUpgradeFile := s.determineUpgradeStatus(ctx, result.Match, result, targetSlotID, isMultiVersion)
	return targetSlotID, slotUpgradeFile, nil
}

func (s *Service) performFileImport(ctx context.Context, job ImportJob, result *ImportResult) error {
	linkMode, err := s.executeImport(job.SourcePath, result.DestinationPath)
	if err != nil {
		result.Error = err
		return err
	}
	result.LinkMode = linkMode

	s.queueMediaInfoProbe(result.DestinationPath, result.Match)

	if result.Match.CandidateQualityID == 0 {
		s.resolveQualityID(ctx, result.Match, job.SourcePath)
	}

	return nil
}

func (s *Service) finalizeImport(ctx context.Context, job ImportJob, result *ImportResult, targetSlotID, slotUpgradeFile *int64, isMultiVersion bool) {
	fileID, updateErr := s.updateLibraryWithID(ctx, result.Match, result.DestinationPath, job.SourcePath, result.MediaInfo)
	if updateErr != nil && !errors.Is(updateErr, ErrNotApplicable) {
		s.logger.Warn().Err(updateErr).Msg("Failed to update library records")
	}

	s.assignFileToSlot(ctx, result.Match, targetSlotID, fileID, result)
	s.cleanupUpgradedFile(ctx, result.Match, result, slotUpgradeFile, result.DestinationPath, isMultiVersion)

	if err := s.logImportHistory(ctx, result); err != nil {
		s.logger.Warn().Err(err).Msg("Failed to log import history")
	}

	if s.health != nil {
		s.health.ClearStatusStr("import", job.SourcePath)
	}

	result.Success = true
}

// loadAndApplySettings loads import settings and updates the renamer.
func (s *Service) loadAndApplySettings(ctx context.Context) *ImportSettings {
	settings, err := s.GetSettings(ctx)
	if err != nil {
		s.logger.Warn().Err(err).Msg("Failed to load settings, using defaults")
		defaultSettings := DefaultImportSettings()
		settings = &defaultSettings
	}
	renamerSettings := settings.ToRenamerSettings()
	s.UpdateRenamerSettings(&renamerSettings)
	return settings
}

// resolveLibraryMatch resolves the import target, either from a confirmed match or by searching the library.
func (s *Service) resolveLibraryMatch(ctx context.Context, job ImportJob, settings *ImportSettings) (*LibraryMatch, error) {
	if job.ConfirmedMatch != nil {
		return job.ConfirmedMatch, nil
	}

	match, err := s.matchToLibraryWithSettings(ctx, job.SourcePath, job.DownloadMapping, settings)
	if err != nil {
		if errors.Is(err, ErrNoMatch) && settings.UnknownMediaBehavior == UnknownAutoAdd {
			s.logger.Warn().Str("path", job.SourcePath).Msg("Auto-add not yet implemented, file will be skipped")
		} else if errors.Is(err, ErrNoMatch) {
			s.logger.Debug().Str("path", job.SourcePath).Msg("No library match found, ignoring file per settings")
		}
		return nil, err
	}
	return match, nil
}

// enforceUpgradePolicy checks if the file is a quality upgrade (single-version mode)
// and rejects non-upgrades for automatic imports.
func (s *Service) enforceUpgradePolicy(ctx context.Context, job ImportJob, match *LibraryMatch) error {
	err := s.checkForExistingFile(ctx, match, job.SourcePath)
	if err == nil {
		return nil
	}
	if !errors.Is(err, ErrNotAnUpgrade) {
		s.logger.Debug().Err(err).Msg("checkForExistingFile returned error")
		return nil
	}
	if job.Manual {
		s.logger.Info().
			Str("path", job.SourcePath).
			Int("candidateQuality", match.CandidateQualityID).
			Int("existingQuality", match.ExistingQualityID).
			Msg("File is not a quality upgrade, but manual import — proceeding anyway")
		return nil
	}
	s.logger.Info().
		Str("path", job.SourcePath).
		Int("candidateQuality", match.CandidateQualityID).
		Int("existingQuality", match.ExistingQualityID).
		Msg("File is not a quality upgrade, rejecting")
	s.recordImportDecision(ctx, job.SourcePath, "not_upgrade", match)
	return err
}

// resolveSlotTarget evaluates slot assignment for multi-version imports.
// Returns the target slot ID and any file to upgrade, or sets RequiresSlotSelection on result.
//
//nolint:nilnil // nil slot ID with nil error means "no slot assignment needed"
func (s *Service) resolveSlotTarget(ctx context.Context, job ImportJob, match *LibraryMatch, result *ImportResult, isMultiVersion bool) (targetSlotID *int64, retErr error) {
	if !isMultiVersion {
		return nil, nil
	}

	slotResult, slotErr := s.evaluateSlotAssignment(ctx, job, match)
	if slotErr != nil && !errors.Is(slotErr, ErrNotApplicable) {
		s.logger.Warn().Err(slotErr).Msg("Slot evaluation failed, continuing without slot assignment")
		return nil, nil
	}
	if slotResult == nil {
		return nil, nil
	}

	result.SlotAssignments = slotResult.Assignments
	result.RecommendedSlotID = slotResult.RecommendedSlotID

	if slotResult.RequiresSelection && job.TargetSlotID == nil && job.Manual {
		result.RequiresSlotSelection = true
		return nil, nil
	}

	targetSlotID = s.selectTargetSlot(job, slotResult)

	if targetSlotID == nil && len(slotResult.Assignments) == 0 {
		s.recordImportDecision(ctx, job.SourcePath, "not_acceptable", match)
		err := ErrNotAnUpgrade
		result.Error = err
		return nil, err
	}

	return targetSlotID, nil
}

func (s *Service) selectTargetSlot(job ImportJob, slotResult *slotEvaluationResult) *int64 {
	if job.TargetSlotID != nil {
		return job.TargetSlotID
	}
	if slotResult.RecommendedSlotID != nil {
		return slotResult.RecommendedSlotID
	}
	return nil
}

// determineUpgradeStatus sets upgrade flags on the result based on slot evaluation or match state.
// Returns the slot upgrade file ID (for multi-version cleanup) or nil.
func (s *Service) determineUpgradeStatus(ctx context.Context, match *LibraryMatch, result *ImportResult, targetSlotID *int64, isMultiVersion bool) *int64 {
	if isMultiVersion {
		return s.determineSlotUpgrade(ctx, match, result, targetSlotID)
	}
	if match.IsUpgrade {
		result.IsUpgrade = true
		result.PreviousFile = match.ExistingFile
	}
	return nil
}

func (s *Service) determineSlotUpgrade(ctx context.Context, match *LibraryMatch, result *ImportResult, targetSlotID *int64) *int64 {
	if targetSlotID == nil {
		return nil
	}
	for _, a := range result.SlotAssignments {
		if a.SlotID != *targetSlotID {
			continue
		}
		if a.IsUpgrade {
			result.IsUpgrade = true
			fileID := s.getSlotFileID(ctx, match, *targetSlotID)
			if fileID != nil {
				if filePath := s.getFilePath(ctx, match.MediaType, *fileID); filePath != "" {
					result.PreviousFile = filePath
				}
			}
			return fileID
		}
		break
	}
	return nil
}

// queueMediaInfoProbe launches a background goroutine to probe MediaInfo for the imported file.
func (s *Service) queueMediaInfoProbe(destPath string, match *LibraryMatch) {
	if s.mediainfo == nil || !s.mediainfo.IsAvailable() {
		return
	}
	go s.runMediaInfoProbe(destPath, match)
}

func (s *Service) runMediaInfoProbe(path string, match *LibraryMatch) {
	probeCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	probedInfo, probeErr := s.mediainfo.Probe(probeCtx, path)
	if probeErr != nil {
		s.logger.Warn().Err(probeErr).Str("path", path).Msg("Background MediaInfo probe failed")
		return
	}
	if probedInfo == nil {
		return
	}

	s.updateMediaInfoForMatch(probeCtx, match, probedInfo)
}

func (s *Service) updateMediaInfoForMatch(ctx context.Context, match *LibraryMatch, probedInfo *mediainfo.MediaInfo) {
	if match.MediaType == mediaTypeMovie && match.MovieID != nil {
		if err := s.movies.UpdateFileMediaInfo(ctx, *match.MovieID, probedInfo); err != nil {
			s.logger.Warn().Err(err).Int64("movieId", *match.MovieID).Msg("Failed to update movie file MediaInfo")
		}
	} else if match.MediaType == mediaTypeEpisode && match.EpisodeID != nil {
		if err := s.tv.UpdateEpisodeFileMediaInfo(ctx, *match.EpisodeID, probedInfo); err != nil {
			s.logger.Warn().Err(err).Int64("episodeId", *match.EpisodeID).Msg("Failed to update episode file MediaInfo")
		}
	}
}

// assignFileToSlot assigns the imported file to a slot in multi-version mode.
func (s *Service) assignFileToSlot(ctx context.Context, match *LibraryMatch, targetSlotID, fileID *int64, result *ImportResult) {
	if targetSlotID == nil || fileID == nil || s.slots == nil {
		return
	}
	mediaID := s.getMediaIDFromMatch(match)
	if mediaID == nil {
		return
	}
	if err := s.slots.AssignFileToSlot(ctx, match.MediaType, *mediaID, *targetSlotID, *fileID); err != nil {
		s.logger.Warn().Err(err).Int64("slotId", *targetSlotID).Msg("Failed to assign file to slot")
	} else {
		result.AssignedSlotID = targetSlotID
		s.logger.Info().Int64("slotId", *targetSlotID).Int64("fileId", *fileID).Msg("File assigned to slot")
	}
}

// cleanupUpgradedFile deletes the old file and database record when an import is an upgrade.
func (s *Service) cleanupUpgradedFile(ctx context.Context, match *LibraryMatch, result *ImportResult, slotUpgradeFile *int64, destPath string, isMultiVersion bool) {
	if !result.IsUpgrade || result.PreviousFile == "" {
		return
	}
	if err := s.organizer.DeleteUpgradedFile(result.PreviousFile, destPath); err != nil {
		s.logger.Warn().Err(err).Str("file", result.PreviousFile).Msg("Failed to delete upgraded file")
	}

	oldFileID := s.getOldFileID(match, slotUpgradeFile, isMultiVersion)
	if oldFileID == nil {
		return
	}

	s.removeOldFileRecord(ctx, match.MediaType, *oldFileID)
}

func (s *Service) getOldFileID(match *LibraryMatch, slotUpgradeFile *int64, isMultiVersion bool) *int64 {
	if isMultiVersion && slotUpgradeFile != nil {
		return slotUpgradeFile
	}
	return match.ExistingFileID
}

func (s *Service) removeOldFileRecord(ctx context.Context, mediaType string, fileID int64) {
	switch mediaType {
	case mediaTypeMovie:
		if err := s.movies.RemoveFile(ctx, fileID); err != nil {
			s.logger.Warn().Err(err).Int64("fileId", fileID).Msg("Failed to remove old movie file record")
		}
	case mediaTypeEpisode:
		if err := s.tv.RemoveEpisodeFile(ctx, fileID); err != nil {
			s.logger.Warn().Err(err).Int64("fileId", fileID).Msg("Failed to remove old episode file record")
		}
	}
}

// computeDestination computes the full destination path for the file.
func (s *Service) computeDestination(
	ctx context.Context,
	match *LibraryMatch,
	mediaInfo *mediainfo.MediaInfo,
	sourcePath string,
) (string, error) {
	ext := filepath.Ext(sourcePath)
	tokenCtx := s.buildTokenContext(ctx, match, mediaInfo, sourcePath)

	if match.MediaType == mediaTypeMovie {
		return s.computeMovieDestination(tokenCtx, match.RootFolder, ext)
	}
	return s.computeEpisodeDestination(tokenCtx, match.RootFolder, ext)
}

func (s *Service) computeMovieDestination(tokenCtx *renamer.TokenContext, rootFolder, ext string) (string, error) {
	filename, err := s.renamer.ResolveMovieFilename(tokenCtx, ext)
	if err != nil {
		return "", fmt.Errorf("failed to resolve movie filename: %w", err)
	}

	movieFolder, err := s.renamer.ResolveMovieFolderName(tokenCtx)
	if err != nil {
		return "", fmt.Errorf("failed to resolve movie folder: %w", err)
	}

	folderPath := filepath.Join(rootFolder, movieFolder)
	fullPath, err := s.renamer.ResolveFullPath(folderPath, "", filename)
	if err != nil {
		return "", ErrPathTooLong
	}
	return fullPath, nil
}

func (s *Service) computeEpisodeDestination(tokenCtx *renamer.TokenContext, rootFolder, ext string) (string, error) {
	filename, err := s.renamer.ResolveEpisodeFilename(tokenCtx, ext)
	if err != nil {
		return "", fmt.Errorf("failed to resolve episode filename: %w", err)
	}

	seriesFolder, err := s.renamer.ResolveSeriesFolderName(tokenCtx)
	if err != nil {
		return "", fmt.Errorf("failed to resolve series folder: %w", err)
	}

	seasonFolder := s.renamer.ResolveSeasonFolderName(tokenCtx.SeasonNumber)
	folderPath := filepath.Join(rootFolder, seriesFolder, seasonFolder)
	fullPath, err := s.renamer.ResolveFullPath(folderPath, "", filename)
	if err != nil {
		return "", ErrPathTooLong
	}
	return fullPath, nil
}

// executeImport performs the actual file import using hardlink/copy.
func (s *Service) executeImport(source, dest string) (organizer.LinkMode, error) {
	// Ensure destination directory exists
	destDir := filepath.Dir(dest)
	if err := os.MkdirAll(destDir, 0o750); err != nil {
		return "", fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Use organizer's ImportFile which handles hardlink/symlink/copy fallback
	return s.organizer.ImportFile(source, dest)
}

// buildTokenContext creates a token context from match and mediainfo.
func (s *Service) buildTokenContext(
	ctx context.Context,
	match *LibraryMatch,
	mediaInfo *mediainfo.MediaInfo,
	sourcePath string,
) *renamer.TokenContext {
	filename := filepath.Base(sourcePath)
	parsed := scanner.ParsePath(sourcePath)

	tc := &renamer.TokenContext{
		OriginalFile:  filename,
		OriginalTitle: strings.TrimSuffix(filename, filepath.Ext(filename)),
	}

	s.applyParsedAttributes(tc, parsed)
	s.applyMediaInfo(tc, mediaInfo, parsed)

	if match.MediaType == mediaTypeMovie && match.MovieID != nil {
		s.applyMovieContext(ctx, tc, *match.MovieID)
	} else if match.MediaType == mediaTypeEpisode && match.SeriesID != nil {
		s.applySeriesContext(ctx, tc, match)
	}

	return tc
}

// applyParsedAttributes populates token context with parsed filename data.
func (s *Service) applyParsedAttributes(tc *renamer.TokenContext, parsed *scanner.ParsedMedia) {
	tc.Quality = parsed.Quality
	tc.Source = parsed.Source
	tc.Codec = parsed.Codec

	for _, attr := range parsed.Attributes {
		if s.isHDRAttribute(attr) {
			if tc.VideoDynamicRange == "" {
				tc.VideoDynamicRange = attr
			} else {
				tc.VideoDynamicRange += " " + attr
			}
		}
	}

	if len(parsed.AudioCodecs) > 0 {
		tc.AudioCodec = strings.Join(parsed.AudioCodecs, " ")
	}
	if len(parsed.AudioChannels) > 0 {
		tc.AudioChannels = strings.Join(parsed.AudioChannels, " ")
	}
	if len(parsed.AudioEnhancements) > 0 {
		enhancement := strings.Join(parsed.AudioEnhancements, " ")
		if tc.AudioCodec != "" {
			tc.AudioCodec += " " + enhancement
		} else {
			tc.AudioCodec = enhancement
		}
	}

	tc.ReleaseGroup = parsed.ReleaseGroup
	tc.Revision = parsed.Revision
	tc.EditionTags = parsed.Edition
}

// isHDRAttribute checks if an attribute is an HDR type.
func (s *Service) isHDRAttribute(attr string) bool {
	switch attr {
	case "HDR10+", "HDR10", "HDR", "DV", "Dolby Vision":
		return true
	default:
		return false
	}
}

// applyMediaInfo overrides token context with MediaInfo data (more accurate than filename parsing).
func (s *Service) applyMediaInfo(tc *renamer.TokenContext, mediaInfo *mediainfo.MediaInfo, parsed *scanner.ParsedMedia) {
	if mediaInfo.VideoCodec != "" {
		tc.VideoCodec = mediaInfo.VideoCodec
	} else {
		tc.VideoCodec = parsed.Codec
	}

	if mediaInfo.VideoBitDepth > 0 {
		tc.VideoBitDepth = mediaInfo.VideoBitDepth
	}
	if mediaInfo.DynamicRangeType != "" {
		tc.VideoDynamicRange = mediaInfo.DynamicRangeType
	}
	if mediaInfo.AudioCodec != "" {
		tc.AudioCodec = mediaInfo.AudioCodec
	}
	if mediaInfo.AudioChannels != "" {
		tc.AudioChannels = mediaInfo.AudioChannels
	}
	if len(mediaInfo.AudioLanguages) > 0 {
		tc.AudioLanguages = mediaInfo.AudioLanguages
	}
	if len(mediaInfo.SubtitleLanguages) > 0 {
		tc.SubtitleLanguages = mediaInfo.SubtitleLanguages
	}
}

// applyMovieContext populates token context with movie library data.
func (s *Service) applyMovieContext(ctx context.Context, tc *renamer.TokenContext, movieID int64) {
	movie, err := s.movies.Get(ctx, movieID)
	if err != nil {
		return
	}
	tc.MovieTitle = movie.Title
	tc.MovieYear = movie.Year
}

// applySeriesContext populates token context with series/episode library data.
func (s *Service) applySeriesContext(ctx context.Context, tc *renamer.TokenContext, match *LibraryMatch) {
	if match.SeriesID == nil {
		return
	}

	series, err := s.tv.GetSeries(ctx, *match.SeriesID)
	if err == nil {
		tc.SeriesTitle = series.Title
		tc.SeriesYear = series.Year
		tc.SeriesType = series.FormatType
		if tc.SeriesType == "" {
			tc.SeriesType = "standard"
		}
	}

	if match.SeasonNum != nil {
		tc.SeasonNumber = *match.SeasonNum
	}

	if match.EpisodeID != nil {
		s.applySingleEpisodeContext(ctx, tc, *match.EpisodeID)
	}

	if len(match.EpisodeIDs) > 1 {
		s.applyMultiEpisodeContext(ctx, tc, match.EpisodeIDs)
	}
}

// applySingleEpisodeContext populates token context with episode data.
func (s *Service) applySingleEpisodeContext(ctx context.Context, tc *renamer.TokenContext, episodeID int64) {
	episode, err := s.tv.GetEpisode(ctx, episodeID)
	if err != nil {
		return
	}
	tc.EpisodeNumber = episode.EpisodeNumber
	tc.EpisodeTitle = episode.Title
	if episode.AirDate != nil {
		tc.AirDate = *episode.AirDate
	}
}

// applyMultiEpisodeContext populates token context with multi-episode data.
func (s *Service) applyMultiEpisodeContext(ctx context.Context, tc *renamer.TokenContext, episodeIDs []int64) {
	tc.EpisodeNumbers = make([]int, 0, len(episodeIDs))
	for _, epID := range episodeIDs {
		ep, err := s.tv.GetEpisode(ctx, epID)
		if err != nil {
			continue
		}
		tc.EpisodeNumbers = append(tc.EpisodeNumbers, ep.EpisodeNumber)
	}
}

// logImportHistory logs the import to history.
func (s *Service) logImportHistory(ctx context.Context, result *ImportResult) error {
	if s.history == nil {
		return nil
	}

	var mediaID int64
	if result.Match.MediaType == mediaTypeMovie && result.Match.MovieID != nil {
		mediaID = *result.Match.MovieID
	} else if result.Match.EpisodeID != nil {
		mediaID = *result.Match.EpisodeID
	}

	parsed := scanner.ParsePath(result.SourcePath)

	data := map[string]any{
		"sourcePath":       result.SourcePath,
		"destinationPath":  result.DestinationPath,
		"originalFilename": filepath.Base(result.SourcePath),
		"finalFilename":    filepath.Base(result.DestinationPath),
		"linkMode":         string(result.LinkMode),
		"isUpgrade":        result.IsUpgrade,
		"previousFile":     result.PreviousFile,
	}

	if result.IsUpgrade {
		if q, ok := quality.GetQualityByID(result.Match.CandidateQualityID); ok {
			data["newQuality"] = q.Name
		}
		if q, ok := quality.GetQualityByID(result.Match.ExistingQualityID); ok {
			data["previousQuality"] = q.Name
		}
	}

	return s.history.Create(ctx, &HistoryInput{
		EventType: "imported",
		MediaType: result.Match.MediaType,
		MediaID:   mediaID,
		Quality:   parsed.Quality,
		Data:      data,
	})
}

// findVideoFiles finds all video files in a directory using configured extensions.
func (s *Service) findVideoFiles(dir string) ([]string, error) {
	return s.findVideoFilesWithSettings(context.Background(), dir, nil)
}

// findVideoFilesWithSettings finds video files using provided or loaded settings.
func (s *Service) findVideoFilesWithSettings(ctx context.Context, dir string, settings *ImportSettings) ([]string, error) {
	// Load settings if not provided
	if settings == nil {
		loaded, err := s.GetSettings(ctx)
		if err != nil {
			s.logger.Warn().Err(err).Msg("Failed to load settings for video scan, using defaults")
			defaults := DefaultImportSettings()
			settings = &defaults
		} else {
			settings = loaded
		}
	}

	var files []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return nil //nolint:nilerr // Skip filesystem errors during walk
		}

		if info.IsDir() {
			// Skip sample directories
			if strings.EqualFold(info.Name(), "sample") || strings.EqualFold(info.Name(), "samples") {
				return filepath.SkipDir
			}
			return nil
		}

		// Check extension against configured extensions
		ext := strings.ToLower(filepath.Ext(path))
		if settings.IsValidExtension(ext) {
			files = append(files, path)
		}

		return nil
	})

	return files, err
}

// applyImportDelay waits for the configured import delay.
func (s *Service) applyImportDelay(ctx context.Context, clientID int64) error {
	client, err := s.downloader.Get(ctx, clientID)
	if err != nil {
		return err
	}

	if client.ImportDelaySeconds > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(client.ImportDelaySeconds) * time.Second):
			return nil
		}
	}

	return nil
}

// slotEvaluationResult holds the result of slot evaluation.
type slotEvaluationResult struct {
	Assignments       []SlotAssignment
	RecommendedSlotID *int64
	RequiresSelection bool
}

// evaluateSlotAssignment evaluates which slot a file should be assigned to.
// Req 5.1.1-5.1.5: Evaluate release against slot profiles and determine target.
func (s *Service) evaluateSlotAssignment(ctx context.Context, job ImportJob, match *LibraryMatch) (*slotEvaluationResult, error) {
	if s.slots == nil {
		return nil, ErrNotApplicable
	}

	parsed := scanner.ParsePath(job.SourcePath)
	mediaType, mediaID, err := s.extractMediaInfo(match)
	if err != nil {
		return nil, err
	}

	eval, err := s.slots.EvaluateRelease(ctx, parsed, mediaType, mediaID)
	if err != nil {
		return nil, err
	}

	if eval == nil || len(eval.Assignments) == 0 {
		return nil, ErrNotApplicable
	}

	return s.convertToImportSlotResult(eval), nil
}

func (s *Service) extractMediaInfo(match *LibraryMatch) (mediaType string, mediaID int64, err error) {
	switch {
	case match.MediaType == mediaTypeMovie && match.MovieID != nil:
		return mediaTypeMovie, *match.MovieID, nil
	case match.MediaType == mediaTypeEpisode && match.EpisodeID != nil:
		return mediaTypeEpisode, *match.EpisodeID, nil
	default:
		return "", 0, ErrNotApplicable
	}
}

func (s *Service) convertToImportSlotResult(eval *slots.SlotEvaluation) *slotEvaluationResult {
	assignments := make([]SlotAssignment, 0, len(eval.Assignments))
	for _, a := range eval.Assignments {
		assignments = append(assignments, SlotAssignment{
			SlotID:     a.SlotID,
			SlotNumber: a.SlotNumber,
			SlotName:   a.SlotName,
			MatchScore: a.MatchScore,
			IsUpgrade:  a.IsUpgrade,
			IsNewFill:  a.IsNewFill,
		})
	}

	result := &slotEvaluationResult{
		Assignments:       assignments,
		RequiresSelection: eval.RequiresSelection,
	}

	if eval.RecommendedSlotID != 0 {
		result.RecommendedSlotID = &eval.RecommendedSlotID
	}

	return result
}

// resolveQualityID parses the source filename and resolves a quality ID against the media's profile.
// Sets match.CandidateQualityID if a quality can be determined.
func (s *Service) resolveQualityID(ctx context.Context, match *LibraryMatch, sourcePath string) {
	if s.quality == nil {
		return
	}

	qualityProfileID := s.getQualityProfileID(ctx, match)
	if qualityProfileID == 0 {
		return
	}

	profile, err := s.quality.Get(ctx, qualityProfileID)
	if err != nil {
		return
	}

	parsed := scanner.ParsePath(sourcePath)
	candidateMatch := quality.MatchQuality(parsed.Quality, parsed.Source, profile)
	if candidateMatch.Matches {
		match.CandidateQualityID = candidateMatch.MatchedQualityID
		match.QualityProfileID = qualityProfileID
	}
}

func (s *Service) getQualityProfileID(ctx context.Context, match *LibraryMatch) int64 {
	if match.QualityProfileID > 0 {
		return match.QualityProfileID
	}
	if match.MediaType == mediaTypeMovie && match.MovieID != nil {
		if movie, err := s.movies.Get(ctx, *match.MovieID); err == nil {
			return movie.QualityProfileID
		}
	}
	if match.MediaType == mediaTypeEpisode && match.SeriesID != nil {
		if series, err := s.tv.GetSeries(ctx, *match.SeriesID); err == nil {
			return series.QualityProfileID
		}
	}
	return 0
}

// updateLibraryWithID updates the library and returns the created file ID.
// sourcePath is the original file path before import (for duplicate detection).
func (s *Service) updateLibraryWithID(ctx context.Context, match *LibraryMatch, destPath, sourcePath string, mediaInfo *mediainfo.MediaInfo) (*int64, error) {
	stat, err := os.Stat(destPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat destination file: %w", err)
	}

	s.removeExistingFileRecord(ctx, match)

	parsed := scanner.ParsePath(sourcePath)
	qualityStr := parsed.Quality
	qualityID := s.getQualityIDPtr(match.CandidateQualityID)

	if match.MediaType == mediaTypeMovie && match.MovieID != nil {
		return s.addMovieFile(ctx, match, destPath, sourcePath, stat, qualityStr, qualityID, mediaInfo)
	}
	if match.MediaType == mediaTypeEpisode && match.EpisodeID != nil {
		return s.addEpisodeFile(ctx, match, destPath, sourcePath, stat, qualityStr, qualityID, mediaInfo)
	}

	return nil, ErrNotApplicable
}

func (s *Service) removeExistingFileRecord(ctx context.Context, match *LibraryMatch) {
	if match.ExistingFileID == nil {
		return
	}
	switch match.MediaType {
	case mediaTypeMovie:
		_ = s.movies.RemoveFile(ctx, *match.ExistingFileID)
	case mediaTypeEpisode:
		_ = s.tv.RemoveEpisodeFile(ctx, *match.ExistingFileID)
	}
	match.ExistingFileID = nil
}

func (s *Service) getQualityIDPtr(candidateQualityID int) *int64 {
	if candidateQualityID > 0 {
		id := int64(candidateQualityID)
		return &id
	}
	return nil
}

func (s *Service) addMovieFile(ctx context.Context, match *LibraryMatch, destPath, sourcePath string, stat os.FileInfo, qualityStr string, qualityID *int64, mediaInfo *mediainfo.MediaInfo) (*int64, error) {
	file, err := s.movies.AddFile(ctx, *match.MovieID, &movies.CreateMovieFileInput{
		Path:             destPath,
		Size:             stat.Size(),
		Quality:          qualityStr,
		QualityID:        qualityID,
		VideoCodec:       mediaInfo.VideoCodec,
		AudioCodec:       mediaInfo.AudioCodec,
		AudioChannels:    mediaInfo.AudioChannels,
		DynamicRange:     mediaInfo.DynamicRangeType,
		Resolution:       mediaInfo.VideoResolution,
		OriginalPath:     sourcePath,
		OriginalFilename: filepath.Base(sourcePath),
	})
	if err != nil {
		return nil, err
	}
	return &file.ID, nil
}

func (s *Service) addEpisodeFile(ctx context.Context, match *LibraryMatch, destPath, sourcePath string, stat os.FileInfo, qualityStr string, qualityID *int64, mediaInfo *mediainfo.MediaInfo) (*int64, error) {
	file, err := s.tv.AddEpisodeFile(ctx, *match.EpisodeID, &tv.CreateEpisodeFileInput{
		Path:             destPath,
		Size:             stat.Size(),
		Quality:          qualityStr,
		QualityID:        qualityID,
		VideoCodec:       mediaInfo.VideoCodec,
		AudioCodec:       mediaInfo.AudioCodec,
		AudioChannels:    mediaInfo.AudioChannels,
		DynamicRange:     mediaInfo.DynamicRangeType,
		Resolution:       mediaInfo.VideoResolution,
		OriginalPath:     sourcePath,
		OriginalFilename: filepath.Base(sourcePath),
	})
	if err != nil {
		return nil, err
	}
	return &file.ID, nil
}

// getMediaIDFromMatch extracts the media ID from a library match.
func (s *Service) getMediaIDFromMatch(match *LibraryMatch) *int64 {
	if match.MediaType == mediaTypeMovie && match.MovieID != nil {
		return match.MovieID
	} else if match.MediaType == mediaTypeEpisode && match.EpisodeID != nil {
		return match.EpisodeID
	}
	return nil
}

// processMockImport handles imports for mock downloads in developer mode.
// Creates file entries in the database and virtual filesystem without actual file operations.
func (s *Service) processMockImport(ctx context.Context, mapping *DownloadMapping) error {
	s.logger.Info().
		Str("downloadId", mapping.DownloadID).
		Str("mediaType", mapping.MediaType).
		Msg("Processing mock import (dev mode)")

	vfs := fsmock.GetInstance()

	if err := s.processMockImportByType(ctx, mapping, vfs); err != nil {
		return err
	}

	s.cleanupMockDownload(ctx, mapping)
	return nil
}

func (s *Service) processMockImportByType(ctx context.Context, mapping *DownloadMapping, vfs *fsmock.VirtualFS) error {
	switch mapping.MediaType {
	case mediaTypeMovie:
		if mapping.MovieID != nil {
			return s.processMockMovieImport(ctx, mapping, vfs)
		}
	case mediaTypeEpisode, mediaSeason:
		if mapping.SeriesID != nil {
			return s.processMockTVImport(ctx, mapping, vfs)
		}
	case mediaSeries:
		if mapping.SeriesID != nil {
			return s.processMockCompleteSeriesImport(ctx, mapping, vfs)
		}
	}
	return nil
}

func (s *Service) cleanupMockDownload(ctx context.Context, mapping *DownloadMapping) {
	if err := s.downloader.DeleteDownloadMapping(ctx, mapping.DownloadClientID, mapping.DownloadID); err != nil {
		s.logger.Warn().Err(err).Msg("Failed to delete download mapping after mock import")
	}

	client, err := s.downloader.GetClient(ctx, mapping.DownloadClientID)
	if err == nil {
		if err := client.Remove(ctx, mapping.DownloadID, false); err != nil {
			s.logger.Warn().Err(err).Msg("Failed to remove mock download after import")
		}
	}
}

// processMockMovieImport handles mock import for movies.
func (s *Service) processMockMovieImport(ctx context.Context, mapping *DownloadMapping, vfs *fsmock.VirtualFS) error {
	movie, err := s.movies.Get(ctx, *mapping.MovieID)
	if err != nil {
		return fmt.Errorf("failed to get movie for mock import: %w", err)
	}

	// Use the movie's configured path or create one
	basePath := movie.Path
	if basePath == "" {
		basePath = fmt.Sprintf("%s/%s (%d)", fsmock.MockMoviesPath, movie.Title, movie.Year)
	}

	// Create file path
	mockFilePath := fmt.Sprintf("%s/%s (%d).mkv", basePath, movie.Title, movie.Year)
	fileSize := int64(10 * 1024 * 1024 * 1024) // 10 GB

	// Add file to virtual filesystem
	vfs.AddFile(mockFilePath, fileSize)

	// Add file record to database
	file, err := s.movies.AddFile(ctx, *mapping.MovieID, &movies.CreateMovieFileInput{
		Path:       mockFilePath,
		Size:       fileSize,
		Quality:    "1080p",
		VideoCodec: "x265",
		Resolution: "1920x1080",
	})
	if err != nil {
		return fmt.Errorf("failed to add movie file for mock import: %w", err)
	}

	// Assign to slot if multi-version mode and target slot specified
	if mapping.TargetSlotID != nil && s.slots != nil {
		if err := s.slots.AssignFileToSlot(ctx, "movie", *mapping.MovieID, *mapping.TargetSlotID, file.ID); err != nil {
			s.logger.Warn().Err(err).Int64("slotId", *mapping.TargetSlotID).Msg("Failed to assign movie file to slot")
		} else {
			s.logger.Debug().Int64("slotId", *mapping.TargetSlotID).Int64("fileId", file.ID).Msg("Assigned movie file to slot")
		}
	}

	s.logger.Info().
		Int64("movieId", *mapping.MovieID).
		Str("title", movie.Title).
		Str("mockPath", mockFilePath).
		Msg("Mock movie import completed")

	// Update portal request status to available
	if s.statusTracker != nil {
		if err := s.statusTracker.OnMovieAvailable(ctx, *mapping.MovieID); err != nil {
			s.logger.Warn().Err(err).Int64("movieId", *mapping.MovieID).Msg("Failed to update request status")
		}
	}

	// Broadcast import event
	if s.hub != nil {
		s.hub.Broadcast("import:completed", map[string]any{
			"mediaType":       "movie",
			"movieId":         mapping.MovieID,
			"title":           movie.Title,
			"destinationPath": mockFilePath,
			"isMock":          true,
		})
		// Also broadcast movie update so detail page refreshes
		s.hub.Broadcast("movie:updated", map[string]any{
			"movieId": mapping.MovieID,
		})
	}

	return nil
}

// processMockTVImport handles mock import for TV episodes and season packs.
func (s *Service) processMockTVImport(ctx context.Context, mapping *DownloadMapping, vfs *fsmock.VirtualFS) error {
	series, err := s.tv.GetSeries(ctx, *mapping.SeriesID)
	if err != nil {
		return fmt.Errorf("failed to get series for mock import: %w", err)
	}

	// Use the series' configured path or create one
	basePath := series.Path
	if basePath == "" {
		basePath = fmt.Sprintf("%s/%s", fsmock.MockTVPath, series.Title)
	}

	// Determine if this is a season pack or single episode
	isSeasonPack := mapping.MediaType == mediaSeason || (mapping.SeasonNumber != nil && mapping.EpisodeID == nil)

	if isSeasonPack && mapping.SeasonNumber != nil {
		// Season pack: create files for all episodes in the season
		return s.processMockSeasonPackImport(ctx, mapping, series, basePath, vfs)
	} else if mapping.EpisodeID != nil {
		// Single episode
		return s.processMockSingleEpisodeImport(ctx, mapping, series, basePath, vfs)
	}

	s.logger.Warn().
		Str("downloadId", mapping.DownloadID).
		Msg("Mock TV import: no episode ID or season number specified")
	return nil
}

// processMockSeasonPackImport creates files for all episodes in a season.
func (s *Service) processMockSeasonPackImport(ctx context.Context, mapping *DownloadMapping, series *tv.Series, basePath string, vfs *fsmock.VirtualFS) error {
	seasonNum := *mapping.SeasonNumber

	episodes, err := s.tv.ListEpisodes(ctx, *mapping.SeriesID, &seasonNum)
	if err != nil {
		return fmt.Errorf("failed to list episodes for season %d: %w", seasonNum, err)
	}

	if len(episodes) == 0 {
		s.logger.Warn().
			Int64("seriesId", *mapping.SeriesID).
			Int(mediaSeason, seasonNum).
			Msg("No episodes found for season pack import")
		return nil
	}

	seasonPath := fmt.Sprintf("%s/Season %02d", basePath, seasonNum)
	vfs.AddDirectory(seasonPath)

	importedCount := s.importSeasonEpisodes(ctx, mapping, series, episodes, seasonPath, seasonNum, vfs)

	s.logger.Info().
		Int64("seriesId", *mapping.SeriesID).
		Str("title", series.Title).
		Int(mediaSeason, seasonNum).
		Int("episodesImported", importedCount).
		Msg("Mock season pack import completed")

	s.broadcastSeasonPackImport(mapping, series, seasonNum, importedCount)
	return nil
}

func (s *Service) importSeasonEpisodes(ctx context.Context, mapping *DownloadMapping, series *tv.Series, episodes []tv.Episode, seasonPath string, seasonNum int, vfs *fsmock.VirtualFS) int {
	fileSize := int64(2 * 1024 * 1024 * 1024)
	importedCount := 0

	for _, ep := range episodes {
		mockFilePath := fmt.Sprintf("%s/%s - S%02dE%02d.mkv",
			seasonPath, series.Title, seasonNum, ep.EpisodeNumber)

		vfs.AddFile(mockFilePath, fileSize)

		file, err := s.tv.AddEpisodeFile(ctx, ep.ID, &tv.CreateEpisodeFileInput{
			Path:       mockFilePath,
			Size:       fileSize,
			Quality:    "1080p",
			VideoCodec: "x265",
			Resolution: "1920x1080",
		})
		if err != nil {
			s.logger.Warn().Err(err).
				Int64("episodeId", ep.ID).
				Int("episode", ep.EpisodeNumber).
				Msg("Failed to add episode file")
			continue
		}

		s.assignMockEpisodeToSlot(ctx, mapping, ep.ID, file.ID)
		s.updateMockEpisodeRequestStatus(ctx, ep.ID)
		importedCount++
	}

	return importedCount
}

func (s *Service) broadcastSeasonPackImport(mapping *DownloadMapping, series *tv.Series, seasonNum, importedCount int) {
	if s.hub == nil {
		return
	}
	s.hub.Broadcast("import:completed", map[string]any{
		"mediaType":        mediaSeason,
		"seriesId":         mapping.SeriesID,
		"seriesTitle":      series.Title,
		"seasonNumber":     seasonNum,
		"episodesImported": importedCount,
		"isMock":           true,
	})
	s.hub.Broadcast("series:updated", map[string]any{
		"seriesId": mapping.SeriesID,
	})
}

// processMockSingleEpisodeImport creates a file for a single episode.
func (s *Service) processMockSingleEpisodeImport(ctx context.Context, mapping *DownloadMapping, series *tv.Series, basePath string, vfs *fsmock.VirtualFS) error {
	episode, err := s.tv.GetEpisode(ctx, *mapping.EpisodeID)
	if err != nil {
		return fmt.Errorf("failed to get episode for mock import: %w", err)
	}

	seasonPath := fmt.Sprintf("%s/Season %02d", basePath, episode.SeasonNumber)
	vfs.AddDirectory(seasonPath)

	mockFilePath := fmt.Sprintf("%s/%s - S%02dE%02d.mkv",
		seasonPath, series.Title, episode.SeasonNumber, episode.EpisodeNumber)
	fileSize := int64(2 * 1024 * 1024 * 1024) // 2 GB

	// Add file to virtual filesystem
	vfs.AddFile(mockFilePath, fileSize)

	// Add file record to database
	file, err := s.tv.AddEpisodeFile(ctx, *mapping.EpisodeID, &tv.CreateEpisodeFileInput{
		Path:       mockFilePath,
		Size:       fileSize,
		Quality:    "1080p",
		VideoCodec: "x265",
		Resolution: "1920x1080",
	})
	if err != nil {
		return fmt.Errorf("failed to add episode file for mock import: %w", err)
	}

	// Assign to slot if multi-version mode and target slot specified
	if mapping.TargetSlotID != nil && s.slots != nil {
		if err := s.slots.AssignFileToSlot(ctx, "episode", *mapping.EpisodeID, *mapping.TargetSlotID, file.ID); err != nil {
			s.logger.Warn().Err(err).Int64("slotId", *mapping.TargetSlotID).Msg("Failed to assign episode file to slot")
		} else {
			s.logger.Debug().Int64("slotId", *mapping.TargetSlotID).Int64("fileId", file.ID).Msg("Assigned episode file to slot")
		}
	}

	s.logger.Info().
		Int64("seriesId", *mapping.SeriesID).
		Str("title", series.Title).
		Int(mediaSeason, episode.SeasonNumber).
		Int("episode", episode.EpisodeNumber).
		Str("mockPath", mockFilePath).
		Msg("Mock episode import completed")

	// Update portal request status to available
	if s.statusTracker != nil {
		if err := s.statusTracker.OnEpisodeAvailable(ctx, *mapping.EpisodeID); err != nil {
			s.logger.Warn().Err(err).Int64("episodeId", *mapping.EpisodeID).Msg("Failed to update request status")
		}
	}

	// Broadcast import event
	if s.hub != nil {
		s.hub.Broadcast("import:completed", map[string]any{
			"mediaType":       "episode",
			"seriesId":        mapping.SeriesID,
			"seriesTitle":     series.Title,
			"episodeId":       mapping.EpisodeID,
			"episodeTitle":    episode.Title,
			"seasonNumber":    episode.SeasonNumber,
			"episodeNumber":   episode.EpisodeNumber,
			"destinationPath": mockFilePath,
			"isMock":          true,
		})
		// Also broadcast series update so detail page refreshes
		s.hub.Broadcast("series:updated", map[string]any{
			"seriesId": mapping.SeriesID,
		})
	}

	return nil
}

// processMockCompleteSeriesImport creates files for all episodes in all seasons of a series.
func (s *Service) processMockCompleteSeriesImport(ctx context.Context, mapping *DownloadMapping, vfs *fsmock.VirtualFS) error {
	series, err := s.tv.GetSeries(ctx, *mapping.SeriesID)
	if err != nil {
		return fmt.Errorf("failed to get series for mock import: %w", err)
	}

	basePath := series.Path
	if basePath == "" {
		basePath = fmt.Sprintf("%s/%s", fsmock.MockTVPath, series.Title)
	}

	seasons, err := s.tv.ListSeasons(ctx, *mapping.SeriesID)
	if err != nil {
		return fmt.Errorf("failed to list seasons for series: %w", err)
	}

	if len(seasons) == 0 {
		s.logger.Warn().
			Int64("seriesId", *mapping.SeriesID).
			Msg("No seasons found for complete series import")
		return nil
	}

	totalImported := s.importAllSeasons(ctx, mapping, series, seasons, basePath, vfs)
	s.broadcastCompleteSeriesImport(series, seasons, totalImported, mapping)

	return nil
}

func (s *Service) importAllSeasons(ctx context.Context, mapping *DownloadMapping, series *tv.Series, seasons []tv.Season, basePath string, vfs *fsmock.VirtualFS) int {
	fileSize := int64(2 * 1024 * 1024 * 1024)
	totalImported := 0

	for i := range seasons {
		season := &seasons[i]
		imported := s.importMockSeason(ctx, mapping, series, season, basePath, fileSize, vfs)
		totalImported += imported
	}

	s.logger.Info().
		Int64("seriesId", *mapping.SeriesID).
		Str("title", series.Title).
		Int("totalImported", totalImported).
		Msg("Mock complete series import completed")

	return totalImported
}

func (s *Service) importMockSeason(ctx context.Context, mapping *DownloadMapping, series *tv.Series, season *tv.Season, basePath string, fileSize int64, vfs *fsmock.VirtualFS) int {
	seasonNum := season.SeasonNumber
	seasonPath := fmt.Sprintf("%s/Season %02d", basePath, seasonNum)
	vfs.AddDirectory(seasonPath)

	episodes, err := s.tv.ListEpisodes(ctx, *mapping.SeriesID, &seasonNum)
	if err != nil {
		s.logger.Warn().Err(err).Int(mediaSeason, seasonNum).Msg("Failed to list episodes for season")
		return 0
	}

	imported := 0
	for i := range episodes {
		ep := &episodes[i]
		if s.importMockEpisode(ctx, mapping, series, ep, seasonPath, seasonNum, fileSize, vfs) {
			imported++
		}
	}

	return imported
}

func (s *Service) importMockEpisode(ctx context.Context, mapping *DownloadMapping, series *tv.Series, ep *tv.Episode, seasonPath string, seasonNum int, fileSize int64, vfs *fsmock.VirtualFS) bool {
	mockFilePath := fmt.Sprintf("%s/%s - S%02dE%02d.mkv",
		seasonPath, series.Title, seasonNum, ep.EpisodeNumber)

	vfs.AddFile(mockFilePath, fileSize)

	file, err := s.tv.AddEpisodeFile(ctx, ep.ID, &tv.CreateEpisodeFileInput{
		Path:       mockFilePath,
		Size:       fileSize,
		Quality:    "1080p",
		VideoCodec: "x265",
		Resolution: "1920x1080",
	})
	if err != nil {
		s.logger.Warn().Err(err).
			Int64("episodeId", ep.ID).
			Int(mediaSeason, seasonNum).
			Int("episode", ep.EpisodeNumber).
			Msg("Failed to add episode file")
		return false
	}

	s.assignMockEpisodeToSlot(ctx, mapping, ep.ID, file.ID)
	s.updateMockEpisodeRequestStatus(ctx, ep.ID)
	return true
}

func (s *Service) assignMockEpisodeToSlot(ctx context.Context, mapping *DownloadMapping, episodeID, fileID int64) {
	if mapping.TargetSlotID == nil || s.slots == nil {
		return
	}
	if err := s.slots.AssignFileToSlot(ctx, "episode", episodeID, *mapping.TargetSlotID, fileID); err != nil {
		s.logger.Warn().Err(err).Int64("slotId", *mapping.TargetSlotID).Int64("episodeId", episodeID).Msg("Failed to assign episode file to slot")
	}
}

func (s *Service) updateMockEpisodeRequestStatus(ctx context.Context, episodeID int64) {
	if s.statusTracker != nil {
		if err := s.statusTracker.OnEpisodeAvailable(ctx, episodeID); err != nil {
			s.logger.Warn().Err(err).Int64("episodeId", episodeID).Msg("Failed to update request status")
		}
	}
}

func (s *Service) broadcastCompleteSeriesImport(series *tv.Series, seasons []tv.Season, totalImported int, mapping *DownloadMapping) {
	if s.hub == nil {
		return
	}

	s.hub.Broadcast("import:completed", map[string]any{
		"mediaType":        mediaSeries,
		"seriesId":         mapping.SeriesID,
		"seriesTitle":      series.Title,
		"seasonsImported":  len(seasons),
		"episodesImported": totalImported,
		"isMock":           true,
	})
	s.hub.Broadcast("series:updated", map[string]any{
		"seriesId": mapping.SeriesID,
	})
}
