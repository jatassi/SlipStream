package autosearch

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/history"
	"github.com/slipstream/slipstream/internal/indexer"
	"github.com/slipstream/slipstream/internal/indexer/grab"
	"github.com/slipstream/slipstream/internal/indexer/search"
	"github.com/slipstream/slipstream/internal/indexer/types"
	"github.com/slipstream/slipstream/internal/library/quality"
	"github.com/slipstream/slipstream/internal/library/scanner"
	"github.com/slipstream/slipstream/internal/library/slots"
)

var (
	ErrNoResults       = errors.New("no suitable releases found")
	ErrItemNotFound    = errors.New("item not found")
	ErrAlreadyInQueue  = errors.New("item already in download queue")
	ErrSearchCancelled = errors.New("search was cancelled")
)

// Broadcaster interface for sending events to clients.
type Broadcaster interface {
	Broadcast(msgType string, payload interface{}) error
}

// Service provides automatic release searching and grabbing functionality.
type Service struct {
	db             *sql.DB
	queries        *sqlc.Queries
	searchService  search.SearchService
	grabService    *grab.Service
	qualityService *quality.Service
	historyService *history.Service
	slotsService   *slots.Service
	broadcaster    Broadcaster
	logger         zerolog.Logger

	// Track currently running searches for cancellation
	mu               sync.RWMutex
	activeSearches   map[string]context.CancelFunc // key: "movie:123" or "episode:456"
}

// NewService creates a new automatic search service.
func NewService(
	db *sql.DB,
	searchService search.SearchService,
	grabService *grab.Service,
	qualityService *quality.Service,
	logger zerolog.Logger,
) *Service {
	return &Service{
		db:             db,
		queries:        sqlc.New(db),
		searchService:  searchService,
		grabService:    grabService,
		qualityService: qualityService,
		logger:         logger.With().Str("component", "autosearch").Logger(),
		activeSearches: make(map[string]context.CancelFunc),
	}
}

// SetDB updates the database connection used by this service.
// This is called when switching between production and development databases.
func (s *Service) SetDB(db *sql.DB) {
	s.db = db
	s.queries = sqlc.New(db)
}

// SetBroadcaster sets the WebSocket broadcaster for real-time events.
func (s *Service) SetBroadcaster(broadcaster Broadcaster) {
	s.broadcaster = broadcaster
}

// SetHistoryService sets the history service for event logging.
func (s *Service) SetHistoryService(historyService *history.Service) {
	s.historyService = historyService
}

// SearchMovie searches for a movie and grabs the best release.
func (s *Service) SearchMovie(ctx context.Context, movieID int64, source SearchSource) (*SearchResult, error) {
	// Get movie details
	movie, err := s.queries.GetMovie(ctx, movieID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrItemNotFound
		}
		return nil, fmt.Errorf("failed to get movie: %w", err)
	}

	item := s.movieToSearchableItem(ctx, movie)
	return s.searchAndGrab(ctx, item, source)
}

// SearchEpisode searches for an episode and grabs the best release.
// For first episodes of a season (E01), if the individual episode search fails,
// it will also try a season pack search as a fallback.
func (s *Service) SearchEpisode(ctx context.Context, episodeID int64, source SearchSource) (*SearchResult, error) {
	// Get episode details with series info
	episode, err := s.queries.GetEpisode(ctx, episodeID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrItemNotFound
		}
		return nil, fmt.Errorf("failed to get episode: %w", err)
	}

	// Get series for additional metadata
	series, err := s.queries.GetSeries(ctx, episode.SeriesID)
	if err != nil {
		return nil, fmt.Errorf("failed to get series: %w", err)
	}

	item := s.episodeToSearchableItem(ctx, episode, series)
	result, err := s.searchAndGrab(ctx, item, source)
	if err != nil {
		return result, err
	}

	// For first episodes of a season, try season pack search as fallback.
	// Only for missing episodes — upgradable episodes already had season pack
	// search attempted via SearchSeasonUpgrade before reaching here.
	if episode.EpisodeNumber == 1 && !result.Downloaded && !item.HasFile {
		s.logger.Debug().
			Int64("seriesId", episode.SeriesID).
			Int64("seasonNumber", episode.SeasonNumber).
			Msg("Episode 1 search didn't grab, trying season pack fallback")

		packResult, packErr := s.searchSeasonPack(ctx, series, int(episode.SeasonNumber), source)
		if packErr == nil && packResult.Downloaded {
			s.logger.Info().
				Int64("seriesId", episode.SeriesID).
				Int64("seasonNumber", episode.SeasonNumber).
				Msg("Season pack fallback succeeded for first episode")
			return packResult, nil
		}
	}

	return result, nil
}

