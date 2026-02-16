package librarymanager

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/library/movies"
	"github.com/slipstream/slipstream/internal/library/rootfolder"
	"github.com/slipstream/slipstream/internal/library/scanner"
	"github.com/slipstream/slipstream/internal/library/tv"
	"github.com/slipstream/slipstream/internal/metadata"
	"github.com/slipstream/slipstream/internal/progress"
)

// ScanRootFolder scans a root folder for media files and matches them to metadata.
func (s *Service) ScanRootFolder(ctx context.Context, rootFolderID int64) (*ScanResult, error) {
	folder, err := s.rootfolders.Get(ctx, rootFolderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get root folder: %w", err)
	}

	if s.isScanActive(rootFolderID) {
		return nil, ErrScanInProgress
	}

	activityID := fmt.Sprintf("scan-%d-%d", rootFolderID, time.Now().UnixNano())
	s.setScanActive(rootFolderID, activityID)
	defer s.clearScanActive(rootFolderID)

	activity := s.initScanActivity(activityID, folder)

	s.logger.Info().
		Int64("rootFolderId", rootFolderID).
		Str("path", folder.Path).
		Str("mediaType", folder.MediaType).
		Msg("Starting library scan")

	defaultProfile, err := s.getDefaultQualityProfile(ctx)
	if err != nil {
		s.failActivity(activity, err.Error())
		return nil, err
	}

	scanResult, err := s.performScan(ctx, folder, activity)
	if err != nil {
		s.failActivity(activity, fmt.Sprintf("Scan failed: %s", err.Error()))
		return nil, fmt.Errorf("scan failed: %w", err)
	}

	result, pending := s.initScanResult(rootFolderID, scanResult)
	s.processMediaByType(ctx, folder, scanResult, defaultProfile.ID, result, activity, pending)
	s.matchUnmatchedByType(ctx, folder, result, activity, pending)
	s.VerifyFileExistence(ctx, rootFolderID, folder.Path)
	s.downloadArtworkIfNeeded(ctx, pending, result, activity)
	s.appendScanErrors(result, scanResult)
	s.completeScanActivity(activity, result)
	s.logScanCompletion(rootFolderID, result)

	return result, nil
}

func (s *Service) initScanActivity(activityID string, folder *rootfolder.RootFolder) *progress.ActivityBuilder {
	if s.progress == nil {
		return nil
	}
	title := fmt.Sprintf("Scanning %s", folder.Name)
	activity := s.progress.NewActivityBuilder(activityID, progress.ActivityTypeScan, title)
	activity.SetMetadata("rootFolderId", folder.ID)
	activity.SetMetadata("mediaType", folder.MediaType)
	activity.SetMetadata("path", folder.Path)
	return activity
}

func (s *Service) failActivity(activity *progress.ActivityBuilder, msg string) {
	if activity != nil {
		activity.Fail(msg)
	}
}

func (s *Service) performScan(ctx context.Context, folder *rootfolder.RootFolder, activity *progress.ActivityBuilder) (*scanner.ScanResult, error) {
	return s.scanner.ScanFolder(ctx, folder.Path, folder.MediaType, func(scanProgress scanner.ScanProgress) {
		if activity != nil {
			subtitle := fmt.Sprintf("Scanning: %s", filepath.Base(scanProgress.CurrentPath))
			activity.Update(subtitle, -1)
			activity.SetMetadata("filesScanned", scanProgress.FilesScanned)
			activity.SetMetadata("moviesFound", scanProgress.MoviesFound)
			activity.SetMetadata("episodesFound", scanProgress.EpisodesFound)
		}
	})
}

func (s *Service) initScanResult(rootFolderID int64, scanResult *scanner.ScanResult) (*ScanResult, *pendingArtwork) {
	result := &ScanResult{
		RootFolderID: rootFolderID,
		TotalFiles:   scanResult.TotalFiles,
		Errors:       make([]string, 0),
	}
	pending := &pendingArtwork{
		movieMeta:  make([]*metadata.MovieResult, 0),
		seriesMeta: make([]*metadata.SeriesResult, 0),
	}
	return result, pending
}

