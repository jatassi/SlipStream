package librarymanager

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/library/movies"
	"github.com/slipstream/slipstream/internal/library/quality"
	"github.com/slipstream/slipstream/internal/library/rootfolder"
	"github.com/slipstream/slipstream/internal/library/scanner"
	"github.com/slipstream/slipstream/internal/library/tv"
	"github.com/slipstream/slipstream/internal/metadata"
	"github.com/slipstream/slipstream/internal/progress"
)

var (
	ErrNoMetadataProvider = errors.New("no metadata provider configured")
	ErrNoQualityProfile   = errors.New("no quality profile available")
	ErrScanInProgress     = errors.New("scan already in progress for this folder")
)

// ScanResult represents the final result of a scan operation.
type ScanResult struct {
	RootFolderID int64    `json:"rootFolderId"`
	TotalFiles   int      `json:"totalFiles"`
	MoviesAdded  int      `json:"moviesAdded"`
	SeriesAdded  int      `json:"seriesAdded"`
	FilesLinked  int      `json:"filesLinked"`
	Errors       []string `json:"errors,omitempty"`
}

// Service orchestrates library scanning, file matching, and metadata lookup.
type Service struct {
	db              *sql.DB
	queries         *sqlc.Queries
	scanner         *scanner.Service
	movies          *movies.Service
	tv              *tv.Service
	metadata        *metadata.Service
	rootfolders     *rootfolder.Service
	qualityProfiles *quality.Service
	progress        *progress.Manager
	logger          zerolog.Logger

	// Track active scans by root folder ID
	activeScans map[int64]string // maps folderID -> activityID
	scanMu      sync.RWMutex
}

// NewService creates a new library manager service.
func NewService(
	db *sql.DB,
	scannerSvc *scanner.Service,
	moviesSvc *movies.Service,
	tvSvc *tv.Service,
	metadataSvc *metadata.Service,
	rootfolderSvc *rootfolder.Service,
	qualityProfileSvc *quality.Service,
	progressMgr *progress.Manager,
	logger zerolog.Logger,
) *Service {
	return &Service{
		db:              db,
		queries:         sqlc.New(db),
		scanner:         scannerSvc,
		movies:          moviesSvc,
		tv:              tvSvc,
		metadata:        metadataSvc,
		rootfolders:     rootfolderSvc,
		qualityProfiles: qualityProfileSvc,
		progress:        progressMgr,
		logger:          logger.With().Str("component", "librarymanager").Logger(),
		activeScans:     make(map[int64]string),
	}
}

