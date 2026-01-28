// Package search provides search orchestration across multiple indexers.
package search

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/indexer"
	"github.com/slipstream/slipstream/internal/indexer/ratelimit"
	"github.com/slipstream/slipstream/internal/indexer/scoring"
	"github.com/slipstream/slipstream/internal/indexer/status"
	"github.com/slipstream/slipstream/internal/indexer/types"
	"github.com/slipstream/slipstream/internal/library/quality"
)

// Broadcaster interface for sending events to clients.
type Broadcaster interface {
	Broadcast(msgType string, payload interface{}) error
}

// Service orchestrates searches across multiple indexers.
type Service struct {
	indexerService *indexer.Service
	statusService  *status.Service
	rateLimiter    *ratelimit.Limiter
	broadcaster    Broadcaster
	logger         zerolog.Logger
}

// NewService creates a new search service.
func NewService(indexerService *indexer.Service, logger zerolog.Logger) *Service {
	return &Service{
		indexerService: indexerService,
		logger:         logger.With().Str("component", "search").Logger(),
	}
}

// SetStatusService sets the status service for tracking indexer health.
func (s *Service) SetStatusService(statusService *status.Service) {
	s.statusService = statusService
}

// SetRateLimiter sets the rate limiter for controlling query rates.
func (s *Service) SetRateLimiter(limiter *ratelimit.Limiter) {
	s.rateLimiter = limiter
}

// SetBroadcaster sets the WebSocket broadcaster for real-time events.
func (s *Service) SetBroadcaster(broadcaster Broadcaster) {
	s.broadcaster = broadcaster
}

// SearchResult contains aggregated search results.
type SearchResult struct {
	Releases      []types.ReleaseInfo  `json:"releases"`
	TotalResults  int                  `json:"total"`
	IndexersUsed  int                  `json:"indexersSearched"`
	IndexerErrors []SearchIndexerError `json:"errors,omitempty"`
}

// TorrentSearchResult contains aggregated torrent search results.
type TorrentSearchResult struct {
	Releases      []types.TorrentInfo  `json:"releases"`
	TotalResults  int                  `json:"total"`
	IndexersUsed  int                  `json:"indexersSearched"`
	IndexerErrors []SearchIndexerError `json:"errors,omitempty"`
}

// SearchIndexerError represents an error from a specific indexer during search.
type SearchIndexerError struct {
	IndexerID   int64  `json:"indexerId"`
	IndexerName string `json:"indexerName"`
	Error       string `json:"error"`
}

// searchTaskResult represents the result from a single indexer search.
type searchTaskResult struct {
	IndexerID   int64
	IndexerName string
	Releases    []types.ReleaseInfo
	Torrents    []types.TorrentInfo
	Error       error
}

// Search executes a search across all enabled indexers.
func (s *Service) Search(ctx context.Context, criteria types.SearchCriteria) (*SearchResult, error) {
	startTime := time.Now()

	// Get enabled indexers that support the search type
	indexers, err := s.getIndexersForSearch(ctx, criteria)
	if err != nil {
		return nil, fmt.Errorf("failed to get indexers: %w", err)
	}

	if len(indexers) == 0 {
		return &SearchResult{
			Releases:     []types.ReleaseInfo{},
			TotalResults: 0,
			IndexersUsed: 0,
		}, nil
	}

	// Broadcast search started event
	indexerIDs := make([]int64, len(indexers))
	for i, idx := range indexers {
		indexerIDs[i] = idx.ID
	}
	s.broadcastSearchStarted(criteria, indexerIDs)

	s.logger.Info().
		Int("indexerCount", len(indexers)).
		Str("query", criteria.Query).
		Str("type", criteria.Type).
		Msg("Starting search across indexers")

	// Dispatch parallel searches
	result := s.dispatchSearches(ctx, indexers, criteria)

	elapsed := time.Since(startTime)

	// Broadcast search completed event
	s.broadcastSearchCompleted(criteria, result, elapsed)

	s.logger.Info().
		Int("totalResults", result.TotalResults).
		Int("indexersUsed", result.IndexersUsed).
		Int("errors", len(result.IndexerErrors)).
		Msg("Search completed")

	return result, nil
}