func (s *Service) processMediaByType(ctx context.Context, folder *rootfolder.RootFolder, scanResult *scanner.ScanResult, qualityProfileID int64, result *ScanResult, activity *progress.ActivityBuilder, pending *pendingArtwork) {
	if folder.MediaType == mediaTypeMovie {
		s.processMovies(ctx, folder, scanResult.Movies, qualityProfileID, result, activity, pending)
	} else {
		s.processEpisodes(ctx, folder, scanResult.Episodes, qualityProfileID, result, activity, pending)
	}
}

func (s *Service) matchUnmatchedByType(ctx context.Context, folder *rootfolder.RootFolder, result *ScanResult, activity *progress.ActivityBuilder, pending *pendingArtwork) {
	if folder.MediaType == mediaTypeMovie {
		s.matchUnmatchedMovies(ctx, folder, result, activity, pending)
	} else {
		s.matchUnmatchedSeries(ctx, folder, result, activity, pending)
	}
}

func (s *Service) downloadArtworkIfNeeded(ctx context.Context, pending *pendingArtwork, result *ScanResult, activity *progress.ActivityBuilder) {
	if s.artwork != nil && (len(pending.movieMeta) > 0 || len(pending.seriesMeta) > 0) {
		s.downloadPendingArtwork(ctx, pending, result, activity)
	}
}

func (s *Service) appendScanErrors(result *ScanResult, scanResult *scanner.ScanResult) {
	for _, scanErr := range scanResult.Errors {
		result.Errors = append(result.Errors, fmt.Sprintf("%s: %s", scanErr.Path, scanErr.Error))
	}
}

func (s *Service) completeScanActivity(activity *progress.ActivityBuilder, result *ScanResult) {
	if activity != nil {
		summary := s.buildScanSummary(result)
		activity.Complete(summary)
	}
}

func (s *Service) logScanCompletion(rootFolderID int64, result *ScanResult) {
	s.logger.Info().
		Int64("rootFolderId", rootFolderID).
		Int("totalFiles", result.TotalFiles).
		Int("moviesAdded", result.MoviesAdded).
		Int("seriesAdded", result.SeriesAdded).
		Int("filesLinked", result.FilesLinked).
		Int("metadataMatched", result.MetadataMatched).
		Int("artworksFetched", result.ArtworksFetched).
		Int("errors", len(result.Errors)).
		Msg("Library scan completed")
}

// processMovies processes scanned movie files.
func (s *Service) processMovies(
	ctx context.Context,
	folder *rootfolder.RootFolder,
	parsedMovies []scanner.ParsedMedia,
	qualityProfileID int64,
	result *ScanResult,
	activity *progress.ActivityBuilder,
	pending *pendingArtwork,
) {
	total := len(parsedMovies)
	for i := range parsedMovies {
		parsed := &parsedMovies[i]
		s.updateMovieProgress(activity, i, total, parsed)

		if s.isMovieFileAlreadyTracked(ctx, parsed, result) {
			continue
		}

		movie, created, _ := s.processMovieMatch(ctx, folder, parsed, qualityProfileID, result, pending)
		s.linkMovieFile(ctx, movie, parsed, result)

		if created {
			result.MoviesAdded++
		}
	}
}

func (s *Service) updateMovieProgress(activity *progress.ActivityBuilder, index, total int, parsed *scanner.ParsedMedia) {
	if activity == nil {
		return
	}
	pct := 0
	if total > 0 {
		pct = (index + 1) * 100 / total
	}
	subtitle := fmt.Sprintf("Processing: %s", filepath.Base(parsed.FilePath))
	activity.Update(subtitle, pct)
}

func (s *Service) isMovieFileAlreadyTracked(ctx context.Context, parsed *scanner.ParsedMedia, result *ScanResult) bool {
	existingFile, err := s.movies.GetFileByPath(ctx, parsed.FilePath)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		result.Errors = append(result.Errors, fmt.Sprintf("failed to check file %s: %v", parsed.FilePath, err))
		return true
	}
	if existingFile != nil {
		s.logger.Debug().Str("path", parsed.FilePath).Msg("Movie file already tracked, skipping")
		return true
	}
	return false
}