// ScanRootFolder scans a root folder for media files and matches them to metadata.
func (s *Service) ScanRootFolder(ctx context.Context, rootFolderID int64) (*ScanResult, error) {
	// Get root folder
	folder, err := s.rootfolders.Get(ctx, rootFolderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get root folder: %w", err)
	}

	// Check if scan is already in progress
	if s.isScanActive(rootFolderID) {
		return nil, ErrScanInProgress
	}

	// Create activity for progress tracking
	activityID := fmt.Sprintf("scan-%d-%d", rootFolderID, time.Now().UnixNano())
	s.setScanActive(rootFolderID, activityID)
	defer s.clearScanActive(rootFolderID)

	var activity *progress.ActivityBuilder
	if s.progress != nil {
		title := fmt.Sprintf("Scanning %s", folder.Name)
		activity = s.progress.NewActivityBuilder(activityID, progress.ActivityTypeScan, title)
		activity.SetMetadata("rootFolderId", rootFolderID)
		activity.SetMetadata("mediaType", folder.MediaType)
		activity.SetMetadata("path", folder.Path)
	}

	s.logger.Info().
		Int64("rootFolderId", rootFolderID).
		Str("path", folder.Path).
		Str("mediaType", folder.MediaType).
		Msg("Starting library scan")

	// Get default quality profile
	defaultProfile, err := s.getDefaultQualityProfile(ctx)
	if err != nil {
		if activity != nil {
			activity.Fail(err.Error())
		}
		return nil, err
	}

	// Scan the folder
	scanResult, err := s.scanner.ScanFolder(ctx, folder.Path, folder.MediaType, func(scanProgress scanner.ScanProgress) {
		if activity != nil {
			subtitle := fmt.Sprintf("Scanning: %s", filepath.Base(scanProgress.CurrentPath))
			activity.Update(subtitle, -1) // Indeterminate during scan phase
			activity.SetMetadata("filesScanned", scanProgress.FilesScanned)
			activity.SetMetadata("moviesFound", scanProgress.MoviesFound)
			activity.SetMetadata("episodesFound", scanProgress.EpisodesFound)
		}
	})
	if err != nil {
		if activity != nil {
			activity.Fail(fmt.Sprintf("Scan failed: %s", err.Error()))
		}
		return nil, fmt.Errorf("scan failed: %w", err)
	}

	result := &ScanResult{
		RootFolderID: rootFolderID,
		TotalFiles:   scanResult.TotalFiles,
		Errors:       make([]string, 0),
	}

	// Process results based on media type
	if folder.MediaType == "movie" {
		s.processMovies(ctx, folder, scanResult.Movies, defaultProfile.ID, result, activity)
	} else {
		s.processEpisodes(ctx, folder, scanResult.Episodes, defaultProfile.ID, result, activity)
	}

	// Add scan errors to result
	for _, scanErr := range scanResult.Errors {
		result.Errors = append(result.Errors, fmt.Sprintf("%s: %s", scanErr.Path, scanErr.Error))
	}

	// Complete
	if activity != nil {
		summary := fmt.Sprintf("Found %d files", result.TotalFiles)
		if result.MoviesAdded > 0 {
			summary = fmt.Sprintf("Added %d movies", result.MoviesAdded)
		} else if result.SeriesAdded > 0 {
			summary = fmt.Sprintf("Added %d series", result.SeriesAdded)
		}
		activity.Complete(summary)
	}

	s.logger.Info().
		Int64("rootFolderId", rootFolderID).
		Int("totalFiles", result.TotalFiles).
		Int("moviesAdded", result.MoviesAdded).
		Int("seriesAdded", result.SeriesAdded).
		Int("filesLinked", result.FilesLinked).
		Int("errors", len(result.Errors)).
		Msg("Library scan completed")

	return result, nil
}

// processMovies processes scanned movie files.
func (s *Service) processMovies(
	ctx context.Context,
	folder *rootfolder.RootFolder,
	parsedMovies []scanner.ParsedMedia,
	qualityProfileID int64,
	result *ScanResult,
	activity *progress.ActivityBuilder,
) {
	total := len(parsedMovies)
	for i, parsed := range parsedMovies {
		// Update progress
		if activity != nil {
			pct := 0
			if total > 0 {
				pct = (i + 1) * 100 / total
			}
			subtitle := fmt.Sprintf("Processing: %s", filepath.Base(parsed.FilePath))
			activity.Update(subtitle, pct)
		}

		// Try to match to existing movie or create new one
		movie, created, err := s.matchOrCreateMovie(ctx, folder, parsed, qualityProfileID)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("failed to process %s: %v", parsed.FilePath, err))
			continue
		}

		if created {
			result.MoviesAdded++
		}

		// Add file to movie
		if movie != nil {
			err = s.addMovieFile(ctx, movie.ID, parsed)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("failed to add file %s: %v", parsed.FilePath, err))
			} else {
				result.FilesLinked++
			}
		}
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
) {
	// Group episodes by series (title)
	seriesMap := make(map[string][]scanner.ParsedMedia)
	for _, parsed := range parsedEpisodes {
		key := strings.ToLower(parsed.Title)
		seriesMap[key] = append(seriesMap[key], parsed)
	}

	total := len(parsedEpisodes)
	processedFiles := 0

	for _, episodes := range seriesMap {
		if len(episodes) == 0 {
			continue
		}

		// Use the first episode to identify the series
		firstEp := episodes[0]

		// Try to match or create series
		series, created, err := s.matchOrCreateSeries(ctx, folder, firstEp, qualityProfileID)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("failed to process series %s: %v", firstEp.Title, err))
			continue
		}

		if created {
			result.SeriesAdded++
		}

		// Process each episode file
		for _, parsed := range episodes {
			processedFiles++

			// Update progress
			if activity != nil {
				pct := 0
				if total > 0 {
					pct = processedFiles * 100 / total
				}
				subtitle := fmt.Sprintf("Processing: %s", filepath.Base(parsed.FilePath))
				activity.Update(subtitle, pct)
			}

			if series != nil {
				err = s.addEpisodeFile(ctx, series.ID, parsed)
				if err != nil {
					result.Errors = append(result.Errors, fmt.Sprintf("failed to add episode file %s: %v", parsed.FilePath, err))
				} else {
					result.FilesLinked++
				}
			}
		}
	}
}

