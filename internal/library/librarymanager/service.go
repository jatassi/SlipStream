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

	"github.com/slipstream/slipstream/internal/autosearch"
	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/library/movies"
	"github.com/slipstream/slipstream/internal/library/quality"
	"github.com/slipstream/slipstream/internal/library/rootfolder"
	"github.com/slipstream/slipstream/internal/library/scanner"
	"github.com/slipstream/slipstream/internal/library/slots"
	"github.com/slipstream/slipstream/internal/library/tv"
	"github.com/slipstream/slipstream/internal/metadata"
	"github.com/slipstream/slipstream/internal/preferences"
	"github.com/slipstream/slipstream/internal/progress"
)

var (
	ErrNoMetadataProvider = errors.New("no metadata provider configured")
	ErrNoQualityProfile   = errors.New("no quality profile available")
	ErrScanInProgress     = errors.New("scan already in progress for this folder")
)

// ScanResult represents the final result of a scan operation.
type ScanResult struct {
	RootFolderID    int64    `json:"rootFolderId"`
	TotalFiles      int      `json:"totalFiles"`
	MoviesAdded     int      `json:"moviesAdded"`
	SeriesAdded     int      `json:"seriesAdded"`
	FilesLinked     int      `json:"filesLinked"`
	MetadataMatched int      `json:"metadataMatched"`
	ArtworksFetched int      `json:"artworksFetched"`
	Errors          []string `json:"errors,omitempty"`
}

// pendingArtwork tracks items that need artwork downloaded.
type pendingArtwork struct {
	movieMeta  []*metadata.MovieResult
	seriesMeta []*metadata.SeriesResult
}

// Service orchestrates library scanning, file matching, and metadata lookup.
type Service struct {
	db              *sql.DB
	queries         *sqlc.Queries
	scanner         *scanner.Service
	movies          *movies.Service
	tv              *tv.Service
	metadata        *metadata.Service
	artwork         *metadata.ArtworkDownloader
	rootfolders     *rootfolder.Service
	qualityProfiles *quality.Service
	progress        *progress.Manager
	logger          zerolog.Logger

	// Optional services for search-on-add
	autosearchSvc  *autosearch.Service
	preferencesSvc *preferences.Service

	// Optional slots service for multi-version support
	slotsSvc *slots.Service

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
	artworkSvc *metadata.ArtworkDownloader,
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
		artwork:         artworkSvc,
		rootfolders:     rootfolderSvc,
		qualityProfiles: qualityProfileSvc,
		progress:        progressMgr,
		logger:          logger.With().Str("component", "librarymanager").Logger(),
		activeScans:     make(map[int64]string),
	}
}

// SetDB updates the database connection used by this service.
// This is called when switching between production and development databases.
func (s *Service) SetDB(db *sql.DB) {
	s.db = db
	s.queries = sqlc.New(db)
}

// SetAutosearchService sets the optional autosearch service for search-on-add functionality
func (s *Service) SetAutosearchService(svc *autosearch.Service) {
	s.autosearchSvc = svc
}

// SetPreferencesService sets the optional preferences service for add-flow defaults
func (s *Service) SetPreferencesService(svc *preferences.Service) {
	s.preferencesSvc = svc
}

// SetSlotsService sets the optional slots service for multi-version slot assignment.
// Req 13.1.2: Auto-assign files to best matching slot based on quality profile matching
func (s *Service) SetSlotsService(svc *slots.Service) {
	s.slotsSvc = svc
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

	// Track pending artwork to download
	pending := &pendingArtwork{
		movieMeta:  make([]*metadata.MovieResult, 0),
		seriesMeta: make([]*metadata.SeriesResult, 0),
	}

	// Process results based on media type
	if folder.MediaType == "movie" {
		s.processMovies(ctx, folder, scanResult.Movies, defaultProfile.ID, result, activity, pending)
	} else {
		s.processEpisodes(ctx, folder, scanResult.Episodes, defaultProfile.ID, result, activity, pending)
	}

	// Try to match metadata for any previously unmatched items in this folder
	if folder.MediaType == "movie" {
		s.matchUnmatchedMovies(ctx, folder, result, activity, pending)
	} else {
		s.matchUnmatchedSeries(ctx, folder, result, activity, pending)
	}

	// Download artwork for newly added and newly matched items
	if s.artwork != nil && (len(pending.movieMeta) > 0 || len(pending.seriesMeta) > 0) {
		s.downloadPendingArtwork(ctx, pending, result, activity)
	}

	// Add scan errors to result
	for _, scanErr := range scanResult.Errors {
		result.Errors = append(result.Errors, fmt.Sprintf("%s: %s", scanErr.Path, scanErr.Error))
	}

	// Complete
	if activity != nil {
		summary := s.buildScanSummary(result)
		activity.Complete(summary)
	}

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
	pending *pendingArtwork,
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

		// Skip files that are already tracked
		existingFile, err := s.movies.GetFileByPath(ctx, parsed.FilePath)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("failed to check file %s: %v", parsed.FilePath, err))
			continue
		}
		if existingFile != nil {
			s.logger.Debug().Str("path", parsed.FilePath).Msg("Movie file already tracked, skipping")
			continue
		}

		// Try to match to existing movie or create new one
		movie, created, meta, err := s.matchOrCreateMovie(ctx, folder, parsed, qualityProfileID)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("failed to process %s: %v", parsed.FilePath, err))
			continue
		}

		if created {
			result.MoviesAdded++
			// Queue artwork for download if we have metadata with images
			if meta != nil && (meta.PosterURL != "" || meta.BackdropURL != "") {
				pending.movieMeta = append(pending.movieMeta, meta)
			}
		}

		// Add file to movie
		if movie != nil {
			if err := s.addMovieFile(ctx, movie.ID, parsed); err != nil {
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
	pending *pendingArtwork,
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

		// First, filter out files that are already tracked
		var newEpisodes []scanner.ParsedMedia
		for _, parsed := range episodes {
			processedFiles++

			// Update progress
			if activity != nil {
				pct := 0
				if total > 0 {
					pct = processedFiles * 100 / total
				}
				subtitle := fmt.Sprintf("Checking: %s", filepath.Base(parsed.FilePath))
				activity.Update(subtitle, pct)
			}

			existingFile, err := s.tv.GetEpisodeFileByPath(ctx, parsed.FilePath)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("failed to check file %s: %v", parsed.FilePath, err))
				continue
			}
			if existingFile != nil {
				s.logger.Debug().Str("path", parsed.FilePath).Msg("Episode file already tracked, skipping")
				continue
			}
			newEpisodes = append(newEpisodes, parsed)
		}

		// Skip series processing if all files are already tracked
		if len(newEpisodes) == 0 {
			continue
		}

		// Use the first new episode to identify the series
		firstEp := newEpisodes[0]

		// Try to match or create series
		series, created, meta, err := s.matchOrCreateSeries(ctx, folder, firstEp, qualityProfileID)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("failed to process series %s: %v", firstEp.Title, err))
			continue
		}

		if created {
			result.SeriesAdded++
			// Queue artwork for download if we have metadata with images
			if meta != nil && (meta.PosterURL != "" || meta.BackdropURL != "") {
				pending.seriesMeta = append(pending.seriesMeta, meta)
			}
		}

		// Add new episode files
		for _, parsed := range newEpisodes {
			if series != nil {
				if err := s.addEpisodeFile(ctx, series.ID, parsed); err != nil {
					result.Errors = append(result.Errors, fmt.Sprintf("failed to add episode file %s: %v", parsed.FilePath, err))
				} else {
					result.FilesLinked++
				}
			}
		}
	}
}