func (s *Service) processMovieMatch(ctx context.Context, folder *rootfolder.RootFolder, parsed *scanner.ParsedMedia, qualityProfileID int64, result *ScanResult, pending *pendingArtwork) (*movies.Movie, bool, *metadata.MovieResult) {
	movie, created, meta, err := s.matchOrCreateMovie(ctx, folder, parsed, qualityProfileID)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("failed to process %s: %v", parsed.FilePath, err))
		return nil, false, nil
	}

	if created && meta != nil && (meta.PosterURL != "" || meta.BackdropURL != "") {
		pending.movieMeta = append(pending.movieMeta, meta)
	}

	return movie, created, meta
}

func (s *Service) linkMovieFile(ctx context.Context, movie *movies.Movie, parsed *scanner.ParsedMedia, result *ScanResult) {
	if movie == nil {
		return
	}
	if err := s.addMovieFile(ctx, movie.ID, parsed); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("failed to add file %s: %v", parsed.FilePath, err))
	} else {
		result.FilesLinked++
	}
}

// processEpisodes processes scanned TV episode files.
func (s *Service) processEpisodes(
	ctx context.Context,
	folder *rootfolder.RootFolder,
	parsedEpisodes []scanner.ParsedMedia,
	qualityProfileID int64,
	result *ScanResult,
	activity *progress.ActivityBuilder,
	pending *pendingArtwork,
) {
	seriesMap := s.groupEpisodesByTitle(parsedEpisodes)
	total := len(parsedEpisodes)
	processedFiles := 0

	for _, episodes := range seriesMap {
		if len(episodes) == 0 {
			continue
		}

		newEpisodes := s.filterUntrackedEpisodes(ctx, episodes, total, &processedFiles, result, activity)
		if len(newEpisodes) == 0 {
			continue
		}

		s.processSeriesEpisodes(ctx, folder, newEpisodes, qualityProfileID, result, pending)
	}
}

func (s *Service) groupEpisodesByTitle(parsedEpisodes []scanner.ParsedMedia) map[string][]scanner.ParsedMedia {
	seriesMap := make(map[string][]scanner.ParsedMedia)
	for i := range parsedEpisodes {
		parsed := &parsedEpisodes[i]
		key := strings.ToLower(parsed.Title)
		seriesMap[key] = append(seriesMap[key], *parsed)
	}
	return seriesMap
}

func (s *Service) filterUntrackedEpisodes(ctx context.Context, episodes []scanner.ParsedMedia, total int, processedFiles *int, result *ScanResult, activity *progress.ActivityBuilder) []scanner.ParsedMedia {
	var newEpisodes []scanner.ParsedMedia
	for i := range episodes {
		parsed := &episodes[i]
		*processedFiles++

		s.updateEpisodeProgress(activity, *processedFiles, total, parsed)

		if s.isEpisodeFileAlreadyTracked(ctx, parsed, result) {
			continue
		}
		newEpisodes = append(newEpisodes, *parsed)
	}
	return newEpisodes
}

func (s *Service) updateEpisodeProgress(activity *progress.ActivityBuilder, processedFiles, total int, parsed *scanner.ParsedMedia) {
	if activity == nil {
		return
	}
	pct := 0
	if total > 0 {
		pct = processedFiles * 100 / total
	}
	subtitle := fmt.Sprintf("Checking: %s", filepath.Base(parsed.FilePath))
	activity.Update(subtitle, pct)
}

func (s *Service) isEpisodeFileAlreadyTracked(ctx context.Context, parsed *scanner.ParsedMedia, result *ScanResult) bool {
	existingFile, err := s.tv.GetEpisodeFileByPath(ctx, parsed.FilePath)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		result.Errors = append(result.Errors, fmt.Sprintf("failed to check file %s: %v", parsed.FilePath, err))
		return true
	}
	if existingFile != nil {
		s.logger.Debug().Str("path", parsed.FilePath).Msg("Episode file already tracked, skipping")
		return true
	}
	return false
}

