package autosearch

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/config"
	"github.com/slipstream/slipstream/internal/database/sqlc"
)

// ScheduledSearcher handles scheduled automatic searches for missing items.
type ScheduledSearcher struct {
	service *Service
	config  *config.AutoSearchConfig
	logger  zerolog.Logger

	// Rate limiting
	rateLimiter *RateLimiter

	// Task state
	mu      sync.Mutex
	running bool
}

// NewScheduledSearcher creates a new scheduled searcher.
func NewScheduledSearcher(service *Service, cfg *config.AutoSearchConfig, logger zerolog.Logger) *ScheduledSearcher {
	return &ScheduledSearcher{
		service: service,
		config:  cfg,
		logger:  logger.With().Str("component", "scheduled-search").Logger(),
		rateLimiter: &RateLimiter{
			baseDelayMs: cfg.BaseDelayMs,
		},
	}
}

// IsRunning returns true if the scheduled search is currently running.
func (s *ScheduledSearcher) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

// Run executes the scheduled search task. Returns error if task is already running.
func (s *ScheduledSearcher) Run(ctx context.Context) error {
	// Check and set running state
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		s.logger.Debug().Msg("Scheduled search task already running, skipping")
		return nil
	}
	s.running = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
	}()

	startTime := time.Now()
	s.logger.Info().Msg("Starting scheduled search task")

	// Collect all searchable items
	items, err := s.collectSearchableItems(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to collect searchable items")
		return err
	}

	if len(items) == 0 {
		s.logger.Info().Msg("No missing items to search")
		return nil
	}

	s.logger.Info().Int("itemCount", len(items)).Msg("Collected searchable items")

	// Broadcast task started
	s.broadcastTaskStarted(len(items))

	// Process items
	result := s.processItems(ctx, items)

	elapsed := time.Since(startTime)
	s.logger.Info().
		Int("searched", result.TotalSearched).
		Int("found", result.Found).
		Int("downloaded", result.Downloaded).
		Int("failed", result.Failed).
		Dur("elapsed", elapsed).
		Msg("Scheduled search task completed")

	// Broadcast task completed
	s.broadcastTaskCompleted(result, elapsed)

	return nil
}

// RunMoviesOnly executes search for missing movies only.
func (s *ScheduledSearcher) RunMoviesOnly(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		s.logger.Debug().Msg("Scheduled search task already running, skipping movies-only")
		return nil
	}
	s.running = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
	}()

	startTime := time.Now()
	s.logger.Info().Msg("Starting search task for missing movies only")

	// Collect only missing movies
	movieItems, err := s.collectMissingMovies(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to collect missing movies")
		return err
	}

	// Sort by release date (newest first)
	sort.Slice(movieItems, func(i, j int) bool {
		return movieItems[i].releaseDate.After(movieItems[j].releaseDate)
	})

	items := make([]SearchableItem, len(movieItems))
	for i, item := range movieItems {
		items[i] = item.item
	}

	if len(items) == 0 {
		s.logger.Info().Msg("No missing movies to search")
		return nil
	}

	s.logger.Info().Int("itemCount", len(items)).Msg("Collected missing movies")
	s.broadcastTaskStarted(len(items))

	result := s.processItems(ctx, items)

	elapsed := time.Since(startTime)
	s.logger.Info().
		Int("searched", result.TotalSearched).
		Int("found", result.Found).
		Int("downloaded", result.Downloaded).
		Int("failed", result.Failed).
		Dur("elapsed", elapsed).
		Msg("Movies search task completed")

	s.broadcastTaskCompleted(result, elapsed)
	return nil
}

// RunSeriesOnly executes search for missing series episodes only.
func (s *ScheduledSearcher) RunSeriesOnly(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		s.logger.Debug().Msg("Scheduled search task already running, skipping series-only")
		return nil
	}
	s.running = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
	}()

	startTime := time.Now()
	s.logger.Info().Msg("Starting search task for missing series only")

	// Collect only missing episodes
	episodeItems, err := s.collectMissingEpisodes(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to collect missing episodes")
		return err
	}

	// Sort by release date (newest first)
	sort.Slice(episodeItems, func(i, j int) bool {
		return episodeItems[i].releaseDate.After(episodeItems[j].releaseDate)
	})

	items := make([]SearchableItem, len(episodeItems))
	for i, item := range episodeItems {
		items[i] = item.item
	}

	if len(items) == 0 {
		s.logger.Info().Msg("No missing episodes to search")
		return nil
	}

	s.logger.Info().Int("itemCount", len(items)).Msg("Collected missing episodes")
	s.broadcastTaskStarted(len(items))

	result := s.processItems(ctx, items)

	elapsed := time.Since(startTime)
	s.logger.Info().
		Int("searched", result.TotalSearched).
		Int("found", result.Found).
		Int("downloaded", result.Downloaded).
		Int("failed", result.Failed).
		Dur("elapsed", elapsed).
		Msg("Series search task completed")

	s.broadcastTaskCompleted(result, elapsed)
	return nil
}