// broadcastSearchStarted sends a search started event.
func (s *Service) broadcastSearchStarted(criteria types.SearchCriteria, indexerIDs []int64) {
	if s.broadcaster == nil {
		return
	}
	s.broadcaster.Broadcast(indexer.EventSearchStarted, indexer.SearchStartedPayload{
		Query:      criteria.Query,
		Type:       criteria.Type,
		IndexerIDs: indexerIDs,
	})
}

// broadcastSearchCompleted sends a search completed event.
func (s *Service) broadcastSearchCompleted(criteria types.SearchCriteria, result *SearchResult, elapsed time.Duration) {
	if s.broadcaster == nil {
		return
	}
	errors := make([]string, len(result.IndexerErrors))
	for i, e := range result.IndexerErrors {
		errors[i] = e.Error
	}
	s.broadcaster.Broadcast(indexer.EventSearchCompleted, indexer.SearchCompletedPayload{
		Query:        criteria.Query,
		Type:         criteria.Type,
		TotalResults: result.TotalResults,
		IndexersUsed: result.IndexersUsed,
		Errors:       errors,
		ElapsedMs:    elapsed.Milliseconds(),
	})
}

// searchTorrentsInternal executes a search across all enabled torrent indexers and returns torrent-specific results.
// This is an internal method; use SearchTorrents for scored results.
func (s *Service) searchTorrentsInternal(ctx context.Context, criteria types.SearchCriteria) (*TorrentSearchResult, error) {
	// Get enabled torrent indexers
	indexers, err := s.indexerService.ListEnabledByProtocol(ctx, indexer.ProtocolTorrent)
	if err != nil {
		return nil, fmt.Errorf("failed to get indexers: %w", err)
	}

	// Filter by search type support
	indexers = s.filterBySearchSupport(indexers, criteria)

	if len(indexers) == 0 {
		return &TorrentSearchResult{
			Releases:     []types.TorrentInfo{},
			TotalResults: 0,
			IndexersUsed: 0,
		}, nil
	}

	s.logger.Info().
		Int("indexerCount", len(indexers)).
		Str("query", criteria.Query).
		Str("type", criteria.Type).
		Int("season", criteria.Season).
		Int("episode", criteria.Episode).
		Int("tvdbId", criteria.TvdbID).
		Msg("Starting torrent search across indexers")

	// Dispatch parallel searches
	result := s.dispatchTorrentSearches(ctx, indexers, criteria)

	s.logger.Info().
		Int("totalResults", result.TotalResults).
		Int("indexersUsed", result.IndexersUsed).
		Int("errors", len(result.IndexerErrors)).
		Msg("Torrent search completed")

	return result, nil
}

// SearchMovies searches for movie releases.
func (s *Service) SearchMovies(ctx context.Context, criteria types.SearchCriteria) (*SearchResult, error) {
	criteria.Type = "movie"
	if len(criteria.Categories) == 0 {
		criteria.Categories = indexer.MovieCategories()
	}
	return s.Search(ctx, criteria)
}

// SearchTV searches for TV releases.
func (s *Service) SearchTV(ctx context.Context, criteria types.SearchCriteria) (*SearchResult, error) {
	criteria.Type = "tvsearch"
	if len(criteria.Categories) == 0 {
		criteria.Categories = indexer.TVCategories()
	}
	return s.Search(ctx, criteria)
}