func (s *Service) processSeriesEpisodes(ctx context.Context, folder *rootfolder.RootFolder, newEpisodes []scanner.ParsedMedia, qualityProfileID int64, result *ScanResult, pending *pendingArtwork) {
	firstEp := newEpisodes[0]

	series, created, meta, err := s.matchOrCreateSeries(ctx, folder, &firstEp, qualityProfileID)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("failed to process series %s: %v", firstEp.Title, err))
		return
	}

	if created {
		result.SeriesAdded++
		if meta != nil && (meta.PosterURL != "" || meta.BackdropURL != "") {
			pending.seriesMeta = append(pending.seriesMeta, meta)
		}
	}

	s.linkEpisodeFiles(ctx, series, newEpisodes, result)
}

func (s *Service) linkEpisodeFiles(ctx context.Context, series *tv.Series, newEpisodes []scanner.ParsedMedia, result *ScanResult) {
	if series == nil {
		return
	}
	for i := range newEpisodes {
		parsed := &newEpisodes[i]
		if err := s.addEpisodeFile(ctx, series.ID, parsed); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("failed to add episode file %s: %v", parsed.FilePath, err))
		} else {
			result.FilesLinked++
		}
	}
}

// ScanSingleFile scans and processes a single file.
func (s *Service) ScanSingleFile(ctx context.Context, filePath string) error {
	folder, err := s.findRootFolderForPath(ctx, filePath)
	if err != nil {
		return err
	}

	parsed, err := s.scanner.ScanFile(filePath)
	if err != nil {
		return err
	}

	if folder.MediaType == mediaTypeMovie {
		return s.scanSingleMovieFile(ctx, folder, parsed)
	}
	return s.scanSingleEpisodeFile(ctx, folder, parsed)
}

func (s *Service) scanSingleMovieFile(ctx context.Context, folder *rootfolder.RootFolder, parsed *scanner.ParsedMedia) error {
	if s.isMovieFileTracked(ctx, parsed.FilePath) {
		return nil
	}

	defaultProfile, err := s.getDefaultQualityProfile(ctx)
	if err != nil {
		return err
	}

	movie, created, meta, err := s.matchOrCreateMovie(ctx, folder, parsed, defaultProfile.ID)
	if err != nil {
		return err
	}
	if movie == nil {
		return nil
	}

	if err := s.addMovieFile(ctx, movie.ID, parsed); err != nil {
		return err
	}

	s.downloadNewMovieArtwork(created, meta)
	return nil
}

func (s *Service) scanSingleEpisodeFile(ctx context.Context, folder *rootfolder.RootFolder, parsed *scanner.ParsedMedia) error {
	if s.isEpisodeFileTracked(ctx, parsed.FilePath) {
		return nil
	}

	defaultProfile, err := s.getDefaultQualityProfile(ctx)
	if err != nil {
		return err
	}

	series, created, meta, err := s.matchOrCreateSeries(ctx, folder, parsed, defaultProfile.ID)
	if err != nil {
		return err
	}
	if series == nil {
		return nil
	}

	if err := s.addEpisodeFile(ctx, series.ID, parsed); err != nil {
		return err
	}

	s.downloadNewSeriesArtwork(created, meta)
	return nil
}

// findRootFolderForPath finds which root folder a file path belongs to.
func (s *Service) findRootFolderForPath(ctx context.Context, filePath string) (*rootfolder.RootFolder, error) {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, err
	}

	folders, err := s.rootfolders.List(ctx)
	if err != nil {
		return nil, err
	}

	for _, folder := range folders {
		if strings.HasPrefix(absPath, folder.Path) {
			return folder, nil
		}
	}

	return nil, fmt.Errorf("file is not within any root folder: %s", filePath)
}

// scanMovieFolder scans a movie's folder for new files and verifies existing files.
func (s *Service) scanMovieFolder(ctx context.Context, movie *movies.Movie) {
	if movie.Path == "" || strings.HasPrefix(movie.Path, "/mock/") {
		return
	}

	scanResult, err := s.scanner.ScanFolder(ctx, movie.Path, "movie", nil)
	if err != nil {
		s.logger.Warn().Err(err).Str("path", movie.Path).Int64("movieId", movie.ID).Msg("Failed to scan movie folder")
		return
	}

	s.linkNewMovieFiles(ctx, movie.ID, scanResult.Movies)
	s.verifyMovieFilesExist(ctx, movie.ID)
}