// matchOrCreateMovie finds an existing movie or creates a new one from parsed media.
func (s *Service) matchOrCreateMovie(
	ctx context.Context,
	folder *rootfolder.RootFolder,
	parsed scanner.ParsedMedia,
	qualityProfileID int64,
) (*movies.Movie, bool, error) {
	searchQuery := parsed.Title
	if parsed.Year > 0 {
		searchQuery = fmt.Sprintf("%s %d", parsed.Title, parsed.Year)
	}

	// Check if we have metadata provider
	if !s.metadata.HasMovieProvider() {
		return s.createMovieFromParsed(ctx, folder, parsed, qualityProfileID, nil)
	}

	// Search metadata
	results, err := s.metadata.SearchMovies(ctx, searchQuery)
	if err != nil {
		s.logger.Warn().Err(err).Str("query", searchQuery).Msg("Metadata search failed, creating movie without metadata")
		return s.createMovieFromParsed(ctx, folder, parsed, qualityProfileID, nil)
	}

	// Find best match
	var bestMatch *metadata.MovieResult
	if len(results) > 0 {
		for i := range results {
			if results[i].Year == parsed.Year {
				bestMatch = &results[i]
				break
			}
		}
		if bestMatch == nil {
			bestMatch = &results[0]
		}

		// Check if movie with this TMDB ID already exists
		existing, err := s.movies.GetByTmdbID(ctx, bestMatch.ID)
		if err == nil && existing != nil {
			return existing, false, nil
		}
	}

	return s.createMovieFromParsed(ctx, folder, parsed, qualityProfileID, bestMatch)
}

// createMovieFromParsed creates a new movie from parsed media and optional metadata.
func (s *Service) createMovieFromParsed(
	ctx context.Context,
	folder *rootfolder.RootFolder,
	parsed scanner.ParsedMedia,
	qualityProfileID int64,
	meta *metadata.MovieResult,
) (*movies.Movie, bool, error) {
	input := movies.CreateMovieInput{
		Title:            parsed.Title,
		Year:             parsed.Year,
		RootFolderID:     folder.ID,
		QualityProfileID: qualityProfileID,
		Monitored:        true,
	}

	input.Path = movies.GenerateMoviePath(folder.Path, parsed.Title, parsed.Year)

	if meta != nil {
		input.Title = meta.Title
		input.Year = meta.Year
		input.TmdbID = meta.ID
		input.ImdbID = meta.ImdbID
		input.Overview = meta.Overview
		input.Runtime = meta.Runtime
		input.Path = movies.GenerateMoviePath(folder.Path, meta.Title, meta.Year)
	}

	movie, err := s.movies.Create(ctx, input)
	if err != nil {
		if errors.Is(err, movies.ErrDuplicateTmdbID) && meta != nil {
			existing, err := s.movies.GetByTmdbID(ctx, meta.ID)
			if err == nil {
				return existing, false, nil
			}
		}
		return nil, false, err
	}

	return movie, true, nil
}

// addMovieFile adds a file to a movie.
func (s *Service) addMovieFile(ctx context.Context, movieID int64, parsed scanner.ParsedMedia) error {
	input := movies.CreateMovieFileInput{
		Path:       parsed.FilePath,
		Size:       parsed.FileSize,
		Quality:    parsed.Quality,
		VideoCodec: parsed.Codec,
		Resolution: fmt.Sprintf("%dp", parsed.Resolution),
	}

	_, err := s.movies.AddFile(ctx, movieID, input)
	return err
}