// matchOrCreateMovie finds an existing movie or creates a new one from parsed media.
// Returns the movie, whether it was created, the metadata used (if any), and any error.
func (s *Service) matchOrCreateMovie(
	ctx context.Context,
	folder *rootfolder.RootFolder,
	parsed scanner.ParsedMedia,
	qualityProfileID int64,
) (*movies.Movie, bool, *metadata.MovieResult, error) {
	// Check if we have metadata provider
	if !s.metadata.HasMovieProvider() {
		movie, created, err := s.createMovieFromParsed(ctx, folder, parsed, qualityProfileID, nil)
		return movie, created, nil, err
	}

	// Search metadata using title and year
	results, err := s.metadata.SearchMovies(ctx, parsed.Title, parsed.Year)
	if err != nil {
		s.logger.Warn().Err(err).Str("title", parsed.Title).Int("year", parsed.Year).Msg("Metadata search failed, creating movie without metadata")
		movie, created, err := s.createMovieFromParsed(ctx, folder, parsed, qualityProfileID, nil)
		return movie, created, nil, err
	}

	// Find best match - prefer exact year match
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
			return existing, false, nil, nil
		}
	}

	movie, created, err := s.createMovieFromParsed(ctx, folder, parsed, qualityProfileID, bestMatch)
	return movie, created, bestMatch, err
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

		// Fetch release dates from TMDB
		if meta.ID > 0 {
			digital, physical, err := s.metadata.GetMovieReleaseDates(ctx, meta.ID)
			if err != nil {
				s.logger.Warn().Err(err).Int("tmdbId", meta.ID).Msg("Failed to fetch release dates during scan")
			} else {
				input.ReleaseDate = digital
				input.PhysicalReleaseDate = physical
			}
		}
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

// addMovieFile adds a file to a movie and assigns it to a slot if multi-version is enabled.
// Req 13.1.2: Auto-assign files to best matching slot based on quality profile matching
// Req 13.1.3: Extra files (more than slot count) queued for user review (slot_id = NULL)
// Callers should check if the file already exists before calling this.
func (s *Service) addMovieFile(ctx context.Context, movieID int64, parsed scanner.ParsedMedia) error {
	input := movies.CreateMovieFileInput{
		Path:       parsed.FilePath,
		Size:       parsed.FileSize,
		Quality:    parsed.Quality,
		VideoCodec: parsed.Codec,
		Resolution: parsed.Quality,
	}

	file, err := s.movies.AddFile(ctx, movieID, input)
	if err != nil {
		return err
	}

	// Try to assign to a slot if slots service is available and multi-version is enabled
	if s.slotsSvc != nil && s.slotsSvc.IsMultiVersionEnabled(ctx) {
		assignment, err := s.slotsSvc.DetermineTargetSlot(ctx, &parsed, "movie", movieID)
		if err != nil {
			// No matching slot or all slots filled - file will be in review queue (slot_id = NULL)
			s.logger.Debug().
				Err(err).
				Int64("movieId", movieID).
				Int64("fileId", file.ID).
				Str("path", parsed.FilePath).
				Msg("Could not assign file to slot, will be in review queue")
			return nil
		}

		// Assign file to the determined slot
		if err := s.slotsSvc.AssignFileToSlot(ctx, "movie", movieID, assignment.SlotID, file.ID); err != nil {
			s.logger.Warn().
				Err(err).
				Int64("movieId", movieID).
				Int64("fileId", file.ID).
				Int64("slotId", assignment.SlotID).
				Msg("Failed to assign file to slot")
		} else {
			s.logger.Debug().
				Int64("movieId", movieID).
				Int64("fileId", file.ID).
				Int64("slotId", assignment.SlotID).
				Str("slotName", assignment.SlotName).
				Bool("isUpgrade", assignment.IsUpgrade).
				Bool("isNewFill", assignment.IsNewFill).
				Msg("Assigned movie file to slot")
		}
	}

	return nil
}