func (s *Service) linkNewMovieFiles(ctx context.Context, movieID int64, parsedMovies []scanner.ParsedMedia) {
	for i := range parsedMovies {
		parsed := &parsedMovies[i]
		if s.isMovieFileTracked(ctx, parsed.FilePath) {
			continue
		}
		if err := s.addMovieFile(ctx, movieID, parsed); err != nil {
			s.logger.Warn().Err(err).Str("path", parsed.FilePath).Msg("Failed to add movie file during refresh")
		}
	}
}

func (s *Service) verifyMovieFilesExist(ctx context.Context, movieID int64) {
	files, err := s.movies.GetFiles(ctx, movieID)
	if err != nil {
		s.logger.Warn().Err(err).Int64("movieId", movieID).Msg("Failed to list movie files for verification")
		return
	}
	for i := range files {
		f := &files[i]
		if _, err := os.Stat(f.Path); os.IsNotExist(err) {
			s.logger.Warn().Str("path", f.Path).Int64("movieId", movieID).Msg("Movie file disappeared during refresh")
			_ = s.queries.UpdateMovieStatus(ctx, sqlc.UpdateMovieStatusParams{
				Status: "missing",
				ID:     movieID,
			})
		}
	}
}

// scanSeriesFolder scans a series folder for new episode files and verifies existing files.
func (s *Service) scanSeriesFolder(ctx context.Context, series *tv.Series) {
	if series.Path == "" || strings.HasPrefix(series.Path, "/mock/") {
		return
	}

	scanResult, err := s.scanner.ScanFolder(ctx, series.Path, "tv", nil)
	if err != nil {
		s.logger.Warn().Err(err).Str("path", series.Path).Int64("seriesId", series.ID).Msg("Failed to scan series folder")
		return
	}

	s.linkNewEpisodeFiles(ctx, series.ID, scanResult.Episodes)
	s.verifyEpisodeFilesExist(ctx, series.ID)
}

func (s *Service) linkNewEpisodeFiles(ctx context.Context, seriesID int64, parsedEpisodes []scanner.ParsedMedia) {
	for i := range parsedEpisodes {
		parsed := &parsedEpisodes[i]
		if s.isEpisodeFileTracked(ctx, parsed.FilePath) {
			continue
		}
		if err := s.addEpisodeFile(ctx, seriesID, parsed); err != nil {
			s.logger.Warn().Err(err).Str("path", parsed.FilePath).Msg("Failed to add episode file during refresh")
		}
	}
}

func (s *Service) verifyEpisodeFilesExist(ctx context.Context, seriesID int64) {
	episodeFiles, err := s.queries.ListEpisodeFilesBySeries(ctx, seriesID)
	if err != nil {
		s.logger.Warn().Err(err).Int64("seriesId", seriesID).Msg("Failed to list episode files for verification")
		return
	}
	for _, ef := range episodeFiles {
		if _, err := os.Stat(ef.Path); os.IsNotExist(err) {
			s.logger.Warn().Str("path", ef.Path).Int64("episodeId", ef.EpisodeID).Msg("Episode file disappeared during refresh")
			_ = s.queries.UpdateEpisodeStatus(ctx, sqlc.UpdateEpisodeStatusParams{
				Status: "missing",
				ID:     ef.EpisodeID,
			})
		}
	}
}

func (s *Service) isMovieFileTracked(ctx context.Context, filePath string) bool {
	existingFile, err := s.movies.GetFileByPath(ctx, filePath)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		s.logger.Warn().Err(err).Str("path", filePath).Msg("Failed to check for existing movie file")
	}
	if existingFile != nil {
		s.logger.Debug().Str("path", filePath).Msg("Movie file already tracked, skipping")
		return true
	}
	return false
}

func (s *Service) isEpisodeFileTracked(ctx context.Context, filePath string) bool {
	existingFile, err := s.tv.GetEpisodeFileByPath(ctx, filePath)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		s.logger.Warn().Err(err).Str("path", filePath).Msg("Failed to check for existing episode file")
	}
	if existingFile != nil {
		s.logger.Debug().Str("path", filePath).Msg("Episode file already tracked, skipping")
		return true
	}
	return false
}

