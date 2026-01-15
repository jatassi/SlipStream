package importer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/slipstream/slipstream/internal/import/renamer"
	"github.com/slipstream/slipstream/internal/library/movies"
	"github.com/slipstream/slipstream/internal/library/organizer"
	"github.com/slipstream/slipstream/internal/library/scanner"
	"github.com/slipstream/slipstream/internal/library/tv"
	"github.com/slipstream/slipstream/internal/mediainfo"
)

// ProcessCompletedDownload processes a completed download from the queue.
func (s *Service) ProcessCompletedDownload(ctx context.Context, mapping *DownloadMapping) error {
	s.logger.Debug().
		Int64("mappingId", mapping.ID).
		Str("mediaType", mapping.MediaType).
		Msg("Processing completed download")

	// Get the download path from the download client
	client, err := s.downloader.GetClient(ctx, mapping.DownloadClientID)
	if err != nil {
		return fmt.Errorf("failed to get download client: %w", err)
	}

	// Get download info to find the path
	items, err := client.List(ctx)
	if err != nil {
		return fmt.Errorf("failed to list downloads: %w", err)
	}

	var downloadPath string
	for _, item := range items {
		if item.ID == mapping.DownloadID {
			downloadPath = item.DownloadDir
			break
		}
	}

	if downloadPath == "" {
		return fmt.Errorf("could not find download path for ID %s", mapping.DownloadID)
	}

	// Find video files in the download path
	files, err := s.findVideoFiles(downloadPath)
	if err != nil {
		return fmt.Errorf("failed to find video files: %w", err)
	}

	if len(files) == 0 {
		return fmt.Errorf("no video files found in %s", downloadPath)
	}

	// Queue each file for import
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

	return nil
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

// ScanForPendingImports scans download folders for files ready to import.
func (s *Service) ScanForPendingImports(ctx context.Context) error {
	s.logger.Info().Msg("Scanning for pending imports")

	// Get all download clients
	clients, err := s.downloader.List(ctx)
	if err != nil {
		return fmt.Errorf("failed to list download clients: %w", err)
	}

	for _, client := range clients {
		if !client.Enabled {
			continue
		}

		// Get the client interface
		dlClient, err := s.downloader.GetClient(ctx, client.ID)
		if err != nil {
			s.logger.Warn().Err(err).Str("client", client.Name).Msg("Failed to get client")
			continue
		}

		// Get download directory
		downloadDir, err := dlClient.GetDownloadDir(ctx)
		if err != nil {
			s.logger.Warn().Err(err).Str("client", client.Name).Msg("Failed to get download dir")
			continue
		}

		// Scan for video files
		files, err := s.findVideoFiles(downloadDir)
		if err != nil {
			s.logger.Warn().Err(err).Str("path", downloadDir).Msg("Failed to scan for files")
			continue
		}

		for _, file := range files {
			// Skip files already being processed
			if s.IsProcessing(file) {
				continue
			}

			// Try to find a matching queue item
			// For now, queue without mapping - matching will happen during processing
			job := ImportJob{
				SourcePath: file,
				Manual:     false,
			}

			if err := s.QueueImport(job); err != nil {
				s.logger.Debug().Err(err).Str("file", file).Msg("Failed to queue file")
			}
		}
	}

	return nil
}

// processImport handles the actual import of a single file.
func (s *Service) processImport(ctx context.Context, job ImportJob) (*ImportResult, error) {
	result := &ImportResult{
		SourcePath: job.SourcePath,
	}

	// Load settings from database
	settings, err := s.GetSettings(ctx)
	if err != nil {
		s.logger.Warn().Err(err).Msg("Failed to load settings, using defaults")
		defaultSettings := DefaultImportSettings()
		settings = &defaultSettings
	}

	// Update renamer with current settings from database
	s.UpdateRenamerSettings(settings.ToRenamerSettings())

	// Step 1: Validate the file
	if err := s.validateFile(ctx, job.SourcePath, settings); err != nil {
		result.Error = err
		return result, err
	}

	// Step 2: Match to library
	var match *LibraryMatch
	if job.ConfirmedMatch != nil {
		match = job.ConfirmedMatch
	} else {
		match, err = s.matchToLibraryWithSettings(ctx, job.SourcePath, job.DownloadMapping, settings)
		if err != nil {
			// Handle unknown media based on settings
			if err == ErrNoMatch {
				switch settings.UnknownMediaBehavior {
				case UnknownAutoAdd:
					// TODO: Implement auto-add to library
					// This requires: parsing filename, searching metadata providers,
					// selecting root folder, and creating library entry
					s.logger.Warn().
						Str("path", job.SourcePath).
						Msg("Auto-add not yet implemented, file will be skipped")
					result.Error = err
					return result, err

				case UnknownIgnore:
					fallthrough
				default:
					s.logger.Debug().
						Str("path", job.SourcePath).
						Msg("No library match found, ignoring file per settings")
					result.Error = err
					return result, err
				}
			}
			result.Error = err
			return result, err
		}
	}

	// Ensure root folder path is properly set from the library item's root folder
	// Req 22.2.1-22.2.3: In multi-version mode, use slot's root folder if specified
	if err := s.populateRootFolder(ctx, match, job.TargetSlotID); err != nil {
		s.logger.Warn().Err(err).Msg("Failed to populate root folder, using match root folder")
	}
	result.Match = match

	// Step 3: Extract MediaInfo
	var mediaInfo *mediainfo.MediaInfo
	if s.mediainfo != nil && s.mediainfo.IsAvailable() {
		mediaInfo, err = s.mediainfo.Probe(ctx, job.SourcePath)
		if err != nil {
			s.logger.Warn().Err(err).Str("path", job.SourcePath).Msg("MediaInfo probe failed, using fallback")
		}
	}
	if mediaInfo == nil {
		mediaInfo = &mediainfo.MediaInfo{}
	}
	result.MediaInfo = mediaInfo

	// Step 4: Compute destination path
	destPath, err := s.computeDestination(ctx, match, mediaInfo, job.SourcePath, settings)
	if err != nil {
		result.Error = err
		return result, err
	}
	result.DestinationPath = destPath

	// Step 4.5: Slot evaluation for multi-version support (Req 5.1.1-5.2.3)
	var targetSlotID *int64
	if s.slots != nil && s.slots.IsMultiVersionEnabled(ctx) {
		slotResult, slotErr := s.evaluateSlotAssignment(ctx, job, match)
		if slotErr != nil {
			s.logger.Warn().Err(slotErr).Msg("Slot evaluation failed, continuing without slot assignment")
		} else if slotResult != nil {
			result.SlotAssignments = slotResult.Assignments
			result.RecommendedSlotID = slotResult.RecommendedSlotID

			// Req 5.2.1: If manual import and multiple slots match without override, require selection
			if slotResult.RequiresSelection && job.TargetSlotID == nil && job.Manual {
				result.RequiresSlotSelection = true
				return result, nil // Return early for slot selection
			}

			// Determine target slot
			if job.TargetSlotID != nil {
				// Req 5.2.3: User override
				targetSlotID = job.TargetSlotID
			} else if slotResult.RecommendedSlotID != nil {
				targetSlotID = slotResult.RecommendedSlotID
			}
		}
	}

	// Step 5: Check for existing file (upgrade scenario)
	if match.ExistingFile != "" {
		result.IsUpgrade = true
		result.PreviousFile = match.ExistingFile
	}

	// Step 6: Execute import (hardlink/copy)
	linkMode, err := s.executeImport(ctx, job.SourcePath, destPath)
	if err != nil {
		result.Error = err
		return result, err
	}
	result.LinkMode = linkMode

	// Step 7: Update library records
	fileID, updateErr := s.updateLibraryWithID(ctx, match, destPath, mediaInfo)
	if updateErr != nil {
		// Import succeeded but library update failed - log warning but don't fail
		s.logger.Warn().Err(updateErr).Msg("Failed to update library records")
	}

	// Step 7.5: Assign file to slot (if multi-version enabled)
	if targetSlotID != nil && fileID != nil && s.slots != nil {
		mediaID := s.getMediaIDFromMatch(match)
		if mediaID != nil {
			if err := s.slots.AssignFileToSlot(ctx, match.MediaType, *mediaID, *targetSlotID, *fileID); err != nil {
				s.logger.Warn().Err(err).Int64("slotId", *targetSlotID).Msg("Failed to assign file to slot")
			} else {
				result.AssignedSlotID = targetSlotID
				s.logger.Info().Int64("slotId", *targetSlotID).Int64("fileId", *fileID).Msg("File assigned to slot")
			}
		}
	}

	// Step 8: Handle upgrade cleanup
	if result.IsUpgrade && result.PreviousFile != "" {
		// Delete old physical file
		if err := s.organizer.DeleteUpgradedFile(result.PreviousFile, destPath); err != nil {
			s.logger.Warn().Err(err).Str("file", result.PreviousFile).Msg("Failed to delete upgraded file")
		}
		// Delete old database record
		if match.ExistingFileID != nil {
			if match.MediaType == "movie" {
				if err := s.movies.RemoveFile(ctx, *match.ExistingFileID); err != nil {
					s.logger.Warn().Err(err).Int64("fileId", *match.ExistingFileID).Msg("Failed to remove old movie file record")
				}
			} else if match.MediaType == "episode" {
				if err := s.tv.RemoveEpisodeFile(ctx, *match.ExistingFileID); err != nil {
					s.logger.Warn().Err(err).Int64("fileId", *match.ExistingFileID).Msg("Failed to remove old episode file record")
				}
			}
		}
	}

	// Step 9: Log to history
	if err := s.logImportHistory(ctx, result); err != nil {
		s.logger.Warn().Err(err).Msg("Failed to log import history")
	}

	// Step 10: Clear health status for this path
	if s.health != nil {
		s.health.ClearStatusStr("import", job.SourcePath)
	}

	result.Success = true
	return result, nil
}

// computeDestination computes the full destination path for the file.
func (s *Service) computeDestination(
	ctx context.Context,
	match *LibraryMatch,
	mediaInfo *mediainfo.MediaInfo,
	sourcePath string,
	settings *ImportSettings,
) (string, error) {
	ext := filepath.Ext(sourcePath)

	// Build token context from match and mediainfo
	tokenCtx := s.buildTokenContext(ctx, match, mediaInfo, sourcePath)

	var filename string
	var folderPath string
	var err error

	if match.MediaType == "movie" {
		// Movie: root folder / movie folder / filename
		filename, err = s.renamer.ResolveMovieFilename(tokenCtx, ext)
		if err != nil {
			return "", fmt.Errorf("failed to resolve movie filename: %w", err)
		}

		movieFolder, err := s.renamer.ResolveMovieFolderName(tokenCtx)
		if err != nil {
			return "", fmt.Errorf("failed to resolve movie folder: %w", err)
		}

		folderPath = filepath.Join(match.RootFolder, movieFolder)
	} else {
		// Episode: root folder / series folder / season folder / filename
		filename, err = s.renamer.ResolveEpisodeFilename(tokenCtx, ext)
		if err != nil {
			return "", fmt.Errorf("failed to resolve episode filename: %w", err)
		}

		seriesFolder, err := s.renamer.ResolveSeriesFolderName(tokenCtx)
		if err != nil {
			return "", fmt.Errorf("failed to resolve series folder: %w", err)
		}

		seasonFolder := s.renamer.ResolveSeasonFolderName(tokenCtx.SeasonNumber)
		folderPath = filepath.Join(match.RootFolder, seriesFolder, seasonFolder)
	}

	// Validate path length
	fullPath, err := s.renamer.ResolveFullPath(folderPath, "", filename)
	if err != nil {
		return "", ErrPathTooLong
	}

	return fullPath, nil
}

// executeImport performs the actual file import using hardlink/copy.
func (s *Service) executeImport(ctx context.Context, source, dest string) (organizer.LinkMode, error) {
	// Ensure destination directory exists
	destDir := filepath.Dir(dest)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Use organizer's ImportFile which handles hardlink/symlink/copy fallback
	return s.organizer.ImportFile(source, dest)
}

// updateLibrary updates the library database with the imported file.
func (s *Service) updateLibrary(ctx context.Context, match *LibraryMatch, destPath string, mediaInfo *mediainfo.MediaInfo) error {
	stat, err := os.Stat(destPath)
	if err != nil {
		return fmt.Errorf("failed to stat destination file: %w", err)
	}

	if match.MediaType == "movie" && match.MovieID != nil {
		// Add/update movie file
		_, err = s.movies.AddFile(ctx, *match.MovieID, movies.CreateMovieFileInput{
			Path:       destPath,
			Size:       stat.Size(),
			Quality:    "", // TODO: Extract from parsed info
			VideoCodec: mediaInfo.VideoCodec,
			AudioCodec: mediaInfo.AudioCodec,
			Resolution: mediaInfo.VideoResolution,
		})
		return err
	} else if match.MediaType == "episode" && match.EpisodeID != nil {
		// Add/update episode file
		_, err = s.tv.AddEpisodeFile(ctx, *match.EpisodeID, tv.CreateEpisodeFileInput{
			Path:       destPath,
			Size:       stat.Size(),
			Quality:    "",
			VideoCodec: mediaInfo.VideoCodec,
			AudioCodec: mediaInfo.AudioCodec,
			Resolution: mediaInfo.VideoResolution,
		})
		return err
	}

	return nil
}

// buildTokenContext creates a token context from match and mediainfo.
func (s *Service) buildTokenContext(
	ctx context.Context,
	match *LibraryMatch,
	mediaInfo *mediainfo.MediaInfo,
	sourcePath string,
) *renamer.TokenContext {
	filename := filepath.Base(sourcePath)

	// Parse filename for quality/source/codec info
	parsed := scanner.ParseFilename(filename)

	tc := &renamer.TokenContext{
		OriginalFile:  filename,
		OriginalTitle: strings.TrimSuffix(filename, filepath.Ext(filename)),
	}

	// Use parsed filename data for quality info
	tc.Quality = parsed.Quality
	tc.Source = parsed.Source
	tc.Codec = parsed.Codec

	// Use parsed filename for video dynamic range (HDR info from Attributes)
	for _, attr := range parsed.Attributes {
		switch attr {
		case "HDR10+", "HDR10", "HDR", "DV", "Dolby Vision":
			if tc.VideoDynamicRange == "" {
				tc.VideoDynamicRange = attr
			} else {
				tc.VideoDynamicRange += " " + attr
			}
		}
	}

	// Use parsed filename for audio info
	if len(parsed.AudioCodecs) > 0 {
		tc.AudioCodec = strings.Join(parsed.AudioCodecs, " ")
	}
	if len(parsed.AudioChannels) > 0 {
		tc.AudioChannels = strings.Join(parsed.AudioChannels, " ")
	}
	// Append audio enhancements to audio codec
	if len(parsed.AudioEnhancements) > 0 {
		if tc.AudioCodec != "" {
			tc.AudioCodec += " " + strings.Join(parsed.AudioEnhancements, " ")
		} else {
			tc.AudioCodec = strings.Join(parsed.AudioEnhancements, " ")
		}
	}

	// Use parsed filename for release group, revision, and edition
	tc.ReleaseGroup = parsed.ReleaseGroup
	tc.Revision = parsed.Revision
	tc.EditionTags = parsed.Edition

	// Override with MediaInfo data if available (more accurate than filename parsing)
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

	// Fill in series/movie info from library
	if match.MediaType == "movie" && match.MovieID != nil {
		movie, err := s.movies.Get(ctx, *match.MovieID)
		if err == nil {
			tc.MovieTitle = movie.Title
			tc.MovieYear = movie.Year
		}
	} else if match.MediaType == "episode" && match.SeriesID != nil {
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
			episode, err := s.tv.GetEpisode(ctx, *match.EpisodeID)
			if err == nil {
				tc.EpisodeNumber = episode.EpisodeNumber
				tc.EpisodeTitle = episode.Title
				if episode.AirDate != nil {
					tc.AirDate = *episode.AirDate
				}
			}
		}

		// Handle multi-episode
		if len(match.EpisodeIDs) > 1 {
			tc.EpisodeNumbers = make([]int, 0, len(match.EpisodeIDs))
			for _, epID := range match.EpisodeIDs {
				ep, err := s.tv.GetEpisode(ctx, epID)
				if err == nil {
					tc.EpisodeNumbers = append(tc.EpisodeNumbers, ep.EpisodeNumber)
				}
			}
		}
	}

	return tc
}