// getIndexersForSearch returns indexers appropriate for the search criteria.
func (s *Service) getIndexersForSearch(ctx context.Context, criteria types.SearchCriteria) ([]*types.IndexerDefinition, error) {
	var indexers []*types.IndexerDefinition
	var err error

	// Get indexers based on search type
	switch criteria.Type {
	case "movie":
		indexers, err = s.indexerService.ListEnabledForMovies(ctx)
	case "tvsearch":
		indexers, err = s.indexerService.ListEnabledForTV(ctx)
	default:
		indexers, err = s.indexerService.ListEnabled(ctx)
	}

	if err != nil {
		return nil, err
	}

	// Filter by search support
	indexers = s.filterBySearchSupport(indexers, criteria)

	// Filter out disabled indexers (due to failures)
	indexers = s.filterDisabledIndexers(ctx, indexers)

	// Filter out rate-limited indexers
	indexers = s.filterRateLimitedIndexers(ctx, indexers)

	// Sort by priority (lower number = higher priority)
	sort.Slice(indexers, func(i, j int) bool {
		return indexers[i].Priority < indexers[j].Priority
	})

	return indexers, nil
}

// filterDisabledIndexers removes indexers that are temporarily disabled due to failures.
func (s *Service) filterDisabledIndexers(ctx context.Context, indexers []*types.IndexerDefinition) []*types.IndexerDefinition {
	if s.statusService == nil {
		return indexers
	}

	filtered := make([]*types.IndexerDefinition, 0, len(indexers))
	for _, idx := range indexers {
		disabled, disabledTill, err := s.statusService.IsDisabled(ctx, idx.ID)
		if err != nil {
			s.logger.Warn().Err(err).Int64("indexerId", idx.ID).Msg("Failed to check indexer status")
			filtered = append(filtered, idx)
			continue
		}

		if disabled {
			s.logger.Debug().
				Int64("indexerId", idx.ID).
				Str("indexerName", idx.Name).
				Time("disabledTill", *disabledTill).
				Msg("Skipping disabled indexer")
			continue
		}

		filtered = append(filtered, idx)
	}

	return filtered
}

// filterRateLimitedIndexers removes indexers that have reached their rate limit.
func (s *Service) filterRateLimitedIndexers(ctx context.Context, indexers []*types.IndexerDefinition) []*types.IndexerDefinition {
	if s.rateLimiter == nil {
		return indexers
	}

	filtered := make([]*types.IndexerDefinition, 0, len(indexers))
	for _, idx := range indexers {
		limited, err := s.rateLimiter.CheckQueryLimit(ctx, idx.ID)
		if err != nil {
			s.logger.Warn().Err(err).Int64("indexerId", idx.ID).Msg("Failed to check rate limit")
			filtered = append(filtered, idx)
			continue
		}

		if limited {
			s.logger.Debug().
				Int64("indexerId", idx.ID).
				Str("indexerName", idx.Name).
				Msg("Skipping rate-limited indexer")
			continue
		}

		filtered = append(filtered, idx)
	}

	return filtered
}

// filterBySearchSupport filters indexers to those that support the requested search type.
func (s *Service) filterBySearchSupport(indexers []*types.IndexerDefinition, criteria types.SearchCriteria) []*types.IndexerDefinition {
	filtered := make([]*types.IndexerDefinition, 0, len(indexers))

	for _, idx := range indexers {
		// Must support search
		if !idx.SupportsSearch {
			continue
		}

		// Filter by search type
		switch criteria.Type {
		case "movie":
			if !idx.SupportsMovies {
				continue
			}
		case "tvsearch":
			if !idx.SupportsTV {
				continue
			}
		}

		filtered = append(filtered, idx)
	}

	return filtered
}

// dispatchSearches runs searches in parallel across indexers.
func (s *Service) dispatchSearches(ctx context.Context, indexers []*types.IndexerDefinition, criteria types.SearchCriteria) *SearchResult {
	var wg sync.WaitGroup
	resultsChan := make(chan searchTaskResult, len(indexers))

	// Set timeout for individual indexer searches
	searchCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	for _, idx := range indexers {
		wg.Add(1)
		go func(def *types.IndexerDefinition) {
			defer wg.Done()
			result := s.searchIndexer(searchCtx, def, criteria)
			resultsChan <- result
		}(idx)
	}

	// Wait and collect results
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	return s.aggregateResults(resultsChan)
}