// searchableItemWithPriority wraps a searchable item with sorting metadata.
type searchableItemWithPriority struct {
	item        SearchableItem
	releaseDate time.Time
	isSeasonPack bool
}

// collectSearchableItems gathers all missing movies and episodes, ordered by release date.
func (s *ScheduledSearcher) collectSearchableItems(ctx context.Context) ([]SearchableItem, error) {
	var items []searchableItemWithPriority

	// Get missing movies
	movies, err := s.collectMissingMovies(ctx)
	if err != nil {
		return nil, err
	}
	items = append(items, movies...)

	// Get missing episodes (with boxset prioritization)
	episodes, err := s.collectMissingEpisodes(ctx)
	if err != nil {
		return nil, err
	}
	items = append(items, episodes...)

	// Sort by release date (newest first)
	sort.Slice(items, func(i, j int) bool {
		return items[i].releaseDate.After(items[j].releaseDate)
	})

	// Extract just the searchable items
	result := make([]SearchableItem, len(items))
	for i, item := range items {
		result[i] = item.item
	}

	return result, nil
}

// collectMissingMovies collects missing movies that are released and monitored.
func (s *ScheduledSearcher) collectMissingMovies(ctx context.Context) ([]searchableItemWithPriority, error) {
	rows, err := s.service.queries.ListMissingMovies(ctx)
	if err != nil {
		return nil, err
	}

	items := make([]searchableItemWithPriority, 0, len(rows))
	for _, row := range rows {
		// Check backoff status
		shouldSkip, err := s.shouldSkipItem(ctx, "movie", row.ID)
		if err != nil {
			s.logger.Warn().Err(err).Int64("movieId", row.ID).Msg("Failed to check backoff status")
		}
		if shouldSkip {
			continue
		}

		item := s.service.movieToSearchableItem(row)

		var releaseDate time.Time
		if row.PhysicalReleaseDate.Valid {
			releaseDate = row.PhysicalReleaseDate.Time
		} else if row.ReleaseDate.Valid {
			releaseDate = row.ReleaseDate.Time
		}

		items = append(items, searchableItemWithPriority{
			item:        item,
			releaseDate: releaseDate,
		})
	}

	return items, nil
}