// logImportHistory logs the import to history.
func (s *Service) logImportHistory(ctx context.Context, result *ImportResult) error {
	if s.history == nil {
		return nil
	}

	var mediaID int64
	if result.Match.MediaType == "movie" && result.Match.MovieID != nil {
		mediaID = *result.Match.MovieID
	} else if result.Match.EpisodeID != nil {
		mediaID = *result.Match.EpisodeID
	}

	eventType := "imported"
	if result.IsUpgrade {
		eventType = "import_upgrade"
	}

	return s.history.Create(ctx, HistoryInput{
		EventType: eventType,
		MediaType: result.Match.MediaType,
		MediaID:   mediaID,
		Quality:   "", // TODO: Extract quality
		Data: map[string]any{
			"sourcePath":       result.SourcePath,
			"destinationPath":  result.DestinationPath,
			"originalFilename": filepath.Base(result.SourcePath),
			"finalFilename":    filepath.Base(result.DestinationPath),
			"linkMode":         string(result.LinkMode),
			"isUpgrade":        result.IsUpgrade,
			"previousFile":     result.PreviousFile,
		},
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

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
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

// handleCleanup handles source file cleanup after import.
func (s *Service) handleCleanup(ctx context.Context, sourcePath string, clientID int64) error {
	client, err := s.downloader.Get(ctx, clientID)
	if err != nil {
		return err
	}

	switch client.CleanupMode {
	case "leave":
		// Do nothing
		return nil

	case "delete_after_import":
		// Delete immediately
		return os.Remove(sourcePath)

	case "delete_after_seed_ratio":
		// Check seed ratio (handled by scheduler)
		// For now, leave the file
		return nil

	default:
		return nil
	}
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
		return nil, nil
	}

	// Parse the filename to get release attributes
	filename := filepath.Base(job.SourcePath)
	parsed := scanner.ParseFilename(filename)

	// Get media ID for slot evaluation
	var mediaType string
	var mediaID int64
	if match.MediaType == "movie" && match.MovieID != nil {
		mediaType = "movie"
		mediaID = *match.MovieID
	} else if match.MediaType == "episode" && match.EpisodeID != nil {
		mediaType = "episode"
		mediaID = *match.EpisodeID
	} else {
		return nil, nil
	}

	// Evaluate release against all slots
	eval, err := s.slots.EvaluateRelease(ctx, parsed, mediaType, mediaID)
	if err != nil {
		return nil, err
	}

	if eval == nil || len(eval.Assignments) == 0 {
		return nil, nil
	}

	// Convert to import-level type
	result := &slotEvaluationResult{
		Assignments:       make([]SlotAssignment, 0, len(eval.Assignments)),
		RequiresSelection: eval.RequiresSelection,
	}

	for _, a := range eval.Assignments {
		result.Assignments = append(result.Assignments, SlotAssignment{
			SlotID:     a.SlotID,
			SlotNumber: a.SlotNumber,
			SlotName:   a.SlotName,
			MatchScore: a.MatchScore,
			IsUpgrade:  a.IsUpgrade,
			IsNewFill:  a.IsNewFill,
		})
	}

	if eval.RecommendedSlotID != 0 {
		result.RecommendedSlotID = &eval.RecommendedSlotID
	}

	return result, nil
}

// updateLibraryWithID updates the library and returns the created file ID.
func (s *Service) updateLibraryWithID(ctx context.Context, match *LibraryMatch, destPath string, mediaInfo *mediainfo.MediaInfo) (*int64, error) {
	stat, err := os.Stat(destPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat destination file: %w", err)
	}

	if match.MediaType == "movie" && match.MovieID != nil {
		file, err := s.movies.AddFile(ctx, *match.MovieID, movies.CreateMovieFileInput{
			Path:       destPath,
			Size:       stat.Size(),
			Quality:    "",
			VideoCodec: mediaInfo.VideoCodec,
			AudioCodec: mediaInfo.AudioCodec,
			Resolution: mediaInfo.VideoResolution,
		})
		if err != nil {
			return nil, err
		}
		return &file.ID, nil
	} else if match.MediaType == "episode" && match.EpisodeID != nil {
		file, err := s.tv.AddEpisodeFile(ctx, *match.EpisodeID, tv.CreateEpisodeFileInput{
			Path:       destPath,
			Size:       stat.Size(),
			Quality:    "",
			VideoCodec: mediaInfo.VideoCodec,
			AudioCodec: mediaInfo.AudioCodec,
			Resolution: mediaInfo.VideoResolution,
		})
		if err != nil {
			return nil, err
		}
		return &file.ID, nil
	}

	return nil, nil
}

// getMediaIDFromMatch extracts the media ID from a library match.
func (s *Service) getMediaIDFromMatch(match *LibraryMatch) *int64 {
	if match.MediaType == "movie" && match.MovieID != nil {
		return match.MovieID
	} else if match.MediaType == "episode" && match.EpisodeID != nil {
		return match.EpisodeID
	}
	return nil
}