// SearchSeason searches for all missing episodes in a season with boxset prioritization.
func (s *Service) SearchSeason(ctx context.Context, seriesID int64, seasonNumber int, source SearchSource) (*BatchSearchResult, error) {
	// Get series details
	series, err := s.queries.GetSeries(ctx, seriesID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrItemNotFound
		}
		return nil, fmt.Errorf("failed to get series: %w", err)
	}

	// Get missing episodes in this season
	episodes, err := s.getMissingEpisodesForSeason(ctx, seriesID, seasonNumber)
	if err != nil {
		// If season doesn't exist yet (e.g., series just added from portal request),
		// try a direct season pack search anyway
		s.logger.Debug().
			Err(err).
			Int64("seriesId", seriesID).
			Int("seasonNumber", seasonNumber).
			Msg("Failed to get missing episodes, attempting direct season pack search")

		packResult, searchErr := s.searchSeasonPack(ctx, series, seasonNumber, source)
		result := &BatchSearchResult{
			TotalSearched: 1,
			Results:       []*SearchResult{packResult},
		}
		if searchErr != nil {
			result.Failed = 1
			return result, nil
		}
		if packResult.Found {
			result.Found = 1
		}
		if packResult.Downloaded {
			result.Downloaded = 1
		}
		return result, nil
	}

	// Check if this season is eligible for season pack search
	eligible := s.isSeasonPackEligible(ctx, seriesID, seasonNumber)
	s.logger.Debug().
		Int64("seriesId", seriesID).
		Int("seasonNumber", seasonNumber).
		Int("missingEpisodes", len(episodes)).
		Bool("seasonPackEligible", eligible).
		Msg("Checking season pack eligibility for season search")

	if eligible {
		// Try season pack search first
		packResult, err := s.searchSeasonPack(ctx, series, seasonNumber, source)
		if err == nil && packResult.Found {
			result := &BatchSearchResult{
				TotalSearched: 1,
				Results:       []*SearchResult{packResult},
			}
			if packResult.Downloaded {
				result.Downloaded = 1
			}
			result.Found = 1
			return result, nil
		}

		// Season pack not found - fall back to individual episode search
		s.logger.Info().
			Int64("seriesId", seriesID).
			Int("seasonNumber", seasonNumber).
			Int("missingEpisodes", len(episodes)).
			Msg("No season pack found, falling back to individual episode search")
	}

	// If no missing episodes found but season exists (all episodes have files),
	// there's nothing to search for
	if len(episodes) == 0 {
		s.logger.Debug().
			Int64("seriesId", seriesID).
			Int("seasonNumber", seasonNumber).
			Msg("No missing episodes found for season")
		return &BatchSearchResult{
			TotalSearched: 0,
			Results:       []*SearchResult{},
		}, nil
	}

	// Use individual episode searches
	result := &BatchSearchResult{
		TotalSearched: len(episodes),
		Results:       make([]*SearchResult, 0, len(episodes)),
	}

	for _, ep := range episodes {
		item := s.episodeToSearchableItem(ctx, ep, series)
		searchResult, err := s.searchAndGrab(ctx, item, source)
		if err != nil {
			s.logger.Warn().Err(err).
				Int64("episodeId", ep.ID).
				Msg("Failed to search episode")
			result.Failed++
			result.Results = append(result.Results, &SearchResult{
				Error: err.Error(),
			})
			continue
		}

		result.Results = append(result.Results, searchResult)
		if searchResult.Found {
			result.Found++
		}
		if searchResult.Downloaded {
			result.Downloaded++
		}
	}

	return result, nil
}

// searchSeasonPack searches for a season pack release (internal method).
func (s *Service) searchSeasonPack(ctx context.Context, series *sqlc.Series, seasonNumber int, source SearchSource) (*SearchResult, error) {
	item := s.seriesToSeasonPackItem(series, seasonNumber)
	return s.searchAndGrab(ctx, item, source)
}

// SearchSeasonUpgrade searches for a season pack upgrade, falling back to individual episodes.
func (s *Service) SearchSeasonUpgrade(ctx context.Context, seriesID int64, seasonNumber int, currentQualityID int, source SearchSource) (*BatchSearchResult, error) {
	series, err := s.queries.GetSeries(ctx, seriesID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrItemNotFound
		}
		return nil, fmt.Errorf("failed to get series: %w", err)
	}

	// Try season pack search first
	item := s.seriesToSeasonPackItem(series, seasonNumber)
	item.HasFile = true
	item.CurrentQualityID = currentQualityID

	packResult, err := s.searchAndGrab(ctx, item, source)
	if err == nil && packResult.Downloaded {
		return &BatchSearchResult{
			TotalSearched: 1,
			Found:         1,
			Downloaded:    1,
			Results:       []*SearchResult{packResult},
		}, nil
	}

	s.logger.Info().
		Int64("seriesId", seriesID).
		Int("season", seasonNumber).
		Msg("No season pack found for upgrade, falling back to individual episodes")

	// Fall back to individual upgradable episodes
	episodes, err := s.getUpgradableEpisodesForSeason(ctx, seriesID, seasonNumber)
	if err != nil || len(episodes) == 0 {
		// Return the season pack result (no results found)
		result := &BatchSearchResult{TotalSearched: 1, Results: []*SearchResult{}}
		if packResult != nil {
			result.Results = append(result.Results, packResult)
		}
		return result, nil
	}

	result := &BatchSearchResult{
		TotalSearched: len(episodes),
		Results:       make([]*SearchResult, 0, len(episodes)),
	}

	for _, ep := range episodes {
		epResult, epErr := s.SearchEpisode(ctx, ep.ID, source)
		if epErr != nil {
			result.Failed++
			result.Results = append(result.Results, &SearchResult{Error: epErr.Error()})
			continue
		}
		result.Results = append(result.Results, epResult)
		if epResult.Found {
			result.Found++
		}
		if epResult.Downloaded {
			result.Downloaded++
		}
	}

	return result, nil
}

// getUpgradableEpisodesForSeason returns upgradable, monitored episodes for a season.
func (s *Service) getUpgradableEpisodesForSeason(ctx context.Context, seriesID int64, seasonNumber int) ([]*sqlc.Episode, error) {
	rows, err := s.queries.ListEpisodesBySeason(ctx, sqlc.ListEpisodesBySeasonParams{
		SeriesID:     seriesID,
		SeasonNumber: int64(seasonNumber),
	})
	if err != nil {
		return nil, err
	}

	upgradable := make([]*sqlc.Episode, 0)
	for _, row := range rows {
		if row.Status == "upgradable" && row.Monitored == 1 {
			upgradable = append(upgradable, row)
		}
	}
	return upgradable, nil
}

// searchSeasonPackByID searches for a season pack by series ID (for scheduled searches).
func (s *Service) searchSeasonPackByID(ctx context.Context, seriesID int64, seasonNumber int, source SearchSource) (*SearchResult, error) {
	series, err := s.queries.GetSeries(ctx, seriesID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrItemNotFound
		}
		return nil, fmt.Errorf("failed to get series: %w", err)
	}
	return s.searchSeasonPack(ctx, series, seasonNumber, source)
}