// matchOrCreateSeries finds an existing series or creates a new one from parsed media.
func (s *Service) matchOrCreateSeries(
	ctx context.Context,
	folder *rootfolder.RootFolder,
	parsed scanner.ParsedMedia,
	qualityProfileID int64,
) (*tv.Series, bool, error) {
	if !s.metadata.HasSeriesProvider() {
		return s.createSeriesFromParsed(ctx, folder, parsed, qualityProfileID, nil)
	}

	results, err := s.metadata.SearchSeries(ctx, parsed.Title)
	if err != nil {
		s.logger.Warn().Err(err).Str("title", parsed.Title).Msg("Metadata search failed, creating series without metadata")
		return s.createSeriesFromParsed(ctx, folder, parsed, qualityProfileID, nil)
	}

	var bestMatch *metadata.SeriesResult
	if len(results) > 0 {
		bestMatch = &results[0]

		if bestMatch.TvdbID > 0 {
			existing, err := s.tv.GetSeriesByTvdbID(ctx, bestMatch.TvdbID)
			if err == nil && existing != nil {
				return existing, false, nil
			}
		}
	}

	return s.createSeriesFromParsed(ctx, folder, parsed, qualityProfileID, bestMatch)
}

// createSeriesFromParsed creates a new series from parsed media and optional metadata.
func (s *Service) createSeriesFromParsed(
	ctx context.Context,
	folder *rootfolder.RootFolder,
	parsed scanner.ParsedMedia,
	qualityProfileID int64,
	meta *metadata.SeriesResult,
) (*tv.Series, bool, error) {
	input := tv.CreateSeriesInput{
		Title:            parsed.Title,
		RootFolderID:     folder.ID,
		QualityProfileID: qualityProfileID,
		Monitored:        true,
		SeasonFolder:     true,
	}

	input.Path = tv.GenerateSeriesPath(folder.Path, parsed.Title)

	if meta != nil {
		input.Title = meta.Title
		input.Year = meta.Year
		input.TvdbID = meta.TvdbID
		input.TmdbID = meta.TmdbID
		input.ImdbID = meta.ImdbID
		input.Overview = meta.Overview
		input.Runtime = meta.Runtime
		input.Path = tv.GenerateSeriesPath(folder.Path, meta.Title)
	}

	series, err := s.tv.CreateSeries(ctx, input)
	if err != nil {
		if errors.Is(err, tv.ErrDuplicateTvdbID) && meta != nil && meta.TvdbID > 0 {
			existing, err := s.tv.GetSeriesByTvdbID(ctx, meta.TvdbID)
			if err == nil {
				return existing, false, nil
			}
		}
		return nil, false, err
	}

	return series, true, nil
}

// addEpisodeFile adds a file to an episode, creating the season/episode if needed.
func (s *Service) addEpisodeFile(ctx context.Context, seriesID int64, parsed scanner.ParsedMedia) error {
	episode, err := s.getOrCreateEpisode(ctx, seriesID, parsed.Season, parsed.Episode)
	if err != nil {
		return err
	}

	input := tv.CreateEpisodeFileInput{
		Path:       parsed.FilePath,
		Size:       parsed.FileSize,
		Quality:    parsed.Quality,
		VideoCodec: parsed.Codec,
		Resolution: fmt.Sprintf("%dp", parsed.Resolution),
	}

	_, err = s.tv.AddEpisodeFile(ctx, episode.ID, input)
	return err
}

