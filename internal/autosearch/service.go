package autosearch

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"

	"github.com/rs/zerolog"
	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/decisioning"
	"github.com/slipstream/slipstream/internal/history"
	"github.com/slipstream/slipstream/internal/indexer"
	"github.com/slipstream/slipstream/internal/indexer/grab"
	"github.com/slipstream/slipstream/internal/indexer/search"
	"github.com/slipstream/slipstream/internal/indexer/types"
	"github.com/slipstream/slipstream/internal/library/quality"
	"github.com/slipstream/slipstream/internal/library/scanner"
	"github.com/slipstream/slipstream/internal/library/slots"
)

const (
	statusUpgradable = "upgradable"
)

var (
	ErrNoResults       = errors.New("no suitable releases found")
	ErrItemNotFound    = errors.New("item not found")
	ErrAlreadyInQueue  = errors.New("item already in download queue")
	ErrSearchCancelled = errors.New("search was cancelled")
)

// Broadcaster interface for sending events to clients.
type Broadcaster interface {
	Broadcast(msgType string, payload interface{})
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
	grabLock       *decisioning.GrabLock
	broadcaster    Broadcaster
	logger         *zerolog.Logger

	// Track currently running searches for cancellation
	mu             sync.RWMutex
	activeSearches map[string]context.CancelFunc // key: "movie:123" or "episode:456"
}

// NewService creates a new automatic search service.
func NewService(
	db *sql.DB,
	searchService search.SearchService,
	grabService *grab.Service,
	qualityService *quality.Service,
	logger *zerolog.Logger,
) *Service {
	loggerWithComponent := logger.With().Str("component", "autosearch").Logger()
	return &Service{
		db:             db,
		queries:        sqlc.New(db),
		searchService:  searchService,
		grabService:    grabService,
		qualityService: qualityService,
		logger:         &loggerWithComponent,
		activeSearches: make(map[string]context.CancelFunc),
	}
}

// SetDB updates the database connection used by this service.
// This is called when switching between production and development databases.
func (s *Service) SetDB(db *sql.DB) {
	s.db = db
	s.queries = sqlc.New(db)
}

// SetGrabLock sets the shared grab lock for concurrent grab protection.
func (s *Service) SetGrabLock(lock *decisioning.GrabLock) {
	s.grabLock = lock
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
	return s.searchAndGrab(ctx, &item, source)
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
	result, err := s.searchAndGrab(ctx, &item, source)
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
// getSeriesForSearch retrieves series details with error handling.
func (s *Service) getSeriesForSearch(ctx context.Context, seriesID int64) (*sqlc.Series, error) {
	series, err := s.queries.GetSeries(ctx, seriesID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrItemNotFound
		}
		return nil, fmt.Errorf("failed to get series: %w", err)
	}
	return series, nil
}

// tryDirectSeasonPackSearch attempts season pack search when episode data is unavailable.
func (s *Service) tryDirectSeasonPackSearch(ctx context.Context, series *sqlc.Series, seasonNumber int, source SearchSource, originalErr error) (*BatchSearchResult, error) {
	s.logger.Debug().
		Err(originalErr).
		Int64("seriesId", series.ID).
		Int("seasonNumber", seasonNumber).
		Msg("Failed to get missing episodes, attempting direct season pack search")

	packResult, searchErr := s.searchSeasonPack(ctx, series, seasonNumber, source)
	result := &BatchSearchResult{
		TotalSearched: 1,
		Results:       []*SearchResult{packResult},
	}
	if searchErr != nil {
		result.Failed = 1
		return result, nil //nolint:nilerr // intentional: season pack search failure is non-fatal, return partial result
	}
	if packResult.Found {
		result.Found = 1
	}
	if packResult.Downloaded {
		result.Downloaded = 1
	}
	return result, nil
}