// collectMissingEpisodes collects missing episodes with boxset prioritization.
func (s *ScheduledSearcher) collectMissingEpisodes(ctx context.Context) ([]searchableItemWithPriority, error) {
	rows, err := s.service.queries.ListMissingEpisodes(ctx)
	if err != nil {
		return nil, err
	}

	// Group episodes by series and season for boxset prioritization
	type seasonKey struct {
		seriesID     int64
		seasonNumber int64
	}
	seasonEpisodes := make(map[seasonKey][]*sqlc.ListMissingEpisodesRow)

	for _, row := range rows {
		key := seasonKey{seriesID: row.SeriesID, seasonNumber: row.SeasonNumber}
		seasonEpisodes[key] = append(seasonEpisodes[key], row)
	}

	// Determine which items to search
	var items []searchableItemWithPriority

	for key, episodes := range seasonEpisodes {
		// Check if entire season is missing and released (for season pack search)
		allReleasedAndMissing := s.service.isSeasonPackEligible(ctx, key.seriesID, int(key.seasonNumber))

		if allReleasedAndMissing {
			// Check backoff for the first episode as proxy for season pack
			shouldSkip, err := s.shouldSkipItem(ctx, "episode", episodes[0].ID)
			if err != nil {
				s.logger.Warn().Err(err).Int64("seriesId", key.seriesID).Int64("season", key.seasonNumber).Msg("Failed to check backoff status for season")
			}
			if shouldSkip {
				continue
			}

			// Add as season pack search (use first episode's data for metadata)
			firstEp := episodes[0]
			var airDate time.Time
			if firstEp.AirDate.Valid {
				airDate = firstEp.AirDate.Time
			}

			item := SearchableItem{
				MediaType:     MediaTypeSeason,
				MediaID:       key.seriesID,
				Title:         firstEp.SeriesTitle,
				SeasonNumber:  int(key.seasonNumber),
			}
			if firstEp.SeriesYear.Valid {
				item.Year = int(firstEp.SeriesYear.Int64)
			}
			if firstEp.SeriesTvdbID.Valid {
				item.TvdbID = int(firstEp.SeriesTvdbID.Int64)
			}
			if firstEp.SeriesTmdbID.Valid {
				item.TmdbID = int(firstEp.SeriesTmdbID.Int64)
			}
			if firstEp.SeriesImdbID.Valid {
				item.ImdbID = firstEp.SeriesImdbID.String
			}
			if firstEp.SeriesQualityProfileID.Valid {
				item.QualityProfileID = firstEp.SeriesQualityProfileID.Int64
			}

			items = append(items, searchableItemWithPriority{
				item:         item,
				releaseDate:  airDate,
				isSeasonPack: true,
			})
		} else {
			// Add individual episodes
			for _, ep := range episodes {
				// Check backoff for each episode
				shouldSkip, err := s.shouldSkipItem(ctx, "episode", ep.ID)
				if err != nil {
					s.logger.Warn().Err(err).Int64("episodeId", ep.ID).Msg("Failed to check backoff status")
				}
				if shouldSkip {
					continue
				}

				var airDate time.Time
				if ep.AirDate.Valid {
					airDate = ep.AirDate.Time
				}

				item := SearchableItem{
					MediaType:     MediaTypeEpisode,
					MediaID:       ep.ID,
					Title:         ep.SeriesTitle,
					SeasonNumber:  int(ep.SeasonNumber),
					EpisodeNumber: int(ep.EpisodeNumber),
				}
				if ep.SeriesYear.Valid {
					item.Year = int(ep.SeriesYear.Int64)
				}
				if ep.SeriesTvdbID.Valid {
					item.TvdbID = int(ep.SeriesTvdbID.Int64)
				}
				if ep.SeriesTmdbID.Valid {
					item.TmdbID = int(ep.SeriesTmdbID.Int64)
				}
				if ep.SeriesImdbID.Valid {
					item.ImdbID = ep.SeriesImdbID.String
				}
				if ep.SeriesQualityProfileID.Valid {
					item.QualityProfileID = ep.SeriesQualityProfileID.Int64
				}

				items = append(items, searchableItemWithPriority{
					item:        item,
					releaseDate: airDate,
				})
			}
		}
	}

	return items, nil
}

// shouldSkipItem checks if an item should be skipped due to backoff.
func (s *ScheduledSearcher) shouldSkipItem(ctx context.Context, itemType string, itemID int64) (bool, error) {
	status, err := s.service.queries.GetAutosearchStatus(ctx, sqlc.GetAutosearchStatusParams{
		ItemType: itemType,
		ItemID:   itemID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil // No status = never searched = don't skip
		}
		return false, err
	}

	// Skip if failure count exceeds threshold
	if status.FailureCount >= int64(s.config.BackoffThreshold) {
		s.logger.Debug().
			Str("itemType", itemType).
			Int64("itemId", itemID).
			Int64("failureCount", status.FailureCount).
			Msg("Skipping item due to backoff threshold")
		return true, nil
	}

	return false, nil
}

// processItems searches each item sequentially with rate limiting.
func (s *ScheduledSearcher) processItems(ctx context.Context, items []SearchableItem) *BatchSearchResult {
	result := &BatchSearchResult{
		TotalSearched: 0,
		Results:       make([]*SearchResult, 0, len(items)),
	}

	for i, item := range items {
		select {
		case <-ctx.Done():
			s.logger.Info().Msg("Scheduled search task cancelled")
			return result
		default:
		}

		// Broadcast progress
		s.broadcastTaskProgress(i+1, len(items), item.Title)

		// Apply rate limiting delay
		delay := s.rateLimiter.GetDelay()
		if delay > 0 && i > 0 {
			time.Sleep(delay)
		}

		// Execute search
		startTime := time.Now()
		searchResult, err := s.searchItem(ctx, item)
		responseTime := time.Since(startTime)

		// Record response time for adaptive rate limiting
		s.rateLimiter.RecordResponse(responseTime, nil)

		result.TotalSearched++

		if err != nil {
			s.logger.Warn().Err(err).
				Str("title", item.Title).
				Str("mediaType", string(item.MediaType)).
				Msg("Search failed")
			result.Failed++
			result.Results = append(result.Results, &SearchResult{Error: err.Error()})

			// Increment failure count for backoff
			s.incrementFailureCount(ctx, item)
			continue
		}

		result.Results = append(result.Results, searchResult)
		if searchResult.Found {
			result.Found++
			// Reset failure count on success
			s.resetFailureCount(ctx, item)
		} else {
			// No results found - increment failure count
			s.incrementFailureCount(ctx, item)
		}
		if searchResult.Downloaded {
			result.Downloaded++
		}
	}

	return result
}