// matchOrCreateSeries finds an existing series or creates a new one from parsed media.
// Returns the series, whether it was created, the metadata used (if any), and any error.
func (s *Service) matchOrCreateSeries(
	ctx context.Context,
	folder *rootfolder.RootFolder,
	parsed scanner.ParsedMedia,
	qualityProfileID int64,
) (*tv.Series, bool, *metadata.SeriesResult, error) {
	if !s.metadata.HasSeriesProvider() {
		series, created, err := s.createSeriesFromParsed(ctx, folder, parsed, qualityProfileID, nil)
		return series, created, nil, err
	}

	results, err := s.metadata.SearchSeries(ctx, parsed.Title)
	if err != nil {
		s.logger.Warn().Err(err).Str("title", parsed.Title).Msg("Metadata search failed, creating series without metadata")
		series, created, err := s.createSeriesFromParsed(ctx, folder, parsed, qualityProfileID, nil)
		return series, created, nil, err
	}

	var bestMatch *metadata.SeriesResult
	if len(results) > 0 {
		bestMatch = &results[0]

		if bestMatch.TvdbID > 0 {
			existing, err := s.tv.GetSeriesByTvdbID(ctx, bestMatch.TvdbID)
			if err == nil && existing != nil {
				return existing, false, nil, nil
			}
		}
	}

	series, created, err := s.createSeriesFromParsed(ctx, folder, parsed, qualityProfileID, bestMatch)
	return series, created, bestMatch, err
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
// Req 13.1.2: Auto-assign files to best matching slot based on quality profile matching
// Req 13.1.3: Extra files (more than slot count) queued for user review (slot_id = NULL)
// Callers should check if the file already exists before calling this.
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
		Resolution: parsed.Quality,
	}

	file, err := s.tv.AddEpisodeFile(ctx, episode.ID, input)
	if err != nil {
		return err
	}

	// Try to assign to a slot if slots service is available and multi-version is enabled
	if s.slotsSvc != nil && s.slotsSvc.IsMultiVersionEnabled(ctx) {
		assignment, err := s.slotsSvc.DetermineTargetSlot(ctx, &parsed, "episode", episode.ID)
		if err != nil {
			// No matching slot or all slots filled - file will be in review queue (slot_id = NULL)
			s.logger.Debug().
				Err(err).
				Int64("episodeId", episode.ID).
				Int64("fileId", file.ID).
				Str("path", parsed.FilePath).
				Msg("Could not assign file to slot, will be in review queue")
			return nil
		}

		// Assign file to the determined slot
		if err := s.slotsSvc.AssignFileToSlot(ctx, "episode", episode.ID, assignment.SlotID, file.ID); err != nil {
			s.logger.Warn().
				Err(err).
				Int64("episodeId", episode.ID).
				Int64("fileId", file.ID).
				Int64("slotId", assignment.SlotID).
				Msg("Failed to assign file to slot")
		} else {
			s.logger.Debug().
				Int64("episodeId", episode.ID).
				Int64("fileId", file.ID).
				Int64("slotId", assignment.SlotID).
				Str("slotName", assignment.SlotName).
				Bool("isUpgrade", assignment.IsUpgrade).
				Bool("isNewFill", assignment.IsNewFill).
				Msg("Assigned episode file to slot")
		}
	}

	return nil
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

	if folder.MediaType == "movie" {
		// Check if movie file already exists in database
		existingFile, err := s.movies.GetFileByPath(ctx, parsed.FilePath)
		if err != nil {
			s.logger.Warn().Err(err).Str("path", parsed.FilePath).Msg("Failed to check for existing movie file")
		}
		if existingFile != nil {
			s.logger.Debug().Str("path", parsed.FilePath).Msg("Movie file already tracked, skipping")
			return nil
		}

		defaultProfile, err := s.getDefaultQualityProfile(ctx)
		if err != nil {
			return err
		}

		movie, created, meta, err := s.matchOrCreateMovie(ctx, folder, *parsed, defaultProfile.ID)
		if err != nil {
			return err
		}
		if movie != nil {
			if err := s.addMovieFile(ctx, movie.ID, *parsed); err != nil {
				return err
			}
			// Download artwork for newly created movie
			if created && meta != nil && s.artwork != nil {
				go func() {
					if err := s.artwork.DownloadMovieArtwork(context.Background(), meta); err != nil {
						s.logger.Warn().Err(err).Int("tmdbId", meta.ID).Msg("Failed to download movie artwork")
					}
				}()
			}
		}
	} else {
		// Check if episode file already exists in database
		existingFile, err := s.tv.GetEpisodeFileByPath(ctx, parsed.FilePath)
		if err != nil {
			s.logger.Warn().Err(err).Str("path", parsed.FilePath).Msg("Failed to check for existing episode file")
		}
		if existingFile != nil {
			s.logger.Debug().Str("path", parsed.FilePath).Msg("Episode file already tracked, skipping")
			return nil
		}

		defaultProfile, err := s.getDefaultQualityProfile(ctx)
		if err != nil {
			return err
		}

		series, created, meta, err := s.matchOrCreateSeries(ctx, folder, *parsed, defaultProfile.ID)
		if err != nil {
			return err
		}
		if series != nil {
			if err := s.addEpisodeFile(ctx, series.ID, *parsed); err != nil {
				return err
			}
			// Download artwork for newly created series
			if created && meta != nil && s.artwork != nil {
				go func() {
					if err := s.artwork.DownloadSeriesArtwork(context.Background(), meta); err != nil {
						s.logger.Warn().Err(err).Int("tvdbId", meta.TvdbID).Msg("Failed to download series artwork")
					}
				}()
			}
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

// downloadPendingArtwork downloads artwork for all pending items in batch.
// This is called after scan processing is complete to batch artwork downloads.
func (s *Service) downloadPendingArtwork(
	ctx context.Context,
	pending *pendingArtwork,
	result *ScanResult,
	activity *progress.ActivityBuilder,
) {
	totalItems := len(pending.movieMeta) + len(pending.seriesMeta)
	if totalItems == 0 {
		return
	}

	s.logger.Info().
		Int("movies", len(pending.movieMeta)).
		Int("series", len(pending.seriesMeta)).
		Msg("Downloading artwork for newly added items")

	if activity != nil {
		activity.Update("Downloading artwork...", -1)
		activity.SetMetadata("artworkTotal", totalItems)
	}

	downloaded := 0

	// Download movie artwork
	for i, movie := range pending.movieMeta {
		if activity != nil {
			pct := (i + 1) * 100 / totalItems
			activity.Update(fmt.Sprintf("Downloading artwork: %s", movie.Title), pct)
		}

		if err := s.artwork.DownloadMovieArtwork(ctx, movie); err != nil {
			s.logger.Warn().Err(err).
				Int("tmdbId", movie.ID).
				Str("title", movie.Title).
				Msg("Failed to download movie artwork")
		} else {
			downloaded++
		}
	}

	// Download series artwork
	for i, series := range pending.seriesMeta {
		if activity != nil {
			pct := (len(pending.movieMeta) + i + 1) * 100 / totalItems
			activity.Update(fmt.Sprintf("Downloading artwork: %s", series.Title), pct)
		}

		if err := s.artwork.DownloadSeriesArtwork(ctx, series); err != nil {
			s.logger.Warn().Err(err).
				Int("tvdbId", series.TvdbID).
				Str("title", series.Title).
				Msg("Failed to download series artwork")
		} else {
			downloaded++
		}
	}

	result.ArtworksFetched = downloaded

	if activity != nil {
		activity.SetMetadata("artworkDownloaded", downloaded)
	}

	s.logger.Info().
		Int("downloaded", downloaded).
		Int("total", totalItems).
		Msg("Artwork download complete")
}

// RefreshMovieMetadata fetches metadata for a single movie and downloads artwork.
func (s *Service) RefreshMovieMetadata(ctx context.Context, movieID int64) (*movies.Movie, error) {
	s.logger.Debug().Int64("movieId", movieID).Msg("[REFRESH] Starting movie metadata refresh")

	// Get the movie
	movie, err := s.movies.Get(ctx, movieID)
	if err != nil {
		s.logger.Error().Err(err).Int64("movieId", movieID).Msg("[REFRESH] Failed to get movie from database")
		return nil, fmt.Errorf("failed to get movie: %w", err)
	}

	s.logger.Debug().
		Int64("movieId", movieID).
		Str("title", movie.Title).
		Int("year", movie.Year).
		Int("currentTmdbId", movie.TmdbID).
		Msg("[REFRESH] Retrieved movie from database")

	// Check if we have a metadata provider
	if !s.metadata.HasMovieProvider() {
		s.logger.Warn().Msg("[REFRESH] No metadata provider configured")
		return nil, ErrNoMetadataProvider
	}

	s.logger.Debug().Msg("[REFRESH] Metadata provider is configured")

	// Search for metadata using title and year
	s.logger.Debug().Str("title", movie.Title).Int("year", movie.Year).Msg("[REFRESH] Searching for metadata")

	results, err := s.metadata.SearchMovies(ctx, movie.Title, movie.Year)
	if err != nil {
		s.logger.Error().Err(err).Str("title", movie.Title).Int("year", movie.Year).Msg("[REFRESH] Metadata search failed")
		return nil, fmt.Errorf("metadata search failed: %w", err)
	}

	s.logger.Debug().Int("resultCount", len(results)).Msg("[REFRESH] Metadata search completed")

	if len(results) == 0 {
		s.logger.Warn().Str("title", movie.Title).Int("year", movie.Year).Msg("[REFRESH] No metadata results found")
		return movie, nil // No results, return existing movie
	}

	// Log all results for debugging
	for i, r := range results {
		s.logger.Debug().
			Int("index", i).
			Int("tmdbId", r.ID).
			Str("title", r.Title).
			Int("year", r.Year).
			Str("imdbId", r.ImdbID).
			Str("posterUrl", r.PosterURL).
			Msg("[REFRESH] Search result")
	}

	// Find best match
	var bestMatch *metadata.MovieResult
	for i := range results {
		if results[i].Year == movie.Year {
			bestMatch = &results[i]
			s.logger.Debug().Int("index", i).Msg("[REFRESH] Found year match")
			break
		}
	}
	if bestMatch == nil {
		bestMatch = &results[0]
		s.logger.Debug().Msg("[REFRESH] No year match, using first result")
	}

	s.logger.Info().
		Int("tmdbId", bestMatch.ID).
		Str("title", bestMatch.Title).
		Int("year", bestMatch.Year).
		Str("imdbId", bestMatch.ImdbID).
		Str("overview", bestMatch.Overview[:min(100, len(bestMatch.Overview))]).
		Int("runtime", bestMatch.Runtime).
		Str("posterUrl", bestMatch.PosterURL).
		Str("backdropUrl", bestMatch.BackdropURL).
		Msg("[REFRESH] Best match selected")

	// Update movie with metadata
	title := bestMatch.Title
	year := bestMatch.Year
	tmdbID := bestMatch.ID
	imdbID := bestMatch.ImdbID
	overview := bestMatch.Overview
	runtime := bestMatch.Runtime

	// Fetch release dates from TMDB
	var releaseDate, physicalReleaseDate string
	if tmdbID > 0 {
		digital, physical, err := s.metadata.GetMovieReleaseDates(ctx, tmdbID)
		if err != nil {
			s.logger.Warn().Err(err).Int("tmdbId", tmdbID).Msg("[REFRESH] Failed to fetch release dates")
		} else {
			releaseDate = digital
			physicalReleaseDate = physical
			s.logger.Debug().
				Str("digital", digital).
				Str("physical", physical).
				Msg("[REFRESH] Fetched release dates from TMDB")
		}
	}

	s.logger.Debug().
		Str("title", title).
		Int("year", year).
		Int("tmdbId", tmdbID).
		Str("imdbId", imdbID).
		Int("runtime", runtime).
		Str("releaseDate", releaseDate).
		Msg("[REFRESH] Calling movies.Update with these values")

	updateInput := movies.UpdateMovieInput{
		Title:    &title,
		Year:     &year,
		TmdbID:   &tmdbID,
		ImdbID:   &imdbID,
		Overview: &overview,
		Runtime:  &runtime,
	}
	if releaseDate != "" {
		updateInput.ReleaseDate = &releaseDate
	}
	if physicalReleaseDate != "" {
		updateInput.PhysicalReleaseDate = &physicalReleaseDate
	}

	updatedMovie, err := s.movies.Update(ctx, movie.ID, updateInput)
	if err != nil {
		s.logger.Error().Err(err).Int64("movieId", movie.ID).Msg("[REFRESH] Failed to update movie in database")
		return nil, fmt.Errorf("failed to update movie: %w", err)
	}

	s.logger.Debug().
		Int64("movieId", updatedMovie.ID).
		Str("title", updatedMovie.Title).
		Int("tmdbId", updatedMovie.TmdbID).
		Str("imdbId", updatedMovie.ImdbID).
		Msg("[REFRESH] Movie updated in database, returned values")

	// Download artwork asynchronously
	if s.artwork != nil && (bestMatch.PosterURL != "" || bestMatch.BackdropURL != "") {
		s.logger.Debug().
			Str("posterUrl", bestMatch.PosterURL).
			Str("backdropUrl", bestMatch.BackdropURL).
			Msg("[REFRESH] Starting artwork download")
		go func() {
			if err := s.artwork.DownloadMovieArtwork(context.Background(), bestMatch); err != nil {
				s.logger.Warn().Err(err).Int("tmdbId", bestMatch.ID).Msg("[REFRESH] Failed to download movie artwork")
			} else {
				s.logger.Info().Int("tmdbId", bestMatch.ID).Msg("[REFRESH] Artwork download completed")
			}
		}()
	} else {
		s.logger.Debug().
			Bool("artworkNil", s.artwork == nil).
			Str("posterUrl", bestMatch.PosterURL).
			Str("backdropUrl", bestMatch.BackdropURL).
			Msg("[REFRESH] Skipping artwork download")
	}

	s.logger.Info().
		Int64("movieId", movie.ID).
		Str("title", bestMatch.Title).
		Int("tmdbId", bestMatch.ID).
		Msg("[REFRESH] Movie metadata refresh completed")

	return updatedMovie, nil
}

// RefreshSeriesMetadata fetches metadata for a single series and downloads artwork.
func (s *Service) RefreshSeriesMetadata(ctx context.Context, seriesID int64) (*tv.Series, error) {
	// Get the series
	series, err := s.tv.GetSeries(ctx, seriesID)
	if err != nil {
		return nil, fmt.Errorf("failed to get series: %w", err)
	}

	// Check if we have a metadata provider
	if !s.metadata.HasSeriesProvider() {
		return nil, ErrNoMetadataProvider
	}

	// Search for metadata
	results, err := s.metadata.SearchSeries(ctx, series.Title)
	if err != nil {
		return nil, fmt.Errorf("metadata search failed: %w", err)
	}

	if len(results) == 0 {
		return series, nil // No results, return existing series
	}

	bestMatch := &results[0]

	// Update series with metadata
	title := bestMatch.Title
	year := bestMatch.Year
	tvdbID := bestMatch.TvdbID
	tmdbID := bestMatch.TmdbID
	imdbID := bestMatch.ImdbID
	overview := bestMatch.Overview
	runtime := bestMatch.Runtime

	_, err = s.tv.UpdateSeries(ctx, series.ID, tv.UpdateSeriesInput{
		Title:    &title,
		Year:     &year,
		TvdbID:   &tvdbID,
		TmdbID:   &tmdbID,
		ImdbID:   &imdbID,
		Overview: &overview,
		Runtime:  &runtime,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update series: %w", err)
	}

	// Fetch and update seasons/episodes metadata
	if tmdbID > 0 || tvdbID > 0 {
		seasonResults, err := s.metadata.GetSeriesSeasons(ctx, tmdbID, tvdbID)
		if err != nil {
			s.logger.Warn().Err(err).Int("tmdbId", tmdbID).Int("tvdbId", tvdbID).Msg("Failed to fetch season metadata")
		} else {
			// Convert metadata.SeasonResult to tv.SeasonMetadata
			seasonMeta := make([]tv.SeasonMetadata, len(seasonResults))
			for i, sr := range seasonResults {
				episodes := make([]tv.EpisodeMetadata, len(sr.Episodes))
				for j, ep := range sr.Episodes {
					episodes[j] = tv.EpisodeMetadata{
						EpisodeNumber: ep.EpisodeNumber,
						SeasonNumber:  ep.SeasonNumber,
						Title:         ep.Title,
						Overview:      ep.Overview,
						AirDate:       ep.AirDate,
						Runtime:       ep.Runtime,
					}
				}
				seasonMeta[i] = tv.SeasonMetadata{
					SeasonNumber: sr.SeasonNumber,
					Name:         sr.Name,
					Overview:     sr.Overview,
					PosterURL:    sr.PosterURL,
					AirDate:      sr.AirDate,
					Episodes:     episodes,
				}
			}

			if err := s.tv.UpdateSeasonsFromMetadata(ctx, seriesID, seasonMeta); err != nil {
				s.logger.Warn().Err(err).Int64("seriesId", seriesID).Msg("Failed to update seasons from metadata")
			} else {
				totalEpisodes := 0
				for _, sm := range seasonMeta {
					totalEpisodes += len(sm.Episodes)
				}
				s.logger.Info().
					Int64("seriesId", seriesID).
					Int("seasons", len(seasonMeta)).
					Int("episodes", totalEpisodes).
					Msg("Updated seasons and episodes from metadata")
			}
		}
	}

	// Download artwork asynchronously
	if s.artwork != nil && (bestMatch.PosterURL != "" || bestMatch.BackdropURL != "") {
		go func() {
			if err := s.artwork.DownloadSeriesArtwork(context.Background(), bestMatch); err != nil {
				s.logger.Warn().Err(err).Int("tvdbId", bestMatch.TvdbID).Msg("Failed to download series artwork")
			}
		}()
	}

	s.logger.Info().
		Int64("seriesId", series.ID).
		Str("title", bestMatch.Title).
		Int("tvdbId", bestMatch.TvdbID).
		Msg("Refreshed series metadata")

	// Re-fetch series to include updated seasons
	return s.tv.GetSeries(ctx, seriesID)
}

// buildScanSummary creates a human-readable summary of scan results.
func (s *Service) buildScanSummary(result *ScanResult) string {
	var parts []string

	if result.MoviesAdded > 0 {
		parts = append(parts, fmt.Sprintf("%d movies added", result.MoviesAdded))
	}
	if result.SeriesAdded > 0 {
		parts = append(parts, fmt.Sprintf("%d series added", result.SeriesAdded))
	}
	if result.MetadataMatched > 0 {
		parts = append(parts, fmt.Sprintf("%d matched", result.MetadataMatched))
	}
	if result.ArtworksFetched > 0 {
		parts = append(parts, fmt.Sprintf("%d artworks", result.ArtworksFetched))
	}

	if len(parts) == 0 {
		return fmt.Sprintf("Found %d files", result.TotalFiles)
	}

	return strings.Join(parts, ", ")
}

// matchUnmatchedMovies finds movies without metadata and attempts to match them.
func (s *Service) matchUnmatchedMovies(
	ctx context.Context,
	folder *rootfolder.RootFolder,
	result *ScanResult,
	activity *progress.ActivityBuilder,
	pending *pendingArtwork,
) {
	if !s.metadata.HasMovieProvider() {
		return
	}

	unmatched, err := s.movies.ListUnmatchedByRootFolder(ctx, folder.ID)
	if err != nil {
		s.logger.Warn().Err(err).Int64("rootFolderId", folder.ID).Msg("Failed to list unmatched movies")
		return
	}

	if len(unmatched) == 0 {
		return
	}

	s.logger.Info().Int("count", len(unmatched)).Msg("Attempting to match unmatched movies")

	if activity != nil {
		activity.Update("Matching unmatched movies...", -1)
		activity.SetMetadata("unmatchedMovies", len(unmatched))
	}

	for i, movie := range unmatched {
		if activity != nil {
			pct := (i + 1) * 100 / len(unmatched)
			activity.Update(fmt.Sprintf("Matching: %s", movie.Title), pct)
		}

		// Search for metadata using title and year
		results, err := s.metadata.SearchMovies(ctx, movie.Title, movie.Year)
		if err != nil {
			s.logger.Warn().Err(err).Str("title", movie.Title).Int("year", movie.Year).Msg("Metadata search failed for unmatched movie")
			continue
		}

		if len(results) == 0 {
			continue
		}

		// Find best match - prefer exact year match
		var bestMatch *metadata.MovieResult
		for i := range results {
			if results[i].Year == movie.Year {
				bestMatch = &results[i]
				break
			}
		}
		if bestMatch == nil {
			bestMatch = &results[0]
		}

		// Update movie with metadata
		title := bestMatch.Title
		year := bestMatch.Year
		tmdbID := bestMatch.ID
		imdbID := bestMatch.ImdbID
		overview := bestMatch.Overview
		runtime := bestMatch.Runtime

		// Fetch release dates from TMDB
		updateInput := movies.UpdateMovieInput{
			Title:    &title,
			Year:     &year,
			TmdbID:   &tmdbID,
			ImdbID:   &imdbID,
			Overview: &overview,
			Runtime:  &runtime,
		}
		if tmdbID > 0 {
			digital, physical, err := s.metadata.GetMovieReleaseDates(ctx, tmdbID)
			if err != nil {
				s.logger.Warn().Err(err).Int("tmdbId", tmdbID).Msg("Failed to fetch release dates for unmatched movie")
			} else {
				if digital != "" {
					updateInput.ReleaseDate = &digital
				}
				if physical != "" {
					updateInput.PhysicalReleaseDate = &physical
				}
			}
		}

		_, err = s.movies.Update(ctx, movie.ID, updateInput)
		if err != nil {
			s.logger.Warn().Err(err).Int64("movieId", movie.ID).Msg("Failed to update movie with metadata")
			continue
		}

		result.MetadataMatched++

		// Queue artwork for download
		if bestMatch.PosterURL != "" || bestMatch.BackdropURL != "" {
			pending.movieMeta = append(pending.movieMeta, bestMatch)
		}

		s.logger.Info().
			Int64("movieId", movie.ID).
			Str("title", bestMatch.Title).
			Int("tmdbId", bestMatch.ID).
			Msg("Matched unmatched movie")
	}
}

// matchUnmatchedSeries finds series without metadata and attempts to match them.
func (s *Service) matchUnmatchedSeries(
	ctx context.Context,
	folder *rootfolder.RootFolder,
	result *ScanResult,
	activity *progress.ActivityBuilder,
	pending *pendingArtwork,
) {
	if !s.metadata.HasSeriesProvider() {
		return
	}

	unmatched, err := s.tv.ListUnmatchedByRootFolder(ctx, folder.ID)
	if err != nil {
		s.logger.Warn().Err(err).Int64("rootFolderId", folder.ID).Msg("Failed to list unmatched series")
		return
	}

	if len(unmatched) == 0 {
		return
	}

	s.logger.Info().Int("count", len(unmatched)).Msg("Attempting to match unmatched series")

	if activity != nil {
		activity.Update("Matching unmatched series...", -1)
		activity.SetMetadata("unmatchedSeries", len(unmatched))
	}

	for i, series := range unmatched {
		if activity != nil {
			pct := (i + 1) * 100 / len(unmatched)
			activity.Update(fmt.Sprintf("Matching: %s", series.Title), pct)
		}

		// Search for metadata
		results, err := s.metadata.SearchSeries(ctx, series.Title)
		if err != nil {
			s.logger.Warn().Err(err).Str("title", series.Title).Msg("Metadata search failed for unmatched series")
			continue
		}

		if len(results) == 0 {
			continue
		}

		bestMatch := &results[0]

		// Update series with metadata
		title := bestMatch.Title
		year := bestMatch.Year
		tvdbID := bestMatch.TvdbID
		tmdbID := bestMatch.TmdbID
		imdbID := bestMatch.ImdbID
		overview := bestMatch.Overview
		runtime := bestMatch.Runtime

		_, err = s.tv.UpdateSeries(ctx, series.ID, tv.UpdateSeriesInput{
			Title:    &title,
			Year:     &year,
			TvdbID:   &tvdbID,
			TmdbID:   &tmdbID,
			ImdbID:   &imdbID,
			Overview: &overview,
			Runtime:  &runtime,
		})
		if err != nil {
			s.logger.Warn().Err(err).Int64("seriesId", series.ID).Msg("Failed to update series with metadata")
			continue
		}

		result.MetadataMatched++

		// Queue artwork for download
		if bestMatch.PosterURL != "" || bestMatch.BackdropURL != "" {
			pending.seriesMeta = append(pending.seriesMeta, bestMatch)
		}

		s.logger.Info().
			Int64("seriesId", series.ID).
			Str("title", bestMatch.Title).
			Int("tvdbId", bestMatch.TvdbID).
			Msg("Matched unmatched series")
	}
}

// AddMovieInput contains fields for adding a movie with artwork.
type AddMovieInput struct {
	Title               string `json:"title"`
	Year                int    `json:"year,omitempty"`
	TmdbID              int    `json:"tmdbId,omitempty"`
	ImdbID              string `json:"imdbId,omitempty"`
	Overview            string `json:"overview,omitempty"`
	Runtime             int    `json:"runtime,omitempty"`
	Path                string `json:"path,omitempty"`
	RootFolderID        int64  `json:"rootFolderId"`
	QualityProfileID    int64  `json:"qualityProfileId"`
	Monitored           bool   `json:"monitored"`
	PosterURL           string `json:"posterUrl,omitempty"`
	BackdropURL         string `json:"backdropUrl,omitempty"`
	ReleaseDate         string `json:"releaseDate,omitempty"`         // Digital/streaming release date
	PhysicalReleaseDate string `json:"physicalReleaseDate,omitempty"` // Bluray release date
	SearchOnAdd         *bool  `json:"searchOnAdd,omitempty"`         // Trigger autosearch after add
}

// AddMovie creates a new movie and downloads artwork in the background.
func (s *Service) AddMovie(ctx context.Context, input AddMovieInput) (*movies.Movie, error) {
	// Fetch release dates from TMDB if we have a TMDB ID and no release dates provided
	releaseDate := input.ReleaseDate
	physicalReleaseDate := input.PhysicalReleaseDate

	if input.TmdbID > 0 && releaseDate == "" && physicalReleaseDate == "" {
		digital, physical, err := s.metadata.GetMovieReleaseDates(ctx, input.TmdbID)
		if err != nil {
			s.logger.Warn().Err(err).Int("tmdbId", input.TmdbID).Msg("Failed to fetch release dates from TMDB")
		} else {
			releaseDate = digital // Digital release is the main release date
			physicalReleaseDate = physical
			s.logger.Debug().
				Int("tmdbId", input.TmdbID).
				Str("digital", digital).
				Str("physical", physical).
				Msg("Fetched release dates from TMDB")
		}
	}

	// Create the movie
	movie, err := s.movies.Create(ctx, movies.CreateMovieInput{
		Title:               input.Title,
		Year:                input.Year,
		TmdbID:              input.TmdbID,
		ImdbID:              input.ImdbID,
		Overview:            input.Overview,
		Runtime:             input.Runtime,
		Path:                input.Path,
		RootFolderID:        input.RootFolderID,
		QualityProfileID:    input.QualityProfileID,
		Monitored:           input.Monitored,
		ReleaseDate:         releaseDate,
		PhysicalReleaseDate: physicalReleaseDate,
	})
	if err != nil {
		return nil, err
	}

	// Download artwork in the background if we have URLs
	if s.artwork != nil && input.TmdbID > 0 && (input.PosterURL != "" || input.BackdropURL != "") {
		go func() {
			movieResult := &metadata.MovieResult{
				ID:          input.TmdbID,
				Title:       input.Title,
				PosterURL:   input.PosterURL,
				BackdropURL: input.BackdropURL,
			}
			if err := s.artwork.DownloadMovieArtwork(context.Background(), movieResult); err != nil {
				s.logger.Warn().Err(err).Int("tmdbId", input.TmdbID).Msg("Failed to download movie artwork")
			} else {
				s.logger.Info().Int("tmdbId", input.TmdbID).Msg("Movie artwork downloaded")
			}
		}()
	}

	// Trigger autosearch in background if requested and movie is released
	if input.SearchOnAdd != nil && *input.SearchOnAdd && s.autosearchSvc != nil && movie.Released {
		go func() {
			s.logger.Info().Int64("movieId", movie.ID).Str("title", movie.Title).Msg("Triggering search-on-add for movie")
			if _, err := s.autosearchSvc.SearchMovie(context.Background(), movie.ID, autosearch.SearchSourceAdd); err != nil {
				s.logger.Warn().Err(err).Int64("movieId", movie.ID).Msg("Search-on-add failed for movie")
			}
		}()
	}

	// Save preference if provided
	if input.SearchOnAdd != nil && s.preferencesSvc != nil {
		go func() {
			if err := s.preferencesSvc.SetMovieSearchOnAdd(context.Background(), *input.SearchOnAdd); err != nil {
				s.logger.Warn().Err(err).Msg("Failed to save movie search-on-add preference")
			}
		}()
	}

	return movie, nil
}

// AddSeriesInput contains fields for adding a series with artwork.
type AddSeriesInput struct {
	Title            string           `json:"title"`
	Year             int              `json:"year,omitempty"`
	TvdbID           int              `json:"tvdbId,omitempty"`
	TmdbID           int              `json:"tmdbId,omitempty"`
	ImdbID           string           `json:"imdbId,omitempty"`
	Overview         string           `json:"overview,omitempty"`
	Runtime          int              `json:"runtime,omitempty"`
	Path             string           `json:"path,omitempty"`
	RootFolderID     int64            `json:"rootFolderId"`
	QualityProfileID int64            `json:"qualityProfileId"`
	Monitored        bool             `json:"monitored"`
	SeasonFolder     bool             `json:"seasonFolder"`
	Seasons          []tv.SeasonInput `json:"seasons,omitempty"`
	PosterURL        string           `json:"posterUrl,omitempty"`
	BackdropURL      string           `json:"backdropUrl,omitempty"`

	// Search and monitoring options for add flow
	SearchOnAdd     *string `json:"searchOnAdd,omitempty"`     // "no", "first_episode", "first_season", "latest_season", "all"
	MonitorOnAdd    *string `json:"monitorOnAdd,omitempty"`    // "none", "first_season", "latest_season", "future", "all"
	IncludeSpecials *bool   `json:"includeSpecials,omitempty"` // Whether to include specials in monitoring/search
}

// applyMonitoringOnAdd applies the monitoring-on-add settings to a newly added series
func (s *Service) applyMonitoringOnAdd(ctx context.Context, seriesID int64, monitorOnAdd string, includeSpecials bool) error {
	monitorType := preferences.SeriesMonitorOnAdd(monitorOnAdd)
	if !preferences.ValidSeriesMonitorOnAdd(monitorOnAdd) {
		monitorType = preferences.SeriesMonitorOnAddFuture // Default
	}

	switch monitorType {
	case preferences.SeriesMonitorOnAddNone:
		// Unmonitor everything
		if err := s.queries.UpdateAllEpisodesMonitoredBySeries(ctx, sqlc.UpdateAllEpisodesMonitoredBySeriesParams{
			Monitored: 0,
			SeriesID:  seriesID,
		}); err != nil {
			return err
		}
		if err := s.queries.UpdateSeasonMonitoredBySeries(ctx, sqlc.UpdateSeasonMonitoredBySeriesParams{
			Monitored: 0,
			SeriesID:  seriesID,
		}); err != nil {
			return err
		}
		// Also unmonitor the series itself
		if _, err := s.tv.UpdateSeries(ctx, seriesID, tv.UpdateSeriesInput{Monitored: boolPtr(false)}); err != nil {
			return err
		}

	case preferences.SeriesMonitorOnAddFirstSeason:
		// Monitor only first season (season 1, not 0)
		// First, unmonitor all
		if err := s.queries.UpdateAllEpisodesMonitoredBySeries(ctx, sqlc.UpdateAllEpisodesMonitoredBySeriesParams{
			Monitored: 0,
			SeriesID:  seriesID,
		}); err != nil {
			return err
		}
		if err := s.queries.UpdateSeasonMonitoredBySeries(ctx, sqlc.UpdateSeasonMonitoredBySeriesParams{
			Monitored: 0,
			SeriesID:  seriesID,
		}); err != nil {
			return err
		}
		// Then monitor season 1
		if err := s.queries.UpdateEpisodesMonitoredBySeason(ctx, sqlc.UpdateEpisodesMonitoredBySeasonParams{
			Monitored:    1,
			SeriesID:     seriesID,
			SeasonNumber: 1,
		}); err != nil {
			return err
		}
		if err := s.queries.UpdateSeasonMonitoredByNumber(ctx, sqlc.UpdateSeasonMonitoredByNumberParams{
			Monitored:    1,
			SeriesID:     seriesID,
			SeasonNumber: 1,
		}); err != nil {
			return err
		}

	case preferences.SeriesMonitorOnAddLatestSeason:
		// Get latest season number
		latestSeasonVal, err := s.queries.GetLatestSeasonNumber(ctx, seriesID)
		if err != nil {
			return err
		}
		// Unmonitor all, then monitor latest
		if err := s.queries.UpdateAllEpisodesMonitoredBySeries(ctx, sqlc.UpdateAllEpisodesMonitoredBySeriesParams{
			Monitored: 0,
			SeriesID:  seriesID,
		}); err != nil {
			return err
		}
		if err := s.queries.UpdateSeasonMonitoredBySeries(ctx, sqlc.UpdateSeasonMonitoredBySeriesParams{
			Monitored: 0,
			SeriesID:  seriesID,
		}); err != nil {
			return err
		}
		// Handle potential NULL value from MAX
		var latestSeason int64
		if latestSeasonVal != nil {
			switch v := latestSeasonVal.(type) {
			case int64:
				latestSeason = v
			case int:
				latestSeason = int64(v)
			}
		}
		if latestSeason > 0 {
			if err := s.queries.UpdateEpisodesMonitoredBySeason(ctx, sqlc.UpdateEpisodesMonitoredBySeasonParams{
				Monitored:    1,
				SeriesID:     seriesID,
				SeasonNumber: latestSeason,
			}); err != nil {
				return err
			}
			if err := s.queries.UpdateSeasonMonitoredByNumber(ctx, sqlc.UpdateSeasonMonitoredByNumberParams{
				Monitored:    1,
				SeriesID:     seriesID,
				SeasonNumber: latestSeason,
			}); err != nil {
				return err
			}
		}

	case preferences.SeriesMonitorOnAddFuture:
		// Monitor only unreleased episodes
		// First, unmonitor all released episodes
		if err := s.queries.UpdateAllEpisodesMonitoredBySeries(ctx, sqlc.UpdateAllEpisodesMonitoredBySeriesParams{
			Monitored: 0,
			SeriesID:  seriesID,
		}); err != nil {
			return err
		}
		// Then monitor unreleased
		if err := s.queries.UpdateFutureEpisodesMonitored(ctx, sqlc.UpdateFutureEpisodesMonitoredParams{
			Monitored: 1,
			SeriesID:  seriesID,
		}); err != nil {
			return err
		}
		// Seasons default to monitored

	case preferences.SeriesMonitorOnAddAll:
		// Everything is already monitored by default, nothing to do
	}

	// Handle specials (season 0) - if not including specials, unmonitor them
	if !includeSpecials {
		if err := s.queries.UpdateEpisodesMonitoredBySeason(ctx, sqlc.UpdateEpisodesMonitoredBySeasonParams{
			Monitored:    0,
			SeriesID:     seriesID,
			SeasonNumber: 0,
		}); err != nil {
			return err
		}
		if err := s.queries.UpdateSeasonMonitoredByNumber(ctx, sqlc.UpdateSeasonMonitoredByNumberParams{
			Monitored:    0,
			SeriesID:     seriesID,
			SeasonNumber: 0,
		}); err != nil {
			return err
		}
	}

	return nil
}

func boolPtr(b bool) *bool {
	return &b
}

// AddSeries creates a new series, fetches metadata, and downloads artwork in the background.
func (s *Service) AddSeries(ctx context.Context, input AddSeriesInput) (*tv.Series, error) {
	// Create the series
	series, err := s.tv.CreateSeries(ctx, tv.CreateSeriesInput{
		Title:            input.Title,
		Year:             input.Year,
		TvdbID:           input.TvdbID,
		TmdbID:           input.TmdbID,
		ImdbID:           input.ImdbID,
		Overview:         input.Overview,
		Runtime:          input.Runtime,
		Path:             input.Path,
		RootFolderID:     input.RootFolderID,
		QualityProfileID: input.QualityProfileID,
		Monitored:        input.Monitored,
		SeasonFolder:     input.SeasonFolder,
		Seasons:          input.Seasons,
	})
	if err != nil {
		return nil, err
	}

	// Fetch and update seasons/episodes metadata
	if input.TmdbID > 0 || input.TvdbID > 0 {
		seasonResults, err := s.metadata.GetSeriesSeasons(ctx, input.TmdbID, input.TvdbID)
		if err != nil {
			s.logger.Warn().Err(err).Int("tmdbId", input.TmdbID).Int("tvdbId", input.TvdbID).Msg("Failed to fetch season metadata for new series")
		} else {
			// Convert metadata.SeasonResult to tv.SeasonMetadata
			seasonMeta := make([]tv.SeasonMetadata, len(seasonResults))
			for i, sr := range seasonResults {
				episodes := make([]tv.EpisodeMetadata, len(sr.Episodes))
				for j, ep := range sr.Episodes {
					episodes[j] = tv.EpisodeMetadata{
						EpisodeNumber: ep.EpisodeNumber,
						SeasonNumber:  ep.SeasonNumber,
						Title:         ep.Title,
						Overview:      ep.Overview,
						AirDate:       ep.AirDate,
						Runtime:       ep.Runtime,
					}
				}
				seasonMeta[i] = tv.SeasonMetadata{
					SeasonNumber: sr.SeasonNumber,
					Name:         sr.Name,
					Overview:     sr.Overview,
					PosterURL:    sr.PosterURL,
					AirDate:      sr.AirDate,
					Episodes:     episodes,
				}
			}

			if err := s.tv.UpdateSeasonsFromMetadata(ctx, series.ID, seasonMeta); err != nil {
				s.logger.Warn().Err(err).Int64("seriesId", series.ID).Msg("Failed to update seasons from metadata for new series")
			} else {
				totalEpisodes := 0
				for _, sm := range seasonMeta {
					totalEpisodes += len(sm.Episodes)
				}
				s.logger.Info().
					Int64("seriesId", series.ID).
					Int("seasons", len(seasonMeta)).
					Int("episodes", totalEpisodes).
					Msg("Updated seasons and episodes for new series")
			}
		}
	}

	// Download artwork in the background if we have URLs
	// Use TmdbID for artwork storage since frontend expects artwork keyed by TMDB ID
	artworkID := input.TmdbID
	if artworkID == 0 {
		artworkID = input.TvdbID
	}
	if s.artwork != nil && artworkID > 0 && (input.PosterURL != "" || input.BackdropURL != "") {
		go func() {
			seriesResult := &metadata.SeriesResult{
				ID:          artworkID,
				TmdbID:      input.TmdbID,
				TvdbID:      input.TvdbID,
				Title:       input.Title,
				PosterURL:   input.PosterURL,
				BackdropURL: input.BackdropURL,
			}
			if err := s.artwork.DownloadSeriesArtwork(context.Background(), seriesResult); err != nil {
				s.logger.Warn().Err(err).Int("tmdbId", input.TmdbID).Int("tvdbId", input.TvdbID).Msg("Failed to download series artwork")
			} else {
				s.logger.Info().Int("tmdbId", input.TmdbID).Int("tvdbId", input.TvdbID).Msg("Series artwork downloaded")
			}
		}()
	}

	// Apply monitoring-on-add settings if provided
	if input.MonitorOnAdd != nil {
		includeSpecials := false
		if input.IncludeSpecials != nil {
			includeSpecials = *input.IncludeSpecials
		}
		if err := s.applyMonitoringOnAdd(ctx, series.ID, *input.MonitorOnAdd, includeSpecials); err != nil {
			s.logger.Warn().Err(err).Int64("seriesId", series.ID).Msg("Failed to apply monitoring-on-add settings")
		}
	}

	// Save preferences if provided
	if s.preferencesSvc != nil {
		go func() {
			if input.SearchOnAdd != nil {
				if err := s.preferencesSvc.SetSeriesSearchOnAdd(context.Background(), preferences.SeriesSearchOnAdd(*input.SearchOnAdd)); err != nil {
					s.logger.Warn().Err(err).Msg("Failed to save series search-on-add preference")
				}
			}
			if input.MonitorOnAdd != nil {
				if err := s.preferencesSvc.SetSeriesMonitorOnAdd(context.Background(), preferences.SeriesMonitorOnAdd(*input.MonitorOnAdd)); err != nil {
					s.logger.Warn().Err(err).Msg("Failed to save series monitor-on-add preference")
				}
			}
			if input.IncludeSpecials != nil {
				if err := s.preferencesSvc.SetSeriesIncludeSpecials(context.Background(), *input.IncludeSpecials); err != nil {
					s.logger.Warn().Err(err).Msg("Failed to save series include-specials preference")
				}
			}
		}()
	}

	// Trigger autosearch in background if requested
	if input.SearchOnAdd != nil && *input.SearchOnAdd != "no" && s.autosearchSvc != nil {
		go func() {
			s.triggerSeriesSearchOnAdd(series.ID, *input.SearchOnAdd, input.IncludeSpecials)
		}()
	}

	// Re-fetch series to include updated seasons and episodes
	return s.tv.GetSeries(ctx, series.ID)
}

// triggerSeriesSearchOnAdd triggers autosearch based on the search-on-add option
func (s *Service) triggerSeriesSearchOnAdd(seriesID int64, searchOnAdd string, includeSpecials *bool) {
	ctx := context.Background()
	searchType := preferences.SeriesSearchOnAdd(searchOnAdd)

	// Get series info
	series, err := s.tv.GetSeries(ctx, seriesID)
	if err != nil {
		s.logger.Warn().Err(err).Int64("seriesId", seriesID).Msg("Failed to get series for search-on-add")
		return
	}

	s.logger.Info().Int64("seriesId", seriesID).Str("title", series.Title).Str("searchType", searchOnAdd).Msg("Triggering search-on-add for series")

	switch searchType {
	case preferences.SeriesSearchOnAddFirstEpisode:
		// Search for S01E01 only
		seasonNum := 1
		episodes, err := s.tv.ListEpisodes(ctx, seriesID, &seasonNum)
		if err != nil {
			s.logger.Warn().Err(err).Int64("seriesId", seriesID).Msg("Failed to get season 1 episodes")
			return
		}
		for _, ep := range episodes {
			if ep.EpisodeNumber == 1 && ep.Released {
				if _, err := s.autosearchSvc.SearchEpisode(ctx, ep.ID, autosearch.SearchSourceAdd); err != nil {
					s.logger.Warn().Err(err).Int64("episodeId", ep.ID).Msg("Search-on-add failed for episode")
				}
				return
			}
		}

	case preferences.SeriesSearchOnAddFirstSeason:
		// Search for all released episodes in season 1
		if _, err := s.autosearchSvc.SearchSeason(ctx, seriesID, 1, autosearch.SearchSourceAdd); err != nil {
			s.logger.Warn().Err(err).Int64("seriesId", seriesID).Msg("Search-on-add failed for first season")
		}

	case preferences.SeriesSearchOnAddLatestSeason:
		// Find and search the latest season
		var latestSeason int
		for _, season := range series.Seasons {
			if season.SeasonNumber > latestSeason && season.SeasonNumber > 0 {
				latestSeason = season.SeasonNumber
			}
		}
		if latestSeason > 0 {
			if _, err := s.autosearchSvc.SearchSeason(ctx, seriesID, latestSeason, autosearch.SearchSourceAdd); err != nil {
				s.logger.Warn().Err(err).Int64("seriesId", seriesID).Int("season", latestSeason).Msg("Search-on-add failed for latest season")
			}
		}

	case preferences.SeriesSearchOnAddAll:
		// Search for entire series
		if _, err := s.autosearchSvc.SearchSeries(ctx, seriesID, autosearch.SearchSourceAdd); err != nil {
			s.logger.Warn().Err(err).Int64("seriesId", seriesID).Msg("Search-on-add failed for series")
		}
	}
}
