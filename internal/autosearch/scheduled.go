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
	"github.com/slipstream/slipstream/internal/module"
)

const (
	statusFailed      = "failed"
	statusMissing     = "missing"
	entityTypeSeries  = "series"
	searchTypeUpgrade = "upgrade"
)

// SeriesRefresher defines the interface for refreshing series metadata.
type SeriesRefresher interface {
	// RefreshMonitoredSeriesMetadata refreshes metadata for all monitored series.
	// Returns the number of series refreshed and any error encountered.
	RefreshMonitoredSeriesMetadata(ctx context.Context) (int, error)
}

// ScheduledSearcher handles scheduled automatic searches for missing items.
type ScheduledSearcher struct {
	service *Service
	config  *config.AutoSearchConfig
	logger  *zerolog.Logger

	// Rate limiting
	rateLimiter *RateLimiter

	// Optional series refresher for pre-search metadata updates
	seriesRefresher SeriesRefresher

	// Module registry for module-aware item collection
	registry *module.Registry

	// Task state
	mu      sync.Mutex
	running bool
}

// NewScheduledSearcher creates a new scheduled searcher.
func NewScheduledSearcher(service *Service, cfg *config.AutoSearchConfig, logger *zerolog.Logger) *ScheduledSearcher {
	loggerWithComponent := logger.With().Str("component", "scheduled-search").Logger()
	return &ScheduledSearcher{
		service: service,
		config:  cfg,
		logger:  &loggerWithComponent,
		rateLimiter: &RateLimiter{
			baseDelayMs: cfg.BaseDelayMs,
		},
	}
}

// SetSeriesRefresher sets the series refresher for pre-search metadata updates.
func (s *ScheduledSearcher) SetSeriesRefresher(refresher SeriesRefresher) {
	s.seriesRefresher = refresher
}