// SearchSeries searches for all missing episodes in a series with boxset prioritization.
func (s *Service) SearchSeries(ctx context.Context, seriesID int64, source SearchSource) (*BatchSearchResult, error) {
	// Get series details
	series, err := s.queries.GetSeries(ctx, seriesID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrItemNotFound
		}
		return nil, fmt.Errorf("failed to get series: %w", err)
	}

	// Get all missing episodes
	episodes, err := s.getMissingEpisodesForSeries(ctx, seriesID)
	if err != nil {
		return nil, fmt.Errorf("failed to get missing episodes: %w", err)
	}

	// Group episodes by season for boxset prioritization
	seasonEpisodes := make(map[int64][]*sqlc.Episode)
	for _, ep := range episodes {
		seasonEpisodes[ep.SeasonNumber] = append(seasonEpisodes[ep.SeasonNumber], ep)
	}

	// Build search items with boxset prioritization
	var items []SearchableItem

	for seasonNum, eps := range seasonEpisodes {
		// Check if this season is eligible for season pack search
		eligible := s.isSeasonPackEligible(ctx, seriesID, int(seasonNum))
		s.logger.Debug().
			Int64("seriesId", seriesID).
			Int64("seasonNumber", seasonNum).
			Int("missingEpisodes", len(eps)).
			Bool("seasonPackEligible", eligible).
			Msg("Checking season pack eligibility")

		if eligible {
			// Use season pack search
			items = append(items, s.seriesToSeasonPackItem(series, int(seasonNum)))
		} else {
			// Use individual episode searches
			for _, ep := range eps {
				items = append(items, s.episodeToSearchableItem(ctx, ep, series))
			}
		}
	}

	result := &BatchSearchResult{
		TotalSearched: len(items),
		Results:       make([]*SearchResult, 0, len(items)),
	}

	// Search each item
	for _, item := range items {
		searchResult, err := s.searchAndGrab(ctx, item, source)
		if err != nil {
			s.logger.Warn().Err(err).
				Str("mediaType", string(item.MediaType)).
				Int64("mediaId", item.MediaID).
				Msg("Failed to search item")
			result.Failed++
			result.Results = append(result.Results, &SearchResult{
				Error: err.Error(),
			})
			continue
		}

		result.Results = append(result.Results, searchResult)
		if searchResult.Found {
			result.Found++
		}
		if searchResult.Downloaded {
			result.Downloaded++
		}
	}

	return result, nil
}

// searchAndGrab is the core function that searches for a release and grabs the best one.
func (s *Service) searchAndGrab(ctx context.Context, item SearchableItem, source SearchSource) (*SearchResult, error) {
	searchKey := fmt.Sprintf("%s:%d", item.MediaType, item.MediaID)

	// Register this search and get a cancellable context
	ctx, cancel := s.registerSearch(searchKey)
	defer s.unregisterSearch(searchKey, cancel)

	// Broadcast search started
	s.broadcastStarted(item, source)

	s.logger.Info().
		Str("mediaType", string(item.MediaType)).
		Int64("mediaId", item.MediaID).
		Str("title", item.Title).
		Msg("Starting automatic search")

	// Get quality profile
	profile, err := s.qualityService.Get(ctx, item.QualityProfileID)
	if err != nil {
		s.logger.Warn().Err(err).Int64("profileId", item.QualityProfileID).
			Msg("Failed to get quality profile, using default")
		defaultProfile := quality.DefaultProfile()
		profile = &defaultProfile
	}

	// Build search criteria
	criteria := s.buildSearchCriteria(item)

	// Build scoring parameters
	scoringParams := search.ScoredSearchParams{
		QualityProfile: profile,
		SearchYear:     item.Year,
		SearchSeason:   item.SeasonNumber,
		SearchEpisode:  item.EpisodeNumber,
	}

	// Execute search
	searchResult, err := s.searchService.SearchTorrents(ctx, criteria, scoringParams)
	if err != nil {
		s.broadcastFailed(item, err.Error())
		s.logAutoSearchFailed(ctx, item, source, err.Error())
		return nil, fmt.Errorf("search failed: %w", err)
	}

	if len(searchResult.Releases) == 0 {
		s.logger.Debug().
			Str("title", item.Title).
			Msg("No releases found")
		result := &SearchResult{Found: false}
		s.broadcastCompleted(item, result)
		return result, nil
	}

	// Select best release (already sorted by score, highest first)
	bestRelease := s.selectBestRelease(searchResult.Releases, profile, item)
	if bestRelease == nil {
		s.logger.Debug().
			Str("title", item.Title).
			Msg("No acceptable releases found")
		result := &SearchResult{Found: false}
		s.broadcastCompleted(item, result)
		return result, nil
	}

	s.logger.Info().
		Str("title", item.Title).
		Str("release", bestRelease.Title).
		Float64("score", bestRelease.Score).
		Int("normalizedScore", bestRelease.NormalizedScore).
		Msg("Selected best release")

	// Determine if this is an upgrade
	isUpgrade := item.HasFile && item.CurrentQualityID > 0

	// Grab the release
	grabReq := grab.GrabRequest{
		Release:      &bestRelease.ReleaseInfo,
		MediaType:    string(item.MediaType),
		MediaID:      item.MediaID,
		SeriesID:     item.SeriesID,
		SeasonNumber: item.SeasonNumber,
		TargetSlotID: item.TargetSlotID,
		Source:       "auto-search",
	}
	if item.MediaType == MediaTypeSeason {
		grabReq.IsSeasonPack = true
		// Check if this is a complete series boxset by parsing the release title
		parsed := scanner.ParseFilename(bestRelease.Title)
		if parsed.IsCompleteSeries {
			grabReq.IsCompleteSeries = true
		}
	}
	grabResult, err := s.grabService.Grab(ctx, grabReq)
	if err != nil {
		s.broadcastFailed(item, err.Error())
		s.logAutoSearchFailed(ctx, item, source, err.Error())
		return nil, fmt.Errorf("grab failed: %w", err)
	}

	result := &SearchResult{
		Found:      true,
		Downloaded: grabResult.Success,
		Release:    bestRelease,
		Upgraded:   isUpgrade,
		ClientName: grabResult.ClientName,
		DownloadID: grabResult.DownloadID,
	}

	if !grabResult.Success {
		result.Error = grabResult.Error
		s.logAutoSearchFailed(ctx, item, source, grabResult.Error)
	} else {
		// Log successful download or upgrade
		s.logAutoSearchSuccess(ctx, item, source, bestRelease, grabResult, isUpgrade)
	}

	s.broadcastCompleted(item, result)

	s.logger.Info().
		Str("title", item.Title).
		Str("release", bestRelease.Title).
		Str("client", grabResult.ClientName).
		Bool("success", grabResult.Success).
		Msg("Automatic search completed")

	return result, nil
}