// searchItem searches for a single item based on its type.
func (s *ScheduledSearcher) searchItem(ctx context.Context, item SearchableItem) (*SearchResult, error) {
	switch item.MediaType {
	case MediaTypeMovie:
		return s.service.SearchMovie(ctx, item.MediaID, SearchSourceScheduled)
	case MediaTypeEpisode:
		return s.service.SearchEpisode(ctx, item.MediaID, SearchSourceScheduled)
	case MediaTypeSeason:
		// Season pack search - searches for a single release containing all episodes
		return s.service.searchSeasonPackByID(ctx, item.MediaID, item.SeasonNumber, SearchSourceScheduled)
	default:
		return &SearchResult{Error: "unsupported media type"}, nil
	}
}

// incrementFailureCount increments the failure count for an item.
func (s *ScheduledSearcher) incrementFailureCount(ctx context.Context, item SearchableItem) {
	itemType := string(item.MediaType)
	if item.MediaType == MediaTypeSeason {
		itemType = "episode" // Track season by its first episode
	}

	err := s.service.queries.IncrementAutosearchFailure(ctx, sqlc.IncrementAutosearchFailureParams{
		ItemType: itemType,
		ItemID:   item.MediaID,
	})
	if err != nil {
		s.logger.Warn().Err(err).
			Str("itemType", itemType).
			Int64("mediaId", item.MediaID).
			Msg("Failed to increment failure count")
	}
}

// resetFailureCount resets the failure count for an item.
func (s *ScheduledSearcher) resetFailureCount(ctx context.Context, item SearchableItem) {
	itemType := string(item.MediaType)
	if item.MediaType == MediaTypeSeason {
		itemType = "episode"
	}

	err := s.service.queries.ResetAutosearchFailure(ctx, sqlc.ResetAutosearchFailureParams{
		ItemType: itemType,
		ItemID:   item.MediaID,
	})
	if err != nil {
		s.logger.Warn().Err(err).
			Str("itemType", itemType).
			Int64("mediaId", item.MediaID).
			Msg("Failed to reset failure count")
	}
}

// Broadcast helpers

func (s *ScheduledSearcher) broadcastTaskStarted(totalItems int) {
	if s.service.broadcaster == nil {
		return
	}
	s.service.broadcaster.Broadcast(EventAutoSearchTaskStarted, AutoSearchTaskStartedPayload{
		TotalItems: totalItems,
	})
}

func (s *ScheduledSearcher) broadcastTaskProgress(current, total int, title string) {
	if s.service.broadcaster == nil {
		return
	}
	s.service.broadcaster.Broadcast(EventAutoSearchTaskProgress, AutoSearchTaskProgressPayload{
		CurrentItem:  current,
		TotalItems:   total,
		CurrentTitle: title,
	})
}

func (s *ScheduledSearcher) broadcastTaskCompleted(result *BatchSearchResult, elapsed time.Duration) {
	if s.service.broadcaster == nil {
		return
	}
	s.service.broadcaster.Broadcast(EventAutoSearchTaskCompleted, AutoSearchTaskCompletedPayload{
		TotalSearched: result.TotalSearched,
		Found:         result.Found,
		Downloaded:    result.Downloaded,
		Failed:        result.Failed,
		ElapsedMs:     elapsed.Milliseconds(),
	})
}

// RateLimiter implements adaptive rate limiting for search requests.
type RateLimiter struct {
	mu               sync.Mutex
	baseDelayMs      int
	lastResponseTime time.Duration
	retryAfter       time.Time
}

// GetDelay returns the delay to wait before the next request.
func (r *RateLimiter) GetDelay() time.Duration {
	r.mu.Lock()
	defer r.mu.Unlock()

	// If Retry-After header was received, wait until that time
	if time.Now().Before(r.retryAfter) {
		return time.Until(r.retryAfter)
	}

	// Adaptive delay: base + (lastResponseTime * 0.5)
	adaptive := r.baseDelayMs + int(r.lastResponseTime.Milliseconds()/2)
	return time.Duration(adaptive) * time.Millisecond
}

// RecordResponse records response timing and rate limit headers.
func (r *RateLimiter) RecordResponse(responseTime time.Duration, headers http.Header) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.lastResponseTime = responseTime

	if headers != nil {
		// Parse Retry-After header
		if retryAfter := headers.Get("Retry-After"); retryAfter != "" {
			if seconds, err := strconv.Atoi(retryAfter); err == nil {
				r.retryAfter = time.Now().Add(time.Duration(seconds) * time.Second)
			}
		}
	}
}