// getOrCreateEpisode gets an existing episode or creates one.
func (s *Service) getOrCreateEpisode(ctx context.Context, seriesID int64, seasonNum, episodeNum int) (*tv.Episode, error) {
	episodes, err := s.tv.ListEpisodes(ctx, seriesID, &seasonNum)
	if err == nil {
		for i := range episodes {
			if episodes[i].EpisodeNumber == episodeNum {
				return &episodes[i], nil
			}
		}
	}

	// Ensure season exists
	if err := s.ensureSeasonExists(ctx, seriesID, seasonNum); err != nil {
		return nil, err
	}

	// Create episode
	_, err = s.queries.CreateEpisode(ctx, sqlc.CreateEpisodeParams{
		SeriesID:      seriesID,
		SeasonNumber:  int64(seasonNum),
		EpisodeNumber: int64(episodeNum),
		Title:         sql.NullString{String: fmt.Sprintf("Episode %d", episodeNum), Valid: true},
		Monitored:     1,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create episode: %w", err)
	}

	// Fetch the created episode
	episodes, err = s.tv.ListEpisodes(ctx, seriesID, &seasonNum)
	if err != nil {
		return nil, err
	}
	for i := range episodes {
		if episodes[i].EpisodeNumber == episodeNum {
			return &episodes[i], nil
		}
	}

	return nil, fmt.Errorf("failed to find created episode")
}

// ensureSeasonExists ensures a season exists for a series.
func (s *Service) ensureSeasonExists(ctx context.Context, seriesID int64, seasonNum int) error {
	_, err := s.queries.GetSeasonByNumber(ctx, sqlc.GetSeasonByNumberParams{
		SeriesID:     seriesID,
		SeasonNumber: int64(seasonNum),
	})
	if err == nil {
		return nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	_, err = s.queries.CreateSeason(ctx, sqlc.CreateSeasonParams{
		SeriesID:     seriesID,
		SeasonNumber: int64(seasonNum),
		Monitored:    1,
	})
	return err
}

// getDefaultQualityProfile returns the first available quality profile.
func (s *Service) getDefaultQualityProfile(ctx context.Context) (*quality.Profile, error) {
	profiles, err := s.qualityProfiles.List(ctx)
	if err != nil {
		return nil, err
	}
	if len(profiles) == 0 {
		return nil, ErrNoQualityProfile
	}
	return profiles[0], nil
}

// IsScanActive returns true if a scan is active for the given folder.
func (s *Service) IsScanActive(rootFolderID int64) bool {
	return s.isScanActive(rootFolderID)
}

// GetActiveScanActivity returns the activity ID for an active scan, or empty string if none.
func (s *Service) GetActiveScanActivity(rootFolderID int64) string {
	s.scanMu.RLock()
	defer s.scanMu.RUnlock()
	return s.activeScans[rootFolderID]
}

// CancelScan cancels an active scan.
func (s *Service) CancelScan(rootFolderID int64) bool {
	s.scanMu.Lock()
	defer s.scanMu.Unlock()

	activityID, exists := s.activeScans[rootFolderID]
	if !exists {
		return false
	}

	if s.progress != nil {
		s.progress.CancelActivity(activityID)
	}
	delete(s.activeScans, rootFolderID)
	return true
}

func (s *Service) isScanActive(rootFolderID int64) bool {
	s.scanMu.RLock()
	defer s.scanMu.RUnlock()
	_, exists := s.activeScans[rootFolderID]
	return exists
}

func (s *Service) setScanActive(rootFolderID int64, activityID string) {
	s.scanMu.Lock()
	defer s.scanMu.Unlock()
	s.activeScans[rootFolderID] = activityID
}

func (s *Service) clearScanActive(rootFolderID int64) {
	s.scanMu.Lock()
	defer s.scanMu.Unlock()
	delete(s.activeScans, rootFolderID)
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

	defaultProfile, err := s.getDefaultQualityProfile(ctx)
	if err != nil {
		return err
	}

	if folder.MediaType == "movie" {
		movie, _, err := s.matchOrCreateMovie(ctx, folder, *parsed, defaultProfile.ID)
		if err != nil {
			return err
		}
		if movie != nil {
			return s.addMovieFile(ctx, movie.ID, *parsed)
		}
	} else {
		series, _, err := s.matchOrCreateSeries(ctx, folder, *parsed, defaultProfile.ID)
		if err != nil {
			return err
		}
		if series != nil {
			return s.addEpisodeFile(ctx, series.ID, *parsed)
		}
	}

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