func (s *Service) downloadNewMovieArtwork(created bool, meta *metadata.MovieResult) {
	if !created || meta == nil || s.artwork == nil {
		return
	}
	go func() {
		if err := s.artwork.DownloadMovieArtwork(context.Background(), meta); err != nil {
			s.logger.Warn().Err(err).Int("tmdbId", meta.ID).Msg("Failed to download movie artwork")
		}
	}()
}

func (s *Service) downloadNewSeriesArtwork(created bool, meta *metadata.SeriesResult) {
	if !created || meta == nil || s.artwork == nil {
		return
	}
	go func() {
		if err := s.artwork.DownloadSeriesArtwork(context.Background(), meta); err != nil {
			s.logger.Warn().Err(err).Int("tvdbId", meta.TvdbID).Msg("Failed to download series artwork")
		}
	}()
}

// VerifyFileExistence checks that tracked files for a root folder still exist on disk.
// Files that have disappeared get their media status set to "missing" and a health alert is registered.
func (s *Service) VerifyFileExistence(ctx context.Context, rootFolderID int64, folderPath string) int {
	if strings.HasPrefix(folderPath, "/mock/") {
		return 0
	}

	rfID := sql.NullInt64{Int64: rootFolderID, Valid: true}
	missing := 0

	missing += s.verifyMovieFiles(ctx, rfID, rootFolderID)
	missing += s.verifyEpisodeFiles(ctx, rfID, rootFolderID)

	if missing > 0 {
		s.logger.Warn().Int("missing", missing).Int64("rootFolderId", rootFolderID).Msg("Detected disappeared files during scan")
	}
	return missing
}

func (s *Service) verifyMovieFiles(ctx context.Context, rfID sql.NullInt64, rootFolderID int64) int {
	movieFiles, err := s.queries.ListMovieFilesForRootFolder(ctx, rfID)
	if err != nil {
		s.logger.Warn().Err(err).Int64("rootFolderId", rootFolderID).Msg("Failed to list movie files for verification")
		return 0
	}

	missing := 0
	for _, mf := range movieFiles {
		if _, err := os.Stat(mf.Path); os.IsNotExist(err) {
			missing++
			s.logger.Warn().Str("path", mf.Path).Int64("movieId", mf.MovieID).Msg("Movie file disappeared from disk")
			_ = s.queries.UpdateMovieStatus(ctx, sqlc.UpdateMovieStatusParams{
				Status: "missing",
				ID:     mf.MovieID,
			})
			if s.healthSvc != nil {
				healthID := fmt.Sprintf("missing-file-movie-%d", mf.FileID)
				s.healthSvc.SetWarningStr("storage", healthID, fmt.Sprintf("Movie file not found: %s", mf.Path))
			}
		}
	}
	return missing
}

func (s *Service) verifyEpisodeFiles(ctx context.Context, rfID sql.NullInt64, rootFolderID int64) int {
	episodeFiles, err := s.queries.ListEpisodeFilesForRootFolder(ctx, rfID)
	if err != nil {
		s.logger.Warn().Err(err).Int64("rootFolderId", rootFolderID).Msg("Failed to list episode files for verification")
		return 0
	}

	missing := 0
	for _, ef := range episodeFiles {
		if _, err := os.Stat(ef.Path); os.IsNotExist(err) {
			missing++
			s.logger.Warn().Str("path", ef.Path).Int64("episodeId", ef.EpisodeID).Msg("Episode file disappeared from disk")
			_ = s.queries.UpdateEpisodeStatus(ctx, sqlc.UpdateEpisodeStatusParams{
				Status: "missing",
				ID:     ef.EpisodeID,
			})
			if s.healthSvc != nil {
				healthID := fmt.Sprintf("missing-file-episode-%d", ef.FileID)
				s.healthSvc.SetWarningStr("storage", healthID, fmt.Sprintf("Episode file not found: %s", ef.Path))
			}
		}
	}
	return missing
}