// dispatchTorrentSearches runs torrent searches in parallel across indexers.
func (s *Service) dispatchTorrentSearches(ctx context.Context, indexers []*types.IndexerDefinition, criteria types.SearchCriteria) *TorrentSearchResult {
	var wg sync.WaitGroup
	resultsChan := make(chan searchTaskResult, len(indexers))

	// Set timeout for individual indexer searches
	searchCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	for _, idx := range indexers {
		wg.Add(1)
		go func(def *types.IndexerDefinition) {
			defer wg.Done()
			result := s.searchIndexerTorrents(searchCtx, def, criteria)
			resultsChan <- result
		}(idx)
	}

	// Wait and collect results
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	return s.aggregateTorrentResults(resultsChan, &criteria)
}

// searchIndexer performs a search on a single indexer.
func (s *Service) searchIndexer(ctx context.Context, def *types.IndexerDefinition, criteria types.SearchCriteria) searchTaskResult {
	result := searchTaskResult{
		IndexerID:   def.ID,
		IndexerName: def.Name,
	}

	// Record the query in rate limiter
	if s.rateLimiter != nil {
		s.rateLimiter.RecordQuery(ctx, def.ID)
	}

	// Get the client from the indexer service
	client, err := s.indexerService.GetClient(ctx, def.ID)
	if err != nil {
		result.Error = fmt.Errorf("failed to get client: %w", err)
		s.logger.Error().
			Err(err).
			Int64("indexerId", def.ID).
			Str("indexerName", def.Name).
			Msg("Failed to get indexer client")
		s.recordFailure(ctx, def.ID, err)
		return result
	}

	// Perform the search
	start := time.Now()
	releases, err := client.Search(ctx, criteria)
	elapsed := time.Since(start)

	if err != nil {
		result.Error = fmt.Errorf("search failed: %w", err)
		s.logger.Error().
			Err(err).
			Int64("indexerId", def.ID).
			Str("indexerName", def.Name).
			Dur("elapsed", elapsed).
			Msg("Search failed")
		s.recordFailure(ctx, def.ID, err)
		return result
	}

	// Record success
	s.recordSuccess(ctx, def.ID)

	result.Releases = releases

	s.logger.Debug().
		Int64("indexerId", def.ID).
		Str("indexerName", def.Name).
		Int("results", len(releases)).
		Dur("elapsed", elapsed).
		Msg("Search completed for indexer")

	return result
}

// recordSuccess records a successful operation for an indexer.
func (s *Service) recordSuccess(ctx context.Context, indexerID int64) {
	if s.statusService == nil {
		return
	}
	if err := s.statusService.RecordSuccess(ctx, indexerID); err != nil {
		s.logger.Warn().Err(err).Int64("indexerId", indexerID).Msg("Failed to record success")
	}
}

// recordFailure records a failed operation for an indexer.
func (s *Service) recordFailure(ctx context.Context, indexerID int64, opError error) {
	if s.statusService == nil {
		return
	}
	if err := s.statusService.RecordFailure(ctx, indexerID, opError); err != nil {
		s.logger.Warn().Err(err).Int64("indexerId", indexerID).Msg("Failed to record failure")
	}
}