// registerSearch registers an active search and returns a cancellable context.
func (s *Service) registerSearch(key string) (context.Context, context.CancelFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Cancel any existing search for this item
	if existingCancel, exists := s.activeSearches[key]; exists {
		existingCancel()
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.activeSearches[key] = cancel
	return ctx, cancel
}

// unregisterSearch removes a search from active tracking.
func (s *Service) unregisterSearch(key string, _ context.CancelFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Remove the search from tracking
	delete(s.activeSearches, key)
}

// CancelSearch cancels an active search for a specific item.
func (s *Service) CancelSearch(mediaType MediaType, mediaID int64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := fmt.Sprintf("%s:%d", mediaType, mediaID)
	if cancel, exists := s.activeSearches[key]; exists {
		cancel()
		delete(s.activeSearches, key)
		return true
	}
	return false
}

// IsSearching returns true if a search is currently active for the item.
func (s *Service) IsSearching(mediaType MediaType, mediaID int64) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := fmt.Sprintf("%s:%d", mediaType, mediaID)
	_, exists := s.activeSearches[key]
	return exists
}

// buildSearchCriteria creates search criteria from a searchable item.
func (s *Service) buildSearchCriteria(item SearchableItem) types.SearchCriteria {
	criteria := types.SearchCriteria{
		Query: item.Title,
	}

	switch item.MediaType {
	case MediaTypeMovie:
		criteria.Type = "movie"
		criteria.Categories = indexer.MovieCategories()
		if item.ImdbID != "" {
			criteria.ImdbID = item.ImdbID
		}
		if item.TmdbID > 0 {
			criteria.TmdbID = item.TmdbID
		}
		if item.Year > 0 {
			criteria.Year = item.Year
		}

	case MediaTypeEpisode:
		criteria.Type = "tvsearch"
		criteria.Categories = indexer.TVCategories()
		if item.TvdbID > 0 {
			criteria.TvdbID = item.TvdbID
		}
		criteria.Season = item.SeasonNumber
		criteria.Episode = item.EpisodeNumber

	case MediaTypeSeason:
		criteria.Type = "tvsearch"
		// Don't filter by categories for season pack searches - some indexers
		// categorize season packs differently than individual episodes, and
		// filtering would exclude them. This matches manual search behavior.
		if item.TvdbID > 0 {
			criteria.TvdbID = item.TvdbID
		}
		// Don't set Season parameter for season pack searches. Setting it would
		// cause indexers to filter server-side, potentially excluding complete
		// series boxsets that aren't tagged with a specific season number.
		// Our client-side selectBestRelease handles filtering to accept:
		// - Single season packs matching the target season
		// - Complete series boxsets that include the target season
	}

	return criteria
}