// SetRegistry sets the module registry for module-aware item collection.
func (s *ScheduledSearcher) SetRegistry(r *module.Registry) {
	s.registry = r
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

	// Refresh series metadata before searching to get latest episode lists
	if s.seriesRefresher != nil {
		s.logger.Info().Msg("Refreshing monitored series metadata before search")
		refreshed, err := s.seriesRefresher.RefreshMonitoredSeriesMetadata(ctx)
		if err != nil {
			s.logger.Warn().Err(err).Msg("Failed to refresh series metadata, continuing with search")
		} else {
			s.logger.Info().Int("refreshed", refreshed).Msg("Series metadata refresh completed")
		}
	}

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
	for i := range movieItems {
		items[i] = movieItems[i].item
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
	for i := range episodeItems {
		items[i] = episodeItems[i].item
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

// RunUpgradeMoviesOnly executes search for upgradable movies only.
func (s *ScheduledSearcher) RunUpgradeMoviesOnly(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		s.logger.Debug().Msg("Scheduled search task already running, skipping upgrade-movies-only")
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
	s.logger.Info().Msg("Starting search task for upgradable movies only")

	movieItems, err := s.collectUpgradeMovies(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to collect upgradable movies")
		return err
	}

	sort.Slice(movieItems, func(i, j int) bool {
		return movieItems[i].releaseDate.After(movieItems[j].releaseDate)
	})

	items := make([]SearchableItem, len(movieItems))
	for i := range movieItems {
		items[i] = movieItems[i].item
	}

	if len(items) == 0 {
		s.logger.Info().Msg("No upgradable movies to search")
		return nil
	}

	s.logger.Info().Int("itemCount", len(items)).Msg("Collected upgradable movies")
	s.broadcastTaskStarted(len(items))

	result := s.processItems(ctx, items)

	elapsed := time.Since(startTime)
	s.logger.Info().
		Int("searched", result.TotalSearched).
		Int("found", result.Found).
		Int("downloaded", result.Downloaded).
		Int("failed", result.Failed).
		Dur("elapsed", elapsed).
		Msg("Upgrade movies search task completed")

	s.broadcastTaskCompleted(result, elapsed)
	return nil
}

// RunUpgradeSeriesOnly executes search for upgradable series episodes only.
func (s *ScheduledSearcher) RunUpgradeSeriesOnly(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		s.logger.Debug().Msg("Scheduled search task already running, skipping upgrade-series-only")
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
	s.logger.Info().Msg("Starting search task for upgradable series only")

	episodeItems, err := s.collectUpgradeEpisodes(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to collect upgradable episodes")
		return err
	}

	sort.Slice(episodeItems, func(i, j int) bool {
		return episodeItems[i].releaseDate.After(episodeItems[j].releaseDate)
	})

	items := make([]SearchableItem, len(episodeItems))
	for i := range episodeItems {
		items[i] = episodeItems[i].item
	}

	if len(items) == 0 {
		s.logger.Info().Msg("No upgradable episodes to search")
		return nil
	}

	s.logger.Info().Int("itemCount", len(items)).Msg("Collected upgradable episodes")
	s.broadcastTaskStarted(len(items))

	result := s.processItems(ctx, items)

	elapsed := time.Since(startTime)
	s.logger.Info().
		Int("searched", result.TotalSearched).
		Int("found", result.Found).
		Int("downloaded", result.Downloaded).
		Int("failed", result.Failed).
		Dur("elapsed", elapsed).
		Msg("Upgrade series search task completed")

	s.broadcastTaskCompleted(result, elapsed)
	return nil
}

// searchableItemWithPriority wraps a searchable item with sorting metadata.
type searchableItemWithPriority struct {
	item         SearchableItem
	releaseDate  time.Time
	isSeasonPack bool
}

type seasonKey struct {
	seriesID     int64
	seasonNumber int64
}

// collectSearchableItems gathers all missing and upgrade-eligible movies and episodes, ordered by release date.
// When a module registry is available, collection is delegated to module WantedCollectors;
// otherwise the legacy sqlc-based collection path is used.
func (s *ScheduledSearcher) collectSearchableItems(ctx context.Context) ([]SearchableItem, error) {
	if s.registry != nil {
		return s.collectFromModules(ctx)
	}
	return s.collectLegacy(ctx)
}

// collectFromModules uses module WantedCollector implementations to gather searchable items.
func (s *ScheduledSearcher) collectFromModules(ctx context.Context) ([]SearchableItem, error) {
	var allItems []searchableItemWithPriority

	for _, mod := range s.registry.Enabled() {
		strategy, hasStrategy := mod.(module.SearchStrategy)

		// Collect missing items
		missingItems, err := mod.CollectMissing(ctx)
		if err != nil {
			s.logger.Warn().Err(err).Str("module", string(mod.ID())).Msg("Failed to collect missing items from module")
			continue
		}

		missingConverted := s.convertAndFilterModuleItems(ctx, missingItems, false)

		// Group episodes into season packs when eligible
		if hasStrategy {
			missingConverted = s.groupModuleItemsIntoSeasonPacks(ctx, missingConverted, strategy, false)
		}

		allItems = append(allItems, missingConverted...)

		// Collect upgradable items
		upgradeItems, err := mod.CollectUpgradable(ctx)
		if err != nil {
			s.logger.Warn().Err(err).Str("module", string(mod.ID())).Msg("Failed to collect upgradable items from module")
			continue
		}

		upgradeConverted := s.convertAndFilterModuleItems(ctx, upgradeItems, true)

		// Group upgrade episodes into season packs when eligible
		if hasStrategy {
			upgradeConverted = s.groupModuleItemsIntoSeasonPacks(ctx, upgradeConverted, strategy, true)
		}

		allItems = append(allItems, upgradeConverted...)
	}

	// Sort by release date (newest first)
	sort.Slice(allItems, func(i, j int) bool {
		return allItems[i].releaseDate.After(allItems[j].releaseDate)
	})

	result := make([]SearchableItem, len(allItems))
	for i := range allItems {
		result[i] = allItems[i].item
	}

	return result, nil
}

// convertAndFilterModuleItems applies backoff checks to filter out items that should be skipped.
func (s *ScheduledSearcher) convertAndFilterModuleItems(
	ctx context.Context,
	moduleItems []module.SearchableItem,
	forUpgrade bool,
) []searchableItemWithPriority {
	items := make([]searchableItemWithPriority, 0, len(moduleItems))

	searchType := statusMissing
	if forUpgrade {
		searchType = searchTypeUpgrade
	}

	for _, mi := range moduleItems {
		entityType := mi.GetMediaType()
		entityID := mi.GetEntityID()

		shouldSkip, err := s.shouldSkipItem(ctx, entityType, entityID, searchType)
		if err != nil {
			s.logger.Warn().Err(err).
				Str("entityType", entityType).
				Int64("entityId", entityID).
				Msg("Failed to check backoff status for module item")
		}
		if shouldSkip {
			continue
		}

		// Extract release date from SearchParams.Extra if available
		var releaseDate time.Time
		params := mi.GetSearchParams()
		if rd, ok := params.Extra["releaseDate"].(time.Time); ok {
			releaseDate = rd
		} else if rd, ok := params.Extra["physicalReleaseDate"].(time.Time); ok {
			releaseDate = rd
		} else if rd, ok := params.Extra["airDate"].(time.Time); ok {
			releaseDate = rd
		}

		items = append(items, searchableItemWithPriority{
			item:        mi,
			releaseDate: releaseDate,
		})
	}

	return items
}

// groupModuleItemsIntoSeasonPacks groups episode items by season and replaces them with
// a single season pack item when all episodes in the season are eligible for group search.
func (s *ScheduledSearcher) groupModuleItemsIntoSeasonPacks(
	ctx context.Context,
	items []searchableItemWithPriority,
	strategy module.SearchStrategy,
	forUpgrade bool,
) []searchableItemWithPriority {
	var result []searchableItemWithPriority
	episodesBySeason := make(map[seasonKey][]searchableItemWithPriority)

	for i := range items {
		if items[i].item.GetMediaType() != string(MediaTypeEpisode) {
			result = append(result, items[i])
			continue
		}
		seriesID := module.ItemSeriesID(items[i].item)
		seasonNum := module.ItemSeasonNumber(items[i].item)
		key := seasonKey{seriesID: seriesID, seasonNumber: int64(seasonNum)}
		episodesBySeason[key] = append(episodesBySeason[key], items[i])
	}

	for key, episodes := range episodesBySeason {
		if !strategy.IsGroupSearchEligible(ctx, "series", key.seriesID, int(key.seasonNumber), forUpgrade) {
			result = append(result, episodes...)
			continue
		}

		// Check backoff for the series-level season pack search
		searchType := statusMissing
		if forUpgrade {
			searchType = searchTypeUpgrade
		}
		shouldSkip, err := s.shouldSkipItem(ctx, entityTypeSeries, key.seriesID, searchType)
		if err != nil {
			s.logger.Warn().Err(err).
				Int64("seriesId", key.seriesID).
				Int64("season", key.seasonNumber).
				Msg("Failed to check backoff status for season pack")
		}
		if shouldSkip {
			continue
		}

		result = append(result, s.buildModuleSeasonPackItem(key, episodes, forUpgrade))
	}

	return result
}

// buildModuleSeasonPackItem constructs a season pack searchableItemWithPriority from
// grouped episode items collected via module WantedCollectors.
func (s *ScheduledSearcher) buildModuleSeasonPackItem(key seasonKey, episodes []searchableItemWithPriority, forUpgrade bool) searchableItemWithPriority {
	first := episodes[0].item

	var currentQID *int64
	if forUpgrade {
		var maxQID int64
		for j := range episodes {
			if qid := episodes[j].item.GetCurrentQualityID(); qid != nil && *qid > maxQID {
				maxQID = *qid
			}
		}
		currentQID = &maxQID
	}

	extra := map[string]any{
		"seriesId":     key.seriesID,
		"seasonNumber": int(key.seasonNumber),
	}
	if year := module.ItemYear(first); year > 0 {
		extra["year"] = year
	}

	seasonItem := module.NewWantedItem(
		module.Type(first.GetModuleType()),
		string(MediaTypeSeason),
		key.seriesID,
		first.GetTitle(),
		first.GetExternalIDs(),
		first.GetQualityProfileID(),
		currentQID,
		module.SearchParams{Extra: extra},
	)

	return searchableItemWithPriority{
		item:         seasonItem,
		releaseDate:  episodes[0].releaseDate,
		isSeasonPack: true,
	}
}

// collectLegacy gathers items using direct sqlc queries (pre-module path).
func (s *ScheduledSearcher) collectLegacy(ctx context.Context) ([]SearchableItem, error) {
	var items []searchableItemWithPriority

	// Get missing movies
	movies, err := s.collectMissingMovies(ctx)
	if err != nil {
		return nil, err
	}
	items = append(items, movies...)

	// Get upgrade candidate movies (files below quality cutoff)
	upgradeMovies, err := s.collectUpgradeMovies(ctx)
	if err != nil {
		return nil, err
	}
	items = append(items, upgradeMovies...)

	// Get missing episodes (with boxset prioritization)
	episodes, err := s.collectMissingEpisodes(ctx)
	if err != nil {
		return nil, err
	}
	items = append(items, episodes...)

	// Get upgrade candidate episodes (files below quality cutoff)
	upgradeEpisodes, err := s.collectUpgradeEpisodes(ctx)
	if err != nil {
		return nil, err
	}
	items = append(items, upgradeEpisodes...)

	// Sort by release date (newest first)
	sort.Slice(items, func(i, j int) bool {
		return items[i].releaseDate.After(items[j].releaseDate)
	})

	// Extract just the searchable items
	result := make([]SearchableItem, len(items))
	for i := range items {
		result[i] = items[i].item
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
		// Skip failed items - they require manual retry to reset
		if row.Status == statusFailed {
			continue
		}

		// Check backoff status
		shouldSkip, err := s.shouldSkipItem(ctx, "movie", row.ID, statusMissing)
		if err != nil {
			s.logger.Warn().Err(err).Int64("movieId", row.ID).Msg("Failed to check backoff status")
		}
		if shouldSkip {
			continue
		}

		item := s.service.movieToSearchableItem(ctx, row)

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
// groupMissingEpisodesBySeason groups missing episodes by series and season.
func groupMissingEpisodesBySeason(rows []*sqlc.ListMissingEpisodesRow) map[seasonKey][]*sqlc.ListMissingEpisodesRow {
	seasonEpisodes := make(map[seasonKey][]*sqlc.ListMissingEpisodesRow)
	for _, row := range rows {
		if row.Status == statusFailed {
			continue
		}
		key := seasonKey{seriesID: row.SeriesID, seasonNumber: row.SeasonNumber}
		seasonEpisodes[key] = append(seasonEpisodes[key], row)
	}
	return seasonEpisodes
}

// buildMissingSeasonPackItem constructs a SearchableItem for season pack search.
func buildMissingSeasonPackItem(firstEp *sqlc.ListMissingEpisodesRow, seasonNumber int64) SearchableItem {
	extIDs := make(map[string]string)
	if firstEp.SeriesTvdbID.Valid {
		extIDs["tvdbId"] = strconv.FormatInt(firstEp.SeriesTvdbID.Int64, 10)
	}
	if firstEp.SeriesTmdbID.Valid {
		extIDs["tmdbId"] = strconv.FormatInt(firstEp.SeriesTmdbID.Int64, 10)
	}
	if firstEp.SeriesImdbID.Valid {
		extIDs["imdbId"] = firstEp.SeriesImdbID.String
	}

	extra := map[string]any{
		"seriesId":     firstEp.SeriesID,
		"seasonNumber": int(seasonNumber),
	}
	if firstEp.SeriesYear.Valid {
		extra["year"] = int(firstEp.SeriesYear.Int64)
	}

	var profileID int64
	if firstEp.SeriesQualityProfileID.Valid {
		profileID = firstEp.SeriesQualityProfileID.Int64
	}

	return module.NewWantedItem(module.TypeTV, string(MediaTypeSeason), firstEp.SeriesID, firstEp.SeriesTitle, extIDs, profileID, nil, module.SearchParams{Extra: extra})
}

// buildMissingEpisodeItem constructs a SearchableItem for individual episode search.
func buildMissingEpisodeItem(ep *sqlc.ListMissingEpisodesRow) SearchableItem {
	extIDs := make(map[string]string)
	if ep.SeriesTvdbID.Valid {
		extIDs["tvdbId"] = strconv.FormatInt(ep.SeriesTvdbID.Int64, 10)
	}
	if ep.SeriesTmdbID.Valid {
		extIDs["tmdbId"] = strconv.FormatInt(ep.SeriesTmdbID.Int64, 10)
	}
	if ep.SeriesImdbID.Valid {
		extIDs["imdbId"] = ep.SeriesImdbID.String
	}

	extra := map[string]any{
		"seriesId":      ep.SeriesID,
		"seasonNumber":  int(ep.SeasonNumber),
		"episodeNumber": int(ep.EpisodeNumber),
	}
	if ep.SeriesYear.Valid {
		extra["year"] = int(ep.SeriesYear.Int64)
	}

	var profileID int64
	if ep.SeriesQualityProfileID.Valid {
		profileID = ep.SeriesQualityProfileID.Int64
	}

	return module.NewWantedItem(module.TypeTV, string(MediaTypeEpisode), ep.ID, ep.SeriesTitle, extIDs, profileID, nil, module.SearchParams{Extra: extra})
}

// tryAddMissingSeasonPackSearch attempts to add a season pack search if eligible.
func (s *ScheduledSearcher) tryAddMissingSeasonPackSearch(
	ctx context.Context,
	key seasonKey,
	episodes []*sqlc.ListMissingEpisodesRow,
) (searchableItemWithPriority, bool) {
	if !s.service.isSeasonPackEligible(ctx, key.seriesID, int(key.seasonNumber)) {
		return searchableItemWithPriority{}, false
	}

	shouldSkip, err := s.shouldSkipItem(ctx, entityTypeSeries, key.seriesID, statusMissing)
	if err != nil {
		s.logger.Warn().Err(err).Int64("seriesId", key.seriesID).Int64("season", key.seasonNumber).Msg("Failed to check backoff status for season")
	}
	if shouldSkip {
		return searchableItemWithPriority{}, false
	}

	firstEp := episodes[0]
	var airDate time.Time
	if firstEp.AirDate.Valid {
		airDate = firstEp.AirDate.Time
	}

	item := buildMissingSeasonPackItem(firstEp, key.seasonNumber)
	return searchableItemWithPriority{
		item:         item,
		releaseDate:  airDate,
		isSeasonPack: true,
	}, true
}

// collectMissingEpisodesIndividually processes episodes individually (not as season pack).
func (s *ScheduledSearcher) collectMissingEpisodesIndividually(
	ctx context.Context,
	episodes []*sqlc.ListMissingEpisodesRow,
) []searchableItemWithPriority {
	var items []searchableItemWithPriority
	for _, ep := range episodes {
		shouldSkip, err := s.shouldSkipItem(ctx, "episode", ep.ID, statusMissing)
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

		item := buildMissingEpisodeItem(ep)
		items = append(items, searchableItemWithPriority{
			item:        item,
			releaseDate: airDate,
		})
	}
	return items
}

// collectMissingEpisodes collects missing episodes with boxset prioritization.
func (s *ScheduledSearcher) collectMissingEpisodes(ctx context.Context) ([]searchableItemWithPriority, error) {
	rows, err := s.service.queries.ListMissingEpisodes(ctx)
	if err != nil {
		return nil, err
	}

	seasonEpisodes := groupMissingEpisodesBySeason(rows)
	var items []searchableItemWithPriority

	for key, episodes := range seasonEpisodes {
		if seasonItem, ok := s.tryAddMissingSeasonPackSearch(ctx, key, episodes); ok {
			items = append(items, seasonItem)
		} else {
			items = append(items, s.collectMissingEpisodesIndividually(ctx, episodes)...)
		}
	}

	return items, nil
}

// collectUpgradeMovies collects movies with files below quality cutoff.
func (s *ScheduledSearcher) collectUpgradeMovies(ctx context.Context) ([]searchableItemWithPriority, error) {
	rows, err := s.service.queries.ListMovieUpgradeCandidates(ctx)
	if err != nil {
		return nil, err
	}

	items := make([]searchableItemWithPriority, 0, len(rows))
	for _, row := range rows {
		shouldSkip, err := s.shouldSkipItem(ctx, "movie", row.ID, searchTypeUpgrade)
		if err != nil {
			s.logger.Warn().Err(err).Int64("movieId", row.ID).Msg("Failed to check backoff status")
		}
		if shouldSkip {
			continue
		}

		item := s.service.movieUpgradeCandidateToSearchableItem(row)

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

// collectUpgradeEpisodes collects episodes with files below quality cutoff.
// When all monitored episodes in a season are upgradable, a single season pack
// search is used instead of individual episode searches.
func (s *ScheduledSearcher) collectUpgradeEpisodes(ctx context.Context) ([]searchableItemWithPriority, error) {
	rows, err := s.service.queries.ListEpisodeUpgradeCandidates(ctx)
	if err != nil {
		return nil, err
	}

	seasonEpisodes := make(map[seasonKey][]*sqlc.ListEpisodeUpgradeCandidatesRow)

	for _, row := range rows {
		key := seasonKey{seriesID: row.SeriesID, seasonNumber: row.SeasonNumber}
		seasonEpisodes[key] = append(seasonEpisodes[key], row)
	}

	var items []searchableItemWithPriority

	for key, episodes := range seasonEpisodes {
		if !s.service.isSeasonPackUpgradeEligible(ctx, key.seriesID, int(key.seasonNumber)) {
			items = append(items, s.collectIndividualEpisodeUpgrades(ctx, episodes)...)
			continue
		}

		if s.shouldSkipSeasonUpgrade(ctx, key) {
			continue
		}

		seasonItem := s.buildSeasonPackUpgradeItem(key, episodes)
		items = append(items, seasonItem)
	}

	return items, nil
}

func (s *ScheduledSearcher) shouldSkipSeasonUpgrade(ctx context.Context, key seasonKey) bool {
	shouldSkip, err := s.shouldSkipItem(ctx, entityTypeSeries, key.seriesID, searchTypeUpgrade)
	if err != nil {
		s.logger.Warn().Err(err).Int64("seriesId", key.seriesID).Int64("season", key.seasonNumber).Msg("Failed to check backoff status for season upgrade")
	}
	return shouldSkip
}

func (s *ScheduledSearcher) buildSeasonPackUpgradeItem(key seasonKey, episodes []*sqlc.ListEpisodeUpgradeCandidatesRow) searchableItemWithPriority {
	firstEp := episodes[0]

	maxQualityID := s.findMaxQualityID(episodes)

	var airDate time.Time
	if firstEp.AirDate.Valid {
		airDate = firstEp.AirDate.Time
	}

	extIDs := make(map[string]string)
	if firstEp.SeriesTvdbID.Valid {
		extIDs["tvdbId"] = strconv.FormatInt(firstEp.SeriesTvdbID.Int64, 10)
	}
	if firstEp.SeriesTmdbID.Valid {
		extIDs["tmdbId"] = strconv.FormatInt(firstEp.SeriesTmdbID.Int64, 10)
	}
	if firstEp.SeriesImdbID.Valid {
		extIDs["imdbId"] = firstEp.SeriesImdbID.String
	}

	extra := map[string]any{
		"seriesId":     key.seriesID,
		"seasonNumber": int(key.seasonNumber),
	}
	if firstEp.SeriesYear.Valid {
		extra["year"] = int(firstEp.SeriesYear.Int64)
	}

	var profileID int64
	if firstEp.SeriesQualityProfileID.Valid {
		profileID = firstEp.SeriesQualityProfileID.Int64
	}

	qid := int64(maxQualityID)
	item := module.NewWantedItem(module.TypeTV, string(MediaTypeSeason), key.seriesID, firstEp.SeriesTitle, extIDs, profileID, &qid, module.SearchParams{Extra: extra})

	s.logger.Info().
		Str("title", firstEp.SeriesTitle).
		Int64("seriesId", key.seriesID).
		Int64("season", key.seasonNumber).
		Int("episodes", len(episodes)).
		Int("maxCurrentQualityId", maxQualityID).
		Msg("Using season pack search for upgrade")

	return searchableItemWithPriority{
		item:         item,
		releaseDate:  airDate,
		isSeasonPack: true,
	}
}

func (s *ScheduledSearcher) findMaxQualityID(episodes []*sqlc.ListEpisodeUpgradeCandidatesRow) int {
	var maxQualityID int
	for _, ep := range episodes {
		if ep.CurrentQualityID.Valid && int(ep.CurrentQualityID.Int64) > maxQualityID {
			maxQualityID = int(ep.CurrentQualityID.Int64)
		}
	}
	return maxQualityID
}

func (s *ScheduledSearcher) collectIndividualEpisodeUpgrades(ctx context.Context, episodes []*sqlc.ListEpisodeUpgradeCandidatesRow) []searchableItemWithPriority {
	var items []searchableItemWithPriority

	for _, ep := range episodes {
		shouldSkip, err := s.shouldSkipItem(ctx, "episode", ep.ID, searchTypeUpgrade)
		if err != nil {
			s.logger.Warn().Err(err).Int64("episodeId", ep.ID).Msg("Failed to check backoff status")
		}
		if shouldSkip {
			continue
		}

		item := s.service.episodeUpgradeCandidateToSearchableItem(ep)

		var airDate time.Time
		if ep.AirDate.Valid {
			airDate = ep.AirDate.Time
		}

		items = append(items, searchableItemWithPriority{
			item:        item,
			releaseDate: airDate,
		})
	}

	return items
}

// shouldSkipItem checks if an item should be skipped due to backoff.
func (s *ScheduledSearcher) shouldSkipItem(ctx context.Context, itemType string, itemID int64, searchType string) (bool, error) {
	modType := s.moduleTypeFromEntityType(itemType)
	status, err := s.service.queries.GetAutosearchStatus(ctx, sqlc.GetAutosearchStatusParams{
		ModuleType: modType,
		EntityType: itemType,
		EntityID:   itemID,
		SearchType: searchType,
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

		hasFile := module.ItemHasFile(item)

		// Determine search type based on whether item has a file (upgrade vs missing)
		searchType := statusMissing
		if hasFile {
			searchType = searchTypeUpgrade
		}

		// Broadcast progress
		s.broadcastTaskProgress(i+1, len(items), item.GetTitle())

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
				Str("title", item.GetTitle()).
				Str("mediaType", item.GetMediaType()).
				Msg("Search failed")
			result.Failed++
			result.Results = append(result.Results, &SearchResult{Error: err.Error()})

			// Increment failure count for backoff
			s.incrementFailureCount(ctx, item, searchType)
			continue
		}

		result.Results = append(result.Results, searchResult)
		if searchResult.Found {
			result.Found++
			// Reset failure count on success
			s.resetFailureCount(ctx, item, searchType)
		} else {
			// No results found - increment failure count
			s.incrementFailureCount(ctx, item, searchType)
		}
		if searchResult.Downloaded {
			result.Downloaded++
		}
	}

	return result
}

// searchItem searches for a single item based on its type.
func (s *ScheduledSearcher) searchItem(ctx context.Context, item SearchableItem) (*SearchResult, error) {
	mediaType := item.GetMediaType()
	switch mediaType {
	case string(MediaTypeMovie):
		return s.service.SearchMovie(ctx, item.GetEntityID(), SearchSourceScheduled)
	case string(MediaTypeEpisode):
		return s.service.SearchEpisode(ctx, item.GetEntityID(), SearchSourceScheduled)
	case string(MediaTypeSeason):
		hasFile := module.ItemHasFile(item)
		if hasFile {
			currentQualityID := module.ItemCurrentQualityID(item)
			seasonNumber := module.ItemSeasonNumber(item)
			// Season pack upgrade search with individual episode fallback
			batchResult, err := s.service.SearchSeasonUpgrade(ctx, item.GetEntityID(), seasonNumber, currentQualityID, SearchSourceScheduled)
			if err != nil {
				return nil, err
			}
			// Aggregate batch result into a single SearchResult for the scheduled search loop
			result := &SearchResult{
				Found:      batchResult.Found > 0,
				Downloaded: batchResult.Downloaded > 0,
			}
			// Use the first downloaded release's info if available
			for _, r := range batchResult.Results {
				if !r.Downloaded || r.Release == nil {
					continue
				}
				result.Release = r.Release
				result.ClientName = r.ClientName
				result.DownloadID = r.DownloadID
				result.Upgraded = true
				break
			}
			return result, nil
		}
		// Missing season pack search
		seasonNumber := module.ItemSeasonNumber(item)
		return s.service.searchSeasonPackByID(ctx, item.GetEntityID(), seasonNumber, SearchSourceScheduled)
	default:
		return &SearchResult{Error: "unsupported media type"}, nil
	}
}

// incrementFailureCount increments the failure count for an item.
func (s *ScheduledSearcher) incrementFailureCount(ctx context.Context, item SearchableItem, searchType string) {
	itemType := item.GetMediaType()
	if itemType == string(MediaTypeSeason) {
		itemType = entityTypeSeries
	}

	modType := s.moduleTypeFromEntityType(itemType)
	err := s.service.queries.IncrementAutosearchFailure(ctx, sqlc.IncrementAutosearchFailureParams{
		ModuleType: modType,
		EntityType: itemType,
		EntityID:   item.GetEntityID(),
		SearchType: searchType,
	})
	if err != nil {
		s.logger.Warn().Err(err).
			Str("itemType", itemType).
			Int64("mediaId", item.GetEntityID()).
			Msg("Failed to increment failure count")
	}
}

// resetFailureCount resets the failure count for an item.
func (s *ScheduledSearcher) resetFailureCount(ctx context.Context, item SearchableItem, searchType string) {
	itemType := item.GetMediaType()
	if itemType == string(MediaTypeSeason) {
		itemType = entityTypeSeries
	}

	modType2 := s.moduleTypeFromEntityType(itemType)
	err := s.service.queries.ResetAutosearchFailure(ctx, sqlc.ResetAutosearchFailureParams{
		ModuleType: modType2,
		EntityType: itemType,
		EntityID:   item.GetEntityID(),
		SearchType: searchType,
	})
	if err != nil {
		s.logger.Warn().Err(err).
			Str("itemType", itemType).
			Int64("mediaId", item.GetEntityID()).
			Msg("Failed to reset failure count")
	}
}

// moduleTypeFromEntityType maps an entity type to its owning module type using
// the registry. Falls back to the legacy movie/tv assumption when the registry
// is not available.
func (s *ScheduledSearcher) moduleTypeFromEntityType(entityType string) string {
	if s.registry != nil {
		if mod := s.registry.ModuleForEntityType(module.EntityType(entityType)); mod != nil {
			return string(mod.ID())
		}
	}
	// Legacy fallback for pre-module code paths
	if entityType == "movie" {
		return "movie"
	}
	return "tv"
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