// searchIndexerTorrents performs a torrent search on a single indexer.
func (s *Service) searchIndexerTorrents(ctx context.Context, def *types.IndexerDefinition, criteria types.SearchCriteria) searchTaskResult {
	result := searchTaskResult{
		IndexerID:   def.ID,
		IndexerName: def.Name,
	}

	// Record the query in rate limiter
	if s.rateLimiter != nil {
		s.rateLimiter.RecordQuery(ctx, def.ID)
	}

	// Only support torrent protocol for torrent searches
	if def.Protocol != types.ProtocolTorrent {
		result.Error = fmt.Errorf("indexer does not support torrent search")
		return result
	}

	// Get the client from the indexer service
	client, err := s.indexerService.GetClient(ctx, def.ID)
	if err != nil {
		result.Error = fmt.Errorf("failed to get client: %w", err)
		s.logger.Error().
			Err(err).
			Int64("indexerId", def.ID).
			Str("indexerName", def.Name).
			Msg("Failed to get indexer client")
		s.recordFailure(ctx, def.ID, err)
		return result
	}

	// Check if client supports torrent-specific search
	torrentClient, ok := client.(indexer.TorrentIndexer)
	if !ok {
		result.Error = fmt.Errorf("indexer does not support torrent search")
		return result
	}

	// Perform the search
	start := time.Now()
	torrents, err := torrentClient.SearchTorrents(ctx, criteria)
	elapsed := time.Since(start)

	if err != nil {
		result.Error = fmt.Errorf("search failed: %w", err)
		s.logger.Error().
			Err(err).
			Int64("indexerId", def.ID).
			Str("indexerName", def.Name).
			Dur("elapsed", elapsed).
			Msg("Torrent search failed")
		s.recordFailure(ctx, def.ID, err)
		return result
	}

	// Record success
	s.recordSuccess(ctx, def.ID)

	result.Torrents = torrents

	s.logger.Debug().
		Int64("indexerId", def.ID).
		Str("indexerName", def.Name).
		Int("results", len(torrents)).
		Dur("elapsed", elapsed).
		Msg("Torrent search completed for indexer")

	return result
}

// ScoredSearchParams contains parameters for scored search operations.
type ScoredSearchParams struct {
	QualityProfile *quality.Profile
	SearchYear     int // Expected year for movies
	SearchSeason   int // Expected season for TV
	SearchEpisode  int // Expected episode for TV
}

// SearchTorrents performs a torrent search and returns results scored by desirability.
// Results are sorted by score descending (highest score first).
// All torrent searches include scoring - unscored searches are not supported.
func (s *Service) SearchTorrents(ctx context.Context, criteria types.SearchCriteria, params ScoredSearchParams) (*TorrentSearchResult, error) {
	// First, perform the internal search
	result, err := s.searchTorrentsInternal(ctx, criteria)
	if err != nil {
		return nil, err
	}

	if len(result.Releases) == 0 {
		return result, nil
	}

	// Build indexer priority map
	indexerPriorities, err := s.getIndexerPriorities(ctx)
	if err != nil {
		s.logger.Warn().Err(err).Msg("Failed to get indexer priorities, using defaults")
		indexerPriorities = make(map[int64]int)
	}

	// Create scoring context
	scoringCtx := scoring.ScoringContext{
		QualityProfile:    params.QualityProfile,
		SearchYear:        params.SearchYear,
		SearchSeason:      params.SearchSeason,
		SearchEpisode:     params.SearchEpisode,
		IndexerPriorities: indexerPriorities,
		Now:               time.Now(),
	}

	// Score and sort torrents
	scorer := scoring.NewDefaultScorer()
	scorer.ScoreTorrents(result.Releases, scoringCtx)

	s.logger.Debug().
		Int("totalResults", len(result.Releases)).
		Msg("Scored and sorted torrent search results")

	return result, nil
}

// getIndexerPriorities returns a map of indexer ID to priority.
func (s *Service) getIndexerPriorities(ctx context.Context) (map[int64]int, error) {
	indexers, err := s.indexerService.ListEnabled(ctx)
	if err != nil {
		return nil, err
	}

	priorities := make(map[int64]int, len(indexers))
	for _, idx := range indexers {
		priorities[idx.ID] = idx.Priority
	}
	return priorities, nil
}