// selectBestRelease selects the best release from scored results.
func (s *Service) selectBestRelease(releases []types.TorrentInfo, profile *quality.Profile, item SearchableItem) *types.TorrentInfo {
	// Releases are already sorted by score (highest first)
	seasonPackCandidates := 0
	seasonPacksFound := 0
	for i := range releases {
		release := &releases[i]

		// For TV searches, verify the release matches the target season/episode
		if item.MediaType == MediaTypeEpisode || item.MediaType == MediaTypeSeason {
			parsed := scanner.ParseFilename(release.Title)

			// Check if the release covers the target season
			if item.SeasonNumber > 0 && parsed.Season > 0 {
				if parsed.IsCompleteSeries && parsed.EndSeason > 0 {
					// Multi-season range (S01-04): check if target is within range
					if item.SeasonNumber < parsed.Season || item.SeasonNumber > parsed.EndSeason {
						continue // Target season not in range
					}
				} else if parsed.Season != item.SeasonNumber {
					continue // Wrong season (single season pack)
				}
			}
			// Note: parsed.Season == 0 means complete series (all seasons) - always matches

			// For specific episode searches, require exact episode match (no season packs)
			if item.MediaType == MediaTypeEpisode && item.EpisodeNumber > 0 {
				if parsed.Episode != item.EpisodeNumber {
					continue // Wrong episode or season pack - need exact match
				}
			}

			// For season pack searches, require actual season pack (not specials like E00)
			if item.MediaType == MediaTypeSeason {
				if !parsed.IsSeasonPack {
					// Log first few rejections for debugging
					if seasonPackCandidates < 5 {
						s.logger.Info().
							Str("release", release.Title).
							Int("parsedSeason", parsed.Season).
							Int("parsedEpisode", parsed.Episode).
							Bool("isSeasonPack", parsed.IsSeasonPack).
							Msg("Rejected release for season pack search")
					}
					seasonPackCandidates++
					continue // Individual episode or special - need season pack
				}
				// Found a valid season pack!
				seasonPacksFound++
				s.logger.Info().
					Str("release", release.Title).
					Int("parsedSeason", parsed.Season).
					Int("parsedEndSeason", parsed.EndSeason).
					Bool("isSeasonPack", parsed.IsSeasonPack).
					Bool("isCompleteSeries", parsed.IsCompleteSeries).
					Msg("Found season pack release")
			}
		}

		// Determine release quality
		releaseQualityID := 0
		if release.ScoreBreakdown != nil {
			releaseQualityID = release.ScoreBreakdown.QualityID
		}

		// Skip if quality is not acceptable
		if releaseQualityID > 0 && !profile.IsAcceptable(releaseQualityID) {
			s.logger.Debug().
				Str("release", release.Title).
				Int("qualityId", releaseQualityID).
				Str("qualityName", release.ScoreBreakdown.QualityName).
				Msg("Rejected - quality not acceptable")
			continue
		}

		// If item already has a file, only grab upgrades
		if item.HasFile {
			if item.CurrentQualityID == 0 {
				s.logger.Debug().
					Str("release", release.Title).
					Msg("Skipping release - have file but current quality unknown")
				continue
			}
			if releaseQualityID == 0 {
				s.logger.Debug().
					Str("release", release.Title).
					Int("currentQualityId", item.CurrentQualityID).
					Msg("Skipping release with unknown quality - already have file")
				continue
			}
			if !profile.IsUpgrade(item.CurrentQualityID, releaseQualityID) {
				s.logger.Debug().
					Str("release", release.Title).
					Int("currentQualityId", item.CurrentQualityID).
					Int("releaseQualityId", releaseQualityID).
					Msg("Skipping release - not an upgrade")
				continue
			}
		}

		return release
	}

	// Log season pack search summary
	if item.MediaType == MediaTypeSeason {
		if seasonPacksFound == 0 && seasonPackCandidates > 0 {
			s.logger.Info().
				Int("totalReleases", len(releases)).
				Int("individualEpisodes", seasonPackCandidates).
				Int("seasonPacksFound", seasonPacksFound).
				Int("targetSeason", item.SeasonNumber).
				Msg("No season pack found - all matching releases were individual episodes")
		} else if seasonPacksFound == 0 {
			s.logger.Info().
				Int("totalReleases", len(releases)).
				Int("targetSeason", item.SeasonNumber).
				Msg("No releases matched the target season")
		}
	}

	return nil
}

// movieToSearchableItem converts a movie database row to a SearchableItem.
func (s *Service) movieToSearchableItem(ctx context.Context, movie *sqlc.Movie) SearchableItem {
	item := SearchableItem{
		MediaType: MediaTypeMovie,
		MediaID:   movie.ID,
		Title:     movie.Title,
	}

	// Populate file info if movie has a file (needed for upgrade checks).
	// Use the highest quality file to prevent re-grabbing when duplicate records exist.
	if movie.Status == "upgradable" || movie.Status == "available" {
		item.HasFile = true
		files, err := s.queries.GetMovieFilesWithImportInfo(ctx, movie.ID)
		if err == nil && len(files) > 0 {
			for _, f := range files {
				if f.QualityID.Valid && int(f.QualityID.Int64) > item.CurrentQualityID {
					item.CurrentQualityID = int(f.QualityID.Int64)
				}
			}
		}
		s.logger.Debug().
			Int64("movieId", movie.ID).
			Str("status", movie.Status).
			Bool("hasFile", item.HasFile).
			Int("currentQualityId", item.CurrentQualityID).
			Msg("Movie upgrade check info")
	}

	if movie.Year.Valid {
		item.Year = int(movie.Year.Int64)
	}
	if movie.ImdbID.Valid {
		item.ImdbID = movie.ImdbID.String
	}
	if movie.TmdbID.Valid {
		item.TmdbID = int(movie.TmdbID.Int64)
	}
	if movie.QualityProfileID.Valid {
		item.QualityProfileID = movie.QualityProfileID.Int64
	}

	return item
}

// movieUpgradeCandidateToSearchableItem converts an upgrade candidate movie to a SearchableItem.
func (s *Service) movieUpgradeCandidateToSearchableItem(movie *sqlc.ListMovieUpgradeCandidatesRow) SearchableItem {
	item := SearchableItem{
		MediaType: MediaTypeMovie,
		MediaID:   movie.ID,
		Title:     movie.Title,
		HasFile:   true,
	}

	if movie.Year.Valid {
		item.Year = int(movie.Year.Int64)
	}
	if movie.ImdbID.Valid {
		item.ImdbID = movie.ImdbID.String
	}
	if movie.TmdbID.Valid {
		item.TmdbID = int(movie.TmdbID.Int64)
	}
	if movie.QualityProfileID.Valid {
		item.QualityProfileID = movie.QualityProfileID.Int64
	}
	if movie.CurrentQualityID.Valid {
		item.CurrentQualityID = int(movie.CurrentQualityID.Int64)
	}

	return item
}

// episodeUpgradeCandidateToSearchableItem converts an upgrade candidate row to a SearchableItem.
func (s *Service) episodeUpgradeCandidateToSearchableItem(row *sqlc.ListEpisodeUpgradeCandidatesRow) SearchableItem {
	item := SearchableItem{
		MediaType:     MediaTypeEpisode,
		MediaID:       row.ID,
		SeriesID:      row.SeriesID,
		Title:         row.SeriesTitle,
		SeasonNumber:  int(row.SeasonNumber),
		EpisodeNumber: int(row.EpisodeNumber),
		HasFile:       true,
	}

	if row.SeriesYear.Valid {
		item.Year = int(row.SeriesYear.Int64)
	}
	if row.SeriesTvdbID.Valid {
		item.TvdbID = int(row.SeriesTvdbID.Int64)
	}
	if row.SeriesTmdbID.Valid {
		item.TmdbID = int(row.SeriesTmdbID.Int64)
	}
	if row.SeriesImdbID.Valid {
		item.ImdbID = row.SeriesImdbID.String
	}
	if row.SeriesQualityProfileID.Valid {
		item.QualityProfileID = row.SeriesQualityProfileID.Int64
	}
	if row.CurrentQualityID.Valid {
		item.CurrentQualityID = int(row.CurrentQualityID.Int64)
	}

	return item
}