// trySeasonPackFirst attempts season pack search and returns result if successful.
func (s *Service) trySeasonPackFirst(ctx context.Context, series *sqlc.Series, seasonNumber int, source SearchSource) (*BatchSearchResult, bool) {
	packResult, err := s.searchSeasonPack(ctx, series, seasonNumber, source)
	if err != nil || !packResult.Found {
		return nil, false
	}

	result := &BatchSearchResult{
		TotalSearched: 1,
		Found:         1,
		Results:       []*SearchResult{packResult},
	}
	if packResult.Downloaded {
		result.Downloaded = 1
	}
	return result, true
}

// searchEpisodesIndividually searches for episodes one by one.
func (s *Service) searchEpisodesIndividually(ctx context.Context, episodes []*sqlc.Episode, series *sqlc.Series, source SearchSource) *BatchSearchResult {
	result := &BatchSearchResult{
		TotalSearched: len(episodes),
		Results:       make([]*SearchResult, 0, len(episodes)),
	}

	for _, ep := range episodes {
		item := s.episodeToSearchableItem(ctx, ep, series)
		searchResult, err := s.searchAndGrab(ctx, &item, source)
		if err != nil {
			s.logger.Warn().Err(err).Int64("episodeId", ep.ID).Msg("Failed to search episode")
			result.Failed++
			result.Results = append(result.Results, &SearchResult{Error: err.Error()})
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

	return result
}

func (s *Service) SearchSeason(ctx context.Context, seriesID int64, seasonNumber int, source SearchSource) (*BatchSearchResult, error) {
	series, err := s.getSeriesForSearch(ctx, seriesID)
	if err != nil {
		return nil, err
	}

	episodes, err := s.getMissingEpisodesForSeason(ctx, seriesID, seasonNumber)
	if err != nil {
		return s.tryDirectSeasonPackSearch(ctx, series, seasonNumber, source, err)
	}

	if len(episodes) == 0 {
		s.logger.Debug().
			Int64("seriesId", seriesID).
			Int("seasonNumber", seasonNumber).
			Msg("No missing episodes found for season")
		return &BatchSearchResult{}, nil
	}

	if s.isSeasonPackEligible(ctx, seriesID, seasonNumber) {
		if result, ok := s.trySeasonPackFirst(ctx, series, seasonNumber, source); ok {
			return result, nil
		}
		s.logger.Info().
			Int64("seriesId", seriesID).
			Int("seasonNumber", seasonNumber).
			Int("missingEpisodes", len(episodes)).
			Msg("No season pack found, falling back to individual episode search")
	}

	return s.searchEpisodesIndividually(ctx, episodes, series, source), nil
}

// searchSeasonPack searches for a season pack release (internal method).
func (s *Service) searchSeasonPack(ctx context.Context, series *sqlc.Series, seasonNumber int, source SearchSource) (*SearchResult, error) {
	item := s.seriesToSeasonPackItem(series, seasonNumber)
	return s.searchAndGrab(ctx, &item, source)
}

// SearchSeasonUpgrade searches for a season pack upgrade, falling back to individual episodes.
func (s *Service) SearchSeasonUpgrade(ctx context.Context, seriesID int64, seasonNumber, currentQualityID int, source SearchSource) (*BatchSearchResult, error) {
	series, err := s.getSeriesForSearch(ctx, seriesID)
	if err != nil {
		return nil, err
	}

	item := s.seriesToSeasonPackItem(series, seasonNumber)
	item.HasFile = true
	item.CurrentQualityID = currentQualityID

	packResult, err := s.searchAndGrab(ctx, &item, source)
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

	episodes, err := s.getUpgradableEpisodesForSeason(ctx, seriesID, seasonNumber)
	if err != nil || len(episodes) == 0 {
		result := &BatchSearchResult{TotalSearched: 1, Results: []*SearchResult{}}
		if packResult != nil {
			result.Results = append(result.Results, packResult)
		}
		return result, nil
	}

	return s.searchUpgradableEpisodesIndividually(ctx, episodes, source), nil
}

// searchUpgradableEpisodesIndividually searches upgradable episodes individually.
func (s *Service) searchUpgradableEpisodesIndividually(ctx context.Context, episodes []*sqlc.Episode, source SearchSource) *BatchSearchResult {
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

	return result
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
		if row.Status == statusUpgradable && row.Monitored == 1 {
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
// groupEpisodesBySeason groups episodes by season number.
func groupEpisodesBySeason(episodes []*sqlc.Episode) map[int64][]*sqlc.Episode {
	seasonEpisodes := make(map[int64][]*sqlc.Episode)
	for _, ep := range episodes {
		seasonEpisodes[ep.SeasonNumber] = append(seasonEpisodes[ep.SeasonNumber], ep)
	}
	return seasonEpisodes
}

// buildSeriesSearchItems builds search items with boxset prioritization.
func (s *Service) buildSeriesSearchItems(ctx context.Context, series *sqlc.Series, seasonEpisodes map[int64][]*sqlc.Episode) []SearchableItem {
	var items []SearchableItem

	for seasonNum, eps := range seasonEpisodes {
		if s.isSeasonPackEligible(ctx, series.ID, int(seasonNum)) {
			items = append(items, s.seriesToSeasonPackItem(series, int(seasonNum)))
		} else {
			for _, ep := range eps {
				items = append(items, s.episodeToSearchableItem(ctx, ep, series))
			}
		}
	}

	return items
}

// executeSeriesSearch searches all items and collects results.
func (s *Service) executeSeriesSearch(ctx context.Context, items []SearchableItem, source SearchSource) *BatchSearchResult {
	result := &BatchSearchResult{
		TotalSearched: len(items),
		Results:       make([]*SearchResult, 0, len(items)),
	}

	for i := range items {
		item := &items[i]
		searchResult, err := s.searchAndGrab(ctx, item, source)
		if err != nil {
			s.logger.Warn().Err(err).
				Str("mediaType", string(item.MediaType)).
				Int64("mediaId", item.MediaID).
				Msg("Failed to search item")
			result.Failed++
			result.Results = append(result.Results, &SearchResult{Error: err.Error()})
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

	return result
}

func (s *Service) SearchSeries(ctx context.Context, seriesID int64, source SearchSource) (*BatchSearchResult, error) {
	series, err := s.getSeriesForSearch(ctx, seriesID)
	if err != nil {
		return nil, err
	}

	episodes, err := s.getMissingEpisodesForSeries(ctx, seriesID)
	if err != nil {
		return nil, fmt.Errorf("failed to get missing episodes: %w", err)
	}

	seasonEpisodes := groupEpisodesBySeason(episodes)
	items := s.buildSeriesSearchItems(ctx, series, seasonEpisodes)

	return s.executeSeriesSearch(ctx, items, source), nil
}

// searchAndGrab is the core function that searches for a release and grabs the best one.
func (s *Service) searchAndGrab(_ctx context.Context, item *SearchableItem, source SearchSource) (*SearchResult, error) {
	searchKey := fmt.Sprintf("%s:%d", item.MediaType, item.MediaID)

	// Register this search and get a cancellable context
	searchCtx, cancel := s.registerSearch(searchKey)
	defer s.unregisterSearch(searchKey, cancel)

	// Broadcast search started
	s.broadcastStarted(item, source)

	s.logger.Info().
		Str("mediaType", string(item.MediaType)).
		Int64("mediaId", item.MediaID).
		Str("title", item.Title).
		Msg("Starting automatic search")

	// Get quality profile
	profile, err := s.qualityService.Get(searchCtx, item.QualityProfileID)
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

	// Execute search and select best release
	bestRelease, earlyResult, err := s.findBestRelease(searchCtx, item, &criteria, &scoringParams, profile, source)
	if err != nil {
		return nil, err
	}
	if earlyResult != nil {
		return earlyResult, nil
	}

	isUpgrade := item.HasFile && item.CurrentQualityID > 0

	// Acquire grab lock to prevent concurrent grabs from RSS sync
	if s.grabLock != nil {
		lockKey := decisioning.Key(item.MediaType, item.MediaID)
		if !s.grabLock.TryAcquire(lockKey) {
			s.logger.Debug().Str("key", lockKey).Msg("skipping: grab lock held")
			result := &SearchResult{Found: true}
			s.broadcastCompleted(item, result)
			return result, nil
		}
		defer s.grabLock.Release(lockKey)
	}

	return s.grabAndReport(searchCtx, item, bestRelease, source, isUpgrade)
}

func (s *Service) findBestRelease(ctx context.Context, item *SearchableItem, criteria *types.SearchCriteria, scoringParams *search.ScoredSearchParams, profile *quality.Profile, source SearchSource) (*types.TorrentInfo, *SearchResult, error) {
	searchResult, err := s.searchService.SearchTorrents(ctx, criteria, scoringParams)
	if err != nil {
		s.broadcastFailed(item, err.Error())
		s.logAutoSearchFailed(ctx, item, source, err.Error())
		return nil, nil, fmt.Errorf("search failed: %w", err)
	}

	if len(searchResult.Releases) == 0 {
		s.logger.Debug().Str("title", item.Title).Msg("No releases found")
		result := &SearchResult{Found: false}
		s.broadcastCompleted(item, result)
		return nil, result, nil
	}

	bestRelease := s.selectBestRelease(searchResult.Releases, profile, item)
	if bestRelease == nil {
		s.logger.Debug().Str("title", item.Title).Msg("No acceptable releases found")
		result := &SearchResult{Found: false}
		s.broadcastCompleted(item, result)
		return nil, result, nil
	}

	s.logger.Info().
		Str("title", item.Title).
		Str("release", bestRelease.Title).
		Float64("score", bestRelease.Score).
		Int("normalizedScore", bestRelease.NormalizedScore).
		Msg("Selected best release")

	return bestRelease, nil, nil
}

func (s *Service) grabAndReport(ctx context.Context, item *SearchableItem, bestRelease *types.TorrentInfo, source SearchSource, isUpgrade bool) (*SearchResult, error) {
	grabReq := s.buildGrabRequest(item, bestRelease)
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

func (s *Service) buildGrabRequest(item *SearchableItem, bestRelease *types.TorrentInfo) *grab.GrabRequest {
	req := &grab.GrabRequest{
		Release:      &bestRelease.ReleaseInfo,
		MediaType:    string(item.MediaType),
		MediaID:      item.MediaID,
		SeriesID:     item.SeriesID,
		SeasonNumber: item.SeasonNumber,
		TargetSlotID: item.TargetSlotID,
		Source:       "auto-search",
	}
	if item.MediaType == MediaTypeSeason {
		req.IsSeasonPack = true
		parsed := scanner.ParseFilename(bestRelease.Title)
		if parsed.IsCompleteSeries {
			req.IsCompleteSeries = true
		}
	}
	return req
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
func (s *Service) buildSearchCriteria(item *SearchableItem) types.SearchCriteria {
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
// Delegates to the shared decisioning.SelectBestRelease.
func (s *Service) selectBestRelease(releases []types.TorrentInfo, profile *quality.Profile, item *SearchableItem) *types.TorrentInfo {
	return decisioning.SelectBestRelease(releases, profile, item, s.logger)
}

func (s *Service) movieToSearchableItem(ctx context.Context, movie *sqlc.Movie) SearchableItem {
	return decisioning.MovieToSearchableItem(ctx, s.queries, s.logger, movie)
}

func (s *Service) episodeToSearchableItem(ctx context.Context, episode *sqlc.Episode, series *sqlc.Series) SearchableItem {
	return decisioning.EpisodeToSearchableItem(ctx, s.queries, s.logger, episode, series)
}

// seriesToSeasonPackItem converts a series and season number to a season pack SearchableItem.
func (s *Service) seriesToSeasonPackItem(series *sqlc.Series, seasonNumber int) SearchableItem {
	return decisioning.SeriesToSeasonPackItem(series, seasonNumber)
}

// movieUpgradeCandidateToSearchableItem converts an upgrade candidate movie to a SearchableItem.
func (s *Service) movieUpgradeCandidateToSearchableItem(movie *sqlc.ListMovieUpgradeCandidatesRow) SearchableItem {
	return decisioning.MovieUpgradeCandidateToSearchableItem(movie)
}

// episodeUpgradeCandidateToSearchableItem converts an upgrade candidate row to a SearchableItem.
func (s *Service) episodeUpgradeCandidateToSearchableItem(row *sqlc.ListEpisodeUpgradeCandidatesRow) SearchableItem {
	return decisioning.EpisodeUpgradeCandidateToSearchableItem(row)
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
		if row.Status == statusMissing && row.Monitored == 1 {
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
		if row.Status == statusMissing && row.Monitored == 1 {
			missing = append(missing, row)
		}
	}

	return missing, nil
}

// isSeasonPackEligible checks if a season pack search should be used.
func (s *Service) isSeasonPackEligible(ctx context.Context, seriesID int64, seasonNumber int) bool {
	return decisioning.IsSeasonPackEligible(ctx, s.queries, s.logger, seriesID, seasonNumber)
}

func (s *Service) isSeasonPackUpgradeEligible(ctx context.Context, seriesID int64, seasonNumber int) bool {
	return decisioning.IsSeasonPackUpgradeEligible(ctx, s.queries, s.logger, seriesID, seasonNumber)
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

	newStatus := statusMissing
	if fileCount > 0 {
		newStatus = statusUpgradable
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
		s.broadcaster.Broadcast("movie:updated", map[string]any{"movieId": movieID})
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

	newStatus := statusMissing
	if fileCount > 0 {
		newStatus = statusUpgradable
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
		s.broadcaster.Broadcast("series:updated", map[string]any{"id": episode.SeriesID})
	}

	return &RetryResult{
		NewStatus: newStatus,
		Message:   fmt.Sprintf("episode reset to %s", newStatus),
	}, nil
}

// Broadcast helpers

func (s *Service) broadcastStarted(item *SearchableItem, source SearchSource) {
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

func (s *Service) broadcastCompleted(item *SearchableItem, result *SearchResult) {
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

func (s *Service) broadcastFailed(item *SearchableItem, errMsg string) {
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

func (s *Service) logAutoSearchSuccess(ctx context.Context, item *SearchableItem, source SearchSource, release *types.TorrentInfo, grabResult *grab.GrabResult, isUpgrade bool) {
	if s.historyService == nil {
		return
	}

	var mediaType history.MediaType
	switch item.MediaType {
	case MediaTypeSeason, MediaTypeSeries:
		mediaType = history.MediaTypeSeason
	case MediaTypeEpisode:
		mediaType = history.MediaTypeEpisode
	default:
		mediaType = history.MediaTypeMovie
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
	if err := s.historyService.LogAutoSearchDownload(ctx, mediaType, item.MediaID, qualityStr, &data); err != nil {
		s.logger.Warn().Err(err).Msg("Failed to log autosearch download event")
	}
}

func (s *Service) logAutoSearchFailed(ctx context.Context, item *SearchableItem, source SearchSource, errMsg string) {
	if s.historyService == nil {
		return
	}

	var mediaType history.MediaType
	switch item.MediaType {
	case MediaTypeSeason, MediaTypeSeries:
		mediaType = history.MediaTypeSeason
	case MediaTypeEpisode:
		mediaType = history.MediaTypeEpisode
	default:
		mediaType = history.MediaTypeMovie
	}

	if err := s.historyService.LogAutoSearchFailed(ctx, mediaType, item.MediaID, history.AutoSearchFailedData{
		Error:  errMsg,
		Source: string(source),
	}); err != nil {
		s.logger.Warn().Err(err).Msg("Failed to log autosearch failed event")
	}
}