// episodeToSearchableItem converts an episode and series to a SearchableItem.
// It checks the episode status to populate file info for upgrade detection.
func (s *Service) episodeToSearchableItem(ctx context.Context, episode *sqlc.Episode, series *sqlc.Series) SearchableItem {
	item := SearchableItem{
		MediaType:     MediaTypeEpisode,
		MediaID:       episode.ID,
		SeriesID:      series.ID,
		SeasonNumber:  int(episode.SeasonNumber),
		EpisodeNumber: int(episode.EpisodeNumber),
	}

	// Use series title for search
	item.Title = series.Title

	// Populate file info if episode has a file (needed for upgrade checks).
	// Use the highest quality file to prevent re-grabbing when duplicate records exist.
	if episode.Status == "upgradable" || episode.Status == "available" {
		item.HasFile = true
		files, err := s.queries.ListEpisodeFilesByEpisode(ctx, episode.ID)
		if err == nil && len(files) > 0 {
			for _, f := range files {
				if f.QualityID.Valid && int(f.QualityID.Int64) > item.CurrentQualityID {
					item.CurrentQualityID = int(f.QualityID.Int64)
				}
			}
		}
		s.logger.Debug().
			Int64("episodeId", episode.ID).
			Str("status", episode.Status).
			Bool("hasFile", item.HasFile).
			Int("currentQualityId", item.CurrentQualityID).
			Int("fileCount", len(files)).
			Msg("Episode upgrade check info")
	}

	if series.Year.Valid {
		item.Year = int(series.Year.Int64)
	}
	if series.TvdbID.Valid {
		item.TvdbID = int(series.TvdbID.Int64)
	}
	if series.TmdbID.Valid {
		item.TmdbID = int(series.TmdbID.Int64)
	}
	if series.ImdbID.Valid {
		item.ImdbID = series.ImdbID.String
	}
	if series.QualityProfileID.Valid {
		item.QualityProfileID = series.QualityProfileID.Int64
	}

	return item
}

// seriesToSeasonPackItem converts a series and season number to a season pack SearchableItem.
func (s *Service) seriesToSeasonPackItem(series *sqlc.Series, seasonNumber int) SearchableItem {
	item := SearchableItem{
		MediaType:    MediaTypeSeason,
		MediaID:      series.ID,
		SeriesID:     series.ID, // Set SeriesID for download mapping
		Title:        series.Title,
		SeasonNumber: seasonNumber,
	}

	if series.Year.Valid {
		item.Year = int(series.Year.Int64)
	}
	if series.TvdbID.Valid {
		item.TvdbID = int(series.TvdbID.Int64)
	}
	if series.TmdbID.Valid {
		item.TmdbID = int(series.TmdbID.Int64)
	}
	if series.ImdbID.Valid {
		item.ImdbID = series.ImdbID.String
	}
	if series.QualityProfileID.Valid {
		item.QualityProfileID = series.QualityProfileID.Int64
	}

	return item
}

// getMissingEpisodesForSeason returns missing, monitored episodes for a specific season.
func (s *Service) getMissingEpisodesForSeason(ctx context.Context, seriesID int64, seasonNumber int) ([]*sqlc.Episode, error) {
	// Get series to check if it's monitored
	series, err := s.queries.GetSeries(ctx, seriesID)
	if err != nil {
		return nil, err
	}
	if series.Monitored != 1 {
		return []*sqlc.Episode{}, nil // Series not monitored
	}

	// Get season to check if it's monitored
	season, err := s.queries.GetSeasonByNumber(ctx, sqlc.GetSeasonByNumberParams{
		SeriesID:     seriesID,
		SeasonNumber: int64(seasonNumber),
	})
	if err != nil {
		return nil, err
	}
	if season.Monitored != 1 {
		return []*sqlc.Episode{}, nil // Season not monitored
	}

	// Get all episodes for this season
	rows, err := s.queries.ListEpisodesBySeason(ctx, sqlc.ListEpisodesBySeasonParams{
		SeriesID:     seriesID,
		SeasonNumber: int64(seasonNumber),
	})
	if err != nil {
		return nil, err
	}

	missing := make([]*sqlc.Episode, 0)
	for _, row := range rows {
		if row.Status == "missing" && row.Monitored == 1 {
			missing = append(missing, row)
		}
	}

	return missing, nil
}

// getMissingEpisodesForSeries returns all missing, monitored episodes for a series.
func (s *Service) getMissingEpisodesForSeries(ctx context.Context, seriesID int64) ([]*sqlc.Episode, error) {
	series, err := s.queries.GetSeries(ctx, seriesID)
	if err != nil {
		return nil, err
	}
	if series.Monitored != 1 {
		return []*sqlc.Episode{}, nil
	}

	seasons, err := s.queries.ListSeasonsBySeries(ctx, seriesID)
	if err != nil {
		return nil, err
	}
	monitoredSeasons := make(map[int64]bool)
	for _, season := range seasons {
		monitoredSeasons[season.SeasonNumber] = season.Monitored == 1
	}

	rows, err := s.queries.ListEpisodesBySeries(ctx, seriesID)
	if err != nil {
		return nil, err
	}

	missing := make([]*sqlc.Episode, 0)
	for _, row := range rows {
		if !monitoredSeasons[row.SeasonNumber] {
			continue
		}
		if row.Status == "missing" && row.Monitored == 1 {
			missing = append(missing, row)
		}
	}

	return missing, nil
}

// isSeasonPackEligible checks if a season pack search should be used.
// A season pack search is only used when ALL episodes in the season are:
// 1. Released (available)
// 2. Monitored
// 3. Missing (no file)
// This ensures we only grab a full season pack when we actually want all episodes.
func (s *Service) isSeasonPackEligible(ctx context.Context, seriesID int64, seasonNumber int) bool {
	// Check if season is monitored
	season, err := s.queries.GetSeasonByNumber(ctx, sqlc.GetSeasonByNumberParams{
		SeriesID:     seriesID,
		SeasonNumber: int64(seasonNumber),
	})
	if err != nil {
		s.logger.Debug().Err(err).Int64("seriesId", seriesID).Int("season", seasonNumber).
			Msg("Season pack ineligible: failed to get season")
		return false
	}
	if season.Monitored != 1 {
		s.logger.Debug().Int64("seriesId", seriesID).Int("season", seasonNumber).
			Msg("Season pack ineligible: season not monitored")
		return false
	}

	// Get all episodes for this season
	episodes, err := s.queries.ListEpisodesBySeason(ctx, sqlc.ListEpisodesBySeasonParams{
		SeriesID:     seriesID,
		SeasonNumber: int64(seasonNumber),
	})
	if err != nil {
		s.logger.Debug().Err(err).Int64("seriesId", seriesID).Int("season", seasonNumber).
			Msg("Season pack ineligible: failed to list episodes")
		return false
	}

	if len(episodes) == 0 {
		s.logger.Debug().Int64("seriesId", seriesID).Int("season", seasonNumber).
			Msg("Season pack ineligible: no episodes in season")
		return false
	}

	// Check that ALL episodes are released, monitored, and missing
	for _, ep := range episodes {
		if ep.Monitored != 1 {
			s.logger.Debug().Int64("seriesId", seriesID).Int("season", seasonNumber).
				Int64("episodeId", ep.ID).Int64("episodeNumber", ep.EpisodeNumber).
				Msg("Season pack ineligible: episode not monitored")
			return false
		}

		// Status must be "missing" - any other status (unreleased, available, upgradable, downloading, failed) disqualifies
		if ep.Status != "missing" {
			s.logger.Debug().Int64("seriesId", seriesID).Int("season", seasonNumber).
				Int64("episodeId", ep.ID).Int64("episodeNumber", ep.EpisodeNumber).
				Str("status", ep.Status).
				Msg("Season pack ineligible: episode status is not missing")
			return false
		}
	}

	// Only use season pack if there's more than one episode
	if len(episodes) <= 1 {
		s.logger.Debug().Int64("seriesId", seriesID).Int("season", seasonNumber).
			Int("episodeCount", len(episodes)).
			Msg("Season pack ineligible: only one episode in season")
		return false
	}

	s.logger.Debug().Int64("seriesId", seriesID).Int("season", seasonNumber).
		Int("episodeCount", len(episodes)).
		Msg("Season pack eligible: all episodes released, monitored, and missing")
	return true
}

// isSeasonPackUpgradeEligible checks if a season pack search should be used for upgrades.
// A season pack upgrade search is used when ALL monitored episodes in the season are upgradable.
func (s *Service) isSeasonPackUpgradeEligible(ctx context.Context, seriesID int64, seasonNumber int) bool {
	season, err := s.queries.GetSeasonByNumber(ctx, sqlc.GetSeasonByNumberParams{
		SeriesID:     seriesID,
		SeasonNumber: int64(seasonNumber),
	})
	if err != nil {
		return false
	}
	if season.Monitored != 1 {
		return false
	}

	episodes, err := s.queries.ListEpisodesBySeason(ctx, sqlc.ListEpisodesBySeasonParams{
		SeriesID:     seriesID,
		SeasonNumber: int64(seasonNumber),
	})
	if err != nil || len(episodes) == 0 {
		return false
	}

	upgradableCount := 0
	for _, ep := range episodes {
		if ep.Monitored != 1 {
			continue
		}
		if ep.Status != "upgradable" {
			return false
		}
		upgradableCount++
	}

	if upgradableCount <= 1 {
		return false
	}

	s.logger.Debug().Int64("seriesId", seriesID).Int("season", seasonNumber).
		Int("upgradableCount", upgradableCount).
		Msg("Season pack upgrade eligible: all monitored episodes upgradable")
	return true
}

// RetryMovie resets a failed movie back to missing or upgradable and clears backoff.
func (s *Service) RetryMovie(ctx context.Context, movieID int64) (*RetryResult, error) {
	movie, err := s.queries.GetMovie(ctx, movieID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrItemNotFound
		}
		return nil, fmt.Errorf("failed to get movie: %w", err)
	}

	if movie.Status != "failed" {
		return &RetryResult{
			NewStatus: movie.Status,
			Message:   "movie is not in failed state",
		}, nil
	}

	// Check if movie has any files to determine new status
	fileCount, err := s.queries.CountMovieFiles(ctx, movieID)
	if err != nil {
		return nil, fmt.Errorf("failed to count movie files: %w", err)
	}

	newStatus := "missing"
	if fileCount > 0 {
		newStatus = "upgradable"
	}

	// Reset status, clear active_download_id and status_message
	err = s.queries.UpdateMovieStatusWithDetails(ctx, sqlc.UpdateMovieStatusWithDetailsParams{
		ID:               movieID,
		Status:           newStatus,
		ActiveDownloadID: sql.NullString{},
		StatusMessage:    sql.NullString{},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update movie status: %w", err)
	}

	// Reset autosearch backoff
	_ = s.queries.ResetAllAutosearchFailuresForItem(ctx, sqlc.ResetAllAutosearchFailuresForItemParams{
		ItemType: "movie",
		ItemID:   movieID,
	})

	if s.historyService != nil {
		_ = s.historyService.LogStatusChanged(ctx, history.MediaTypeMovie, movieID, history.StatusChangedData{
			From: "failed", To: newStatus, Reason: "Manual retry",
		})
	}
	if s.broadcaster != nil {
		_ = s.broadcaster.Broadcast("movie:updated", map[string]any{"movieId": movieID})
	}

	return &RetryResult{
		NewStatus: newStatus,
		Message:   fmt.Sprintf("movie reset to %s", newStatus),
	}, nil
}

// RetryEpisode resets a failed episode back to missing or upgradable and clears backoff.
func (s *Service) RetryEpisode(ctx context.Context, episodeID int64) (*RetryResult, error) {
	episode, err := s.queries.GetEpisode(ctx, episodeID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrItemNotFound
		}
		return nil, fmt.Errorf("failed to get episode: %w", err)
	}

	if episode.Status != "failed" {
		return &RetryResult{
			NewStatus: episode.Status,
			Message:   "episode is not in failed state",
		}, nil
	}

	// Check if episode has any files to determine new status
	fileCount, err := s.queries.CountEpisodeFiles(ctx, episodeID)
	if err != nil {
		return nil, fmt.Errorf("failed to count episode files: %w", err)
	}

	newStatus := "missing"
	if fileCount > 0 {
		newStatus = "upgradable"
	}

	// Reset status, clear active_download_id and status_message
	err = s.queries.UpdateEpisodeStatusWithDetails(ctx, sqlc.UpdateEpisodeStatusWithDetailsParams{
		ID:               episodeID,
		Status:           newStatus,
		ActiveDownloadID: sql.NullString{},
		StatusMessage:    sql.NullString{},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update episode status: %w", err)
	}

	// Reset autosearch backoff for this episode
	_ = s.queries.ResetAllAutosearchFailuresForItem(ctx, sqlc.ResetAllAutosearchFailuresForItemParams{
		ItemType: "episode",
		ItemID:   episodeID,
	})
	// Also reset series-level backoff so season pack searches are unblocked
	_ = s.queries.ResetAllAutosearchFailuresForItem(ctx, sqlc.ResetAllAutosearchFailuresForItemParams{
		ItemType: "series",
		ItemID:   episode.SeriesID,
	})

	if s.historyService != nil {
		_ = s.historyService.LogStatusChanged(ctx, history.MediaTypeEpisode, episodeID, history.StatusChangedData{
			From: "failed", To: newStatus, Reason: "Manual retry",
		})
	}
	if s.broadcaster != nil {
		_ = s.broadcaster.Broadcast("series:updated", map[string]any{"id": episode.SeriesID})
	}

	return &RetryResult{
		NewStatus: newStatus,
		Message:   fmt.Sprintf("episode reset to %s", newStatus),
	}, nil
}

// Broadcast helpers

func (s *Service) broadcastStarted(item SearchableItem, source SearchSource) {
	if s.broadcaster == nil {
		return
	}
	s.broadcaster.Broadcast(EventAutoSearchStarted, AutoSearchStartedPayload{
		MediaType: item.MediaType,
		MediaID:   item.MediaID,
		Title:     item.Title,
		Source:    source,
	})
}

func (s *Service) broadcastCompleted(item SearchableItem, result *SearchResult) {
	if s.broadcaster == nil {
		return
	}
	payload := AutoSearchCompletedPayload{
		MediaType:  item.MediaType,
		MediaID:    item.MediaID,
		Title:      item.Title,
		Found:      result.Found,
		Downloaded: result.Downloaded,
		Upgraded:   result.Upgraded,
		ClientName: result.ClientName,
	}
	if result.Release != nil {
		payload.ReleaseName = result.Release.Title
	}
	s.broadcaster.Broadcast(EventAutoSearchCompleted, payload)
}

func (s *Service) broadcastFailed(item SearchableItem, errMsg string) {
	if s.broadcaster == nil {
		return
	}
	s.broadcaster.Broadcast(EventAutoSearchFailed, AutoSearchFailedPayload{
		MediaType: item.MediaType,
		MediaID:   item.MediaID,
		Title:     item.Title,
		Error:     errMsg,
	})
}

// History logging helpers

func (s *Service) logAutoSearchSuccess(ctx context.Context, item SearchableItem, source SearchSource, release *types.TorrentInfo, grabResult *grab.GrabResult, isUpgrade bool) {
	if s.historyService == nil {
		return
	}

	mediaType := history.MediaTypeMovie
	if item.MediaType == MediaTypeEpisode {
		mediaType = history.MediaTypeEpisode
	}

	qualityStr := ""
	if release.ScoreBreakdown != nil {
		qualityStr = release.ScoreBreakdown.QualityName
	}

	data := history.AutoSearchDownloadData{
		ReleaseName: release.Title,
		Indexer:     release.IndexerName,
		ClientName:  grabResult.ClientName,
		DownloadID:  grabResult.DownloadID,
		Source:      string(source),
		IsUpgrade:   isUpgrade,
	}
	if isUpgrade {
		data.NewQuality = qualityStr
	}
	if err := s.historyService.LogAutoSearchDownload(ctx, mediaType, item.MediaID, qualityStr, data); err != nil {
		s.logger.Warn().Err(err).Msg("Failed to log autosearch download event")
	}
}

func (s *Service) logAutoSearchFailed(ctx context.Context, item SearchableItem, source SearchSource, errMsg string) {
	if s.historyService == nil {
		return
	}

	mediaType := history.MediaTypeMovie
	if item.MediaType == MediaTypeEpisode {
		mediaType = history.MediaTypeEpisode
	}

	if err := s.historyService.LogAutoSearchFailed(ctx, mediaType, item.MediaID, history.AutoSearchFailedData{
		Error:  errMsg,
		Source: string(source),
	}); err != nil {
		s.logger.Warn().Err(err).Msg("Failed to log autosearch failed event")
	}
}
