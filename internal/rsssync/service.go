package rsssync

import (
	"context"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/decisioning"
	"github.com/slipstream/slipstream/internal/health"
	"github.com/slipstream/slipstream/internal/history"
	"github.com/slipstream/slipstream/internal/indexer/grab"
	"github.com/slipstream/slipstream/internal/indexer/scoring"
	"github.com/slipstream/slipstream/internal/indexer/types"
	"github.com/slipstream/slipstream/internal/library/quality"
	"github.com/slipstream/slipstream/internal/module"
	"github.com/slipstream/slipstream/internal/websocket"
)

// Media type string constants for module.SearchableItem comparisons.
const (
	mediaTypeMovie   = "movie"
	mediaTypeEpisode = "episode"
	mediaTypeSeason  = "season"
)

// SyncStatus holds the result of the last RSS sync run.
type SyncStatus struct {
	Running       bool      `json:"running"`
	LastRun       time.Time `json:"lastRun,omitempty"`
	TotalReleases int       `json:"totalReleases"`
	Matched       int       `json:"matched"`
	Grabbed       int       `json:"grabbed"`
	ElapsedMs     int       `json:"elapsed"`
	Error         string    `json:"error,omitempty"`
}

// Service orchestrates RSS sync operations.
type Service struct {
	queries        *sqlc.Queries
	fetcher        *FeedFetcher
	grabService    *grab.Service
	qualityService *quality.Service
	historyService *history.Service
	grabLock       *decisioning.GrabLock
	healthService  *health.Service
	hub            *websocket.Hub
	logger         *zerolog.Logger
	registry       *module.Registry

	running atomic.Bool
	mu      sync.RWMutex
	status  SyncStatus
}

// NewService creates a new RSS sync service.
func NewService(
	queries *sqlc.Queries,
	fetcher *FeedFetcher,
	grabService *grab.Service,
	qualityService *quality.Service,
	historyService *history.Service,
	grabLock *decisioning.GrabLock,
	healthService *health.Service,
	hub *websocket.Hub,
	logger *zerolog.Logger,
) *Service {
	return &Service{
		queries:        queries,
		fetcher:        fetcher,
		grabService:    grabService,
		qualityService: qualityService,
		historyService: historyService,
		grabLock:       grabLock,
		healthService:  healthService,
		hub:            hub,
		logger:         logger,
	}
}

// SetDB updates the database connection (for dev mode switching).
func (s *Service) SetDB(queries *sqlc.Queries) {
	s.queries = queries
	s.fetcher.queries = queries
}

// SetRegistry sets the module registry for module-aware wanted collection.
func (s *Service) SetRegistry(r *module.Registry) {
	s.registry = r
}

// IsRunning returns whether an RSS sync is currently running.
func (s *Service) IsRunning() bool {
	return s.running.Load()
}

// LastStatus returns the last sync status.
func (s *Service) LastStatus() SyncStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	st := s.status
	st.Running = s.running.Load()
	return st
}

// Run executes a full RSS sync cycle.
func (s *Service) Run(ctx context.Context) error {
	if !s.running.CompareAndSwap(false, true) {
		return nil
	}
	defer s.running.Store(false)

	start := time.Now()
	s.logger.Info().Msg("RSS sync starting")

	feeds := s.fetcher.FetchAll(ctx)
	if len(feeds) == 0 {
		s.logger.Info().Msg("no RSS feeds fetched")
		s.setStatus(&SyncStatus{LastRun: start})
		return nil
	}

	s.broadcast(EventStarted, StartedEvent{IndexerCount: len(feeds)})

	wantedItems, err := s.collectWantedItems(ctx, start)
	if err != nil {
		return err
	}
	if wantedItems == nil {
		return nil
	}

	index := BuildWantedIndex(wantedItems)
	s.logger.Info().Int("wantedItems", len(wantedItems)).Msg("built wanted index")

	totalReleases, allMatches := s.matchFeeds(ctx, feeds, index)
	s.logger.Info().Int("totalReleases", totalReleases).Int("matched", len(allMatches)).Msg("RSS matching complete")

	grabbed := s.scoreAndGrab(ctx, groupMatches(allMatches))
	s.updateCacheBoundaries(ctx, feeds)
	s.completeSyncStatus(start, totalReleases, len(allMatches), grabbed)

	return nil
}

func (s *Service) collectWantedItems(ctx context.Context, start time.Time) ([]module.SearchableItem, error) {
	if s.registry != nil && len(s.registry.All()) > 0 {
		return s.collectWantedFromModules(ctx, start)
	}
	return s.collectWantedLegacy(ctx, start)
}

func (s *Service) collectWantedFromModules(ctx context.Context, start time.Time) ([]module.SearchableItem, error) {
	var allItems []module.SearchableItem
	for _, mod := range s.registry.All() {
		collector, ok := mod.(module.WantedCollector)
		if !ok {
			continue
		}
		modDesc, ok := mod.(module.Descriptor)
		if !ok {
			continue
		}

		missing, err := collector.CollectMissing(ctx)
		if err != nil {
			s.logger.Warn().Err(err).Str("module", string(modDesc.ID())).Msg("failed to collect missing items")
			continue
		}
		allItems = append(allItems, missing...)

		upgradable, err := collector.CollectUpgradable(ctx)
		if err != nil {
			s.logger.Warn().Err(err).Str("module", string(modDesc.ID())).Msg("failed to collect upgradable items")
			continue
		}
		allItems = append(allItems, upgradable...)
	}

	if len(allItems) == 0 {
		s.logger.Info().Msg("no wanted items from modules, skipping RSS sync")
		s.setStatus(&SyncStatus{LastRun: start})
		s.broadcast(EventCompleted, CompletedEvent{ElapsedMs: int(time.Since(start).Milliseconds())})
		return nil, nil
	}
	return allItems, nil
}

func (s *Service) collectWantedLegacy(ctx context.Context, start time.Time) ([]module.SearchableItem, error) {
	collector := &decisioning.Collector{
		Queries:        s.queries,
		Logger:         s.logger,
		BackoffChecker: decisioning.NoBackoff{},
	}
	legacyItems, err := decisioning.CollectWantedItems(ctx, collector)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to collect wanted items")
		s.failSync(start, err.Error())
		return nil, err
	}
	if len(legacyItems) == 0 {
		s.logger.Info().Msg("no wanted items, skipping RSS sync")
		s.setStatus(&SyncStatus{LastRun: start})
		s.broadcast(EventCompleted, CompletedEvent{ElapsedMs: int(time.Since(start).Milliseconds())})
		return nil, nil
	}
	return legacyItems, nil
}

func (s *Service) matchFeeds(ctx context.Context, feeds []IndexerFeed, index *WantedIndex) (int, []MatchResult) {
	var totalReleases int
	var allMatches []MatchResult

	for _, feed := range feeds {
		if feed.Error != nil {
			s.reportFeedHealth(feed.IndexerName, feed.Error)
			continue
		}

		matcher := NewMatcher(index, s.queries, s.registry, s.logger)
		feedMatches := s.processFeed(ctx, feed, matcher)
		totalReleases += len(feed.Releases)
		allMatches = append(allMatches, feedMatches...)

		s.reportFeedHealth(feed.IndexerName, nil)
		s.broadcast(EventProgress, ProgressEvent{
			Indexer:       feed.IndexerName,
			ReleasesFound: len(feed.Releases),
			Matched:       len(feedMatches),
		})
	}
	return totalReleases, allMatches
}

func (s *Service) updateCacheBoundaries(ctx context.Context, feeds []IndexerFeed) {
	for _, feed := range feeds {
		if feed.Error != nil || len(feed.Releases) == 0 {
			continue
		}
		newest := &feed.Releases[0]
		if err := UpdateCacheBoundary(ctx, s.queries, feed.IndexerID, newest); err != nil {
			s.logger.Warn().Err(err).Int64("indexerID", feed.IndexerID).Msg("failed to update RSS cache boundary")
		}
	}
}

func (s *Service) completeSyncStatus(start time.Time, totalReleases, matched, grabbed int) {
	elapsed := int(time.Since(start).Milliseconds())
	s.setStatus(&SyncStatus{
		LastRun:       start,
		TotalReleases: totalReleases,
		Matched:       matched,
		Grabbed:       grabbed,
		ElapsedMs:     elapsed,
	})

	s.broadcast(EventCompleted, CompletedEvent{
		TotalReleases: totalReleases,
		Matched:       matched,
		Grabbed:       grabbed,
		ElapsedMs:     elapsed,
	})

	s.logger.Info().
		Int("totalReleases", totalReleases).
		Int("matched", matched).
		Int("grabbed", grabbed).
		Int("elapsedMs", elapsed).
		Msg("RSS sync completed")
}

// processFeed matches releases from a single feed against the wanted index.
func (s *Service) processFeed(ctx context.Context, feed IndexerFeed, matcher *Matcher) []MatchResult {
	boundary, _ := GetCacheBoundary(ctx, s.queries, feed.IndexerID)

	var matches []MatchResult
	for i := range feed.Releases {
		release := &feed.Releases[i]

		// Stop at cache boundary
		if IsAtCacheBoundary(release, boundary) {
			s.logger.Debug().
				Str("indexer", feed.IndexerName).
				Str("url", release.DownloadURL).
				Msg("reached RSS cache boundary")
			break
		}

		results := matcher.Match(ctx, release)
		matches = append(matches, results...)
	}

	if boundary == nil && len(feed.Releases) >= maxResultsPerIndexer {
		s.logger.Warn().
			Str("indexer", feed.IndexerName).
			Int("releases", len(feed.Releases)).
			Msg("RSS cache boundary not reached — consider increasing sync frequency")
	}

	return matches
}

// matchGroup holds matches grouped by a single wanted item.
type matchGroup struct {
	item     module.SearchableItem
	releases []types.TorrentInfo
	isSeason bool
}

func groupMatches(matches []MatchResult) map[string]*matchGroup {
	groups := make(map[string]*matchGroup)
	for i := range matches {
		m := &matches[i]
		key := itemKey(m.WantedItem)
		g, ok := groups[key]
		if !ok {
			g = &matchGroup{
				item:     m.WantedItem,
				isSeason: m.IsSeason,
			}
			groups[key] = g
		}
		g.releases = append(g.releases, m.Release)
	}
	return groups
}

// scoreAndGrab scores matched releases, selects the best, and grabs them.
func (s *Service) scoreAndGrab(ctx context.Context, groups map[string]*matchGroup) int {
	scorer := scoring.NewDefaultScorer()

	// Track season pack grabs for suppression
	seasonGrabs := make(map[string]bool) // "season:seriesID:season" → grabbed

	// Process season packs first so we can suppress individual episodes
	var seasonKeys, episodeKeys []string
	for key, g := range groups {
		if g.isSeason {
			seasonKeys = append(seasonKeys, key)
		} else {
			episodeKeys = append(episodeKeys, key)
		}
	}

	grabbed := 0

	// Process seasons
	for _, key := range seasonKeys {
		if s.processGroup(ctx, scorer, groups[key]) {
			seasonGrabs[key] = true
			grabbed++
		}
	}

	// Process individual items, suppressing episodes covered by season packs
	for _, key := range episodeKeys {
		g := groups[key]
		if g.item.GetMediaType() == mediaTypeEpisode {
			sKey := seasonKeyForItem(g.item)
			if seasonGrabs[sKey] {
				sp := g.item.GetSearchParams()
				sn, _ := sp.Extra["seasonNumber"].(int)
				en, _ := sp.Extra["episodeNumber"].(int)
				s.logger.Debug().
					Str("title", g.item.GetTitle()).
					Int("season", sn).
					Int("episodeNumber", en).
					Msg("skipping episode covered by season pack grab")
				continue
			}
		}
		if s.processGroup(ctx, scorer, g) {
			grabbed++
		}
	}

	return grabbed
}

// processGroup scores, selects, and grabs the best release for a single wanted item group.
func (s *Service) processGroup(ctx context.Context, scorer *scoring.Scorer, g *matchGroup) bool {
	profile, err := s.qualityService.Get(ctx, g.item.GetQualityProfileID())
	if err != nil {
		s.logger.Warn().Err(err).Int64("profileID", g.item.GetQualityProfileID()).Msg("failed to load quality profile")
		return false
	}

	s.scoreReleases(scorer, g, profile)

	strategy := s.strategyForItem(g.item)
	parser := s.releaseParser()
	best := decisioning.SelectBestRelease(g.releases, profile, g.item, strategy, parser, s.logger)
	if best == nil {
		return false
	}

	if s.hasRecentGrabForGroup(ctx, g) {
		s.logger.Debug().
			Str("title", best.Title).
			Str("mediaType", g.item.GetMediaType()).
			Int64("mediaID", g.item.GetEntityID()).
			Msg("skipping: recently grabbed")
		return false
	}

	mediaType := decisioning.MediaType(g.item.GetMediaType())
	lockKey := decisioning.Key(mediaType, g.item.GetEntityID())
	if !s.grabLock.TryAcquire(lockKey) {
		s.logger.Debug().Str("key", lockKey).Msg("skipping: grab lock held")
		return false
	}
	defer s.grabLock.Release(lockKey)

	return s.executeGrab(ctx, g, best)
}

// strategyForItem returns the module's SearchStrategy for the given item.
func (s *Service) strategyForItem(item module.SearchableItem) module.SearchStrategy {
	if s.registry != nil {
		if mod := s.registry.Get(module.Type(item.GetModuleType())); mod != nil {
			return mod
		}
	}
	return decisioning.DefaultStrategy{}
}

// releaseParser returns a function that converts release titles to ReleaseForFilter.
func (s *Service) releaseParser() decisioning.ReleaseParser {
	if s.registry != nil {
		return s.registry.ParseReleaseForFilter
	}
	return decisioning.FallbackReleaseParser
}

func (s *Service) scoreReleases(scorer *scoring.Scorer, g *matchGroup, profile *quality.Profile) {
	scoringCtx := scoring.ScoringContext{QualityProfile: profile}
	sp := g.item.GetSearchParams()
	if g.item.GetMediaType() == mediaTypeMovie {
		if y, ok := sp.Extra["year"].(int); ok {
			scoringCtx.SearchYear = y
		}
	} else {
		if sn, ok := sp.Extra["seasonNumber"].(int); ok {
			scoringCtx.SearchSeason = sn
		}
		if en, ok := sp.Extra["episodeNumber"].(int); ok {
			scoringCtx.SearchEpisode = en
		}
	}

	for i := range g.releases {
		scorer.ScoreTorrent(&g.releases[i], &scoringCtx)
	}

	sort.Slice(g.releases, func(i, j int) bool {
		return g.releases[i].Score > g.releases[j].Score
	})
}

func (s *Service) hasRecentGrabForGroup(ctx context.Context, g *matchGroup) bool {
	hasRecent, err := s.queries.HasRecentGrab(ctx, sqlc.HasRecentGrabParams{
		EntityType: g.item.GetMediaType(),
		EntityID:   g.item.GetEntityID(),
	})
	if err != nil {
		s.logger.Warn().Err(err).Msg("failed to check recent grab history")
	}

	sp := g.item.GetSearchParams()
	if !hasRecent && g.isSeason {
		seriesID, _ := sp.Extra["seriesId"].(int64)
		season, _ := sp.Extra["seasonNumber"].(int)
		hasRecent, err = s.queries.HasRecentSeasonGrab(ctx, sqlc.HasRecentSeasonGrabParams{
			SeriesID:     seriesID,
			SeasonNumber: int64(season),
		})
		if err != nil {
			s.logger.Warn().Err(err).Msg("failed to check recent season grab history")
		}
	}

	if !hasRecent && g.item.GetMediaType() == mediaTypeEpisode {
		seriesID, _ := sp.Extra["seriesId"].(int64)
		hasRecent, err = s.queries.HasRecentGrab(ctx, sqlc.HasRecentGrabParams{
			EntityType: mediaTypeSeason,
			EntityID:   seriesID,
		})
		if err != nil {
			s.logger.Warn().Err(err).Msg("failed to check recent season-level grab history")
		}
	}

	return hasRecent
}

func (s *Service) executeGrab(ctx context.Context, g *matchGroup, best *types.TorrentInfo) bool {
	req := s.buildGrabRequest(g, best)

	result, err := s.grabService.Grab(ctx, req)
	if err != nil {
		s.logger.Warn().Err(err).Str("title", best.Title).Msg("RSS grab failed")
		s.logGrabFailed(ctx, g.item, err.Error())
		return false
	}

	if !result.Success {
		s.logger.Warn().Str("title", best.Title).Msg("RSS grab unsuccessful")
		s.logGrabFailed(ctx, g.item, "grab unsuccessful")
		return false
	}

	currentQID := g.item.GetCurrentQualityID()
	isUpgrade := currentQID != nil && *currentQID > 0

	s.logger.Info().
		Str("title", best.Title).
		Str("mediaType", g.item.GetMediaType()).
		Int64("mediaID", g.item.GetEntityID()).
		Str("quality", best.Quality).
		Bool("isUpgrade", isUpgrade).
		Msg("RSS sync grabbed release")

	s.logGrabSuccess(ctx, g.item, best, result, isUpgrade)
	return true
}

func (s *Service) buildGrabRequest(g *matchGroup, best *types.TorrentInfo) *grab.GrabRequest {
	sp := g.item.GetSearchParams()
	req := &grab.GrabRequest{
		Release:   &best.ReleaseInfo,
		Source:    "rss-sync",
		MediaType: g.item.GetMediaType(),
		MediaID:   g.item.GetEntityID(),
	}

	// TargetSlotID from search params (if present)
	if slotID, ok := sp.Extra["targetSlotId"].(*int64); ok {
		req.TargetSlotID = slotID
	}

	seriesID, _ := sp.Extra["seriesId"].(int64)
	season, _ := sp.Extra["seasonNumber"].(int)

	if g.isSeason {
		req.IsSeasonPack = true
		req.SeriesID = seriesID
		req.SeasonNumber = season
	} else if g.item.GetMediaType() == mediaTypeEpisode {
		req.SeriesID = seriesID
		req.SeasonNumber = season
	}

	return req
}

func (s *Service) setStatus(status *SyncStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.status = *status
}

func (s *Service) failSync(start time.Time, errMsg string) {
	s.setStatus(&SyncStatus{
		LastRun: start,
		Error:   errMsg,
	})
	s.broadcast(EventFailed, FailedEvent{Error: errMsg})
}

func (s *Service) reportFeedHealth(indexerName string, err error) {
	if s.healthService == nil {
		return
	}
	id := "rss-" + indexerName
	if err != nil {
		s.healthService.SetWarning(health.CategoryIndexers, id, "RSS feed fetch failed: "+err.Error())
	} else {
		s.healthService.ClearStatus(health.CategoryIndexers, id)
	}
}

func (s *Service) broadcast(eventType string, payload interface{}) {
	if s.hub == nil {
		return
	}
	s.hub.Broadcast(eventType, payload)
}

func (s *Service) logGrabSuccess(ctx context.Context, item module.SearchableItem, release *types.TorrentInfo, grabResult *grab.GrabResult, isUpgrade bool) {
	if s.historyService == nil {
		return
	}

	mediaType := mapHistoryMediaType(item.GetMediaType())

	qualityStr := ""
	if release.ScoreBreakdown != nil {
		qualityStr = release.ScoreBreakdown.QualityName
	}

	data := history.AutoSearchDownloadData{
		ReleaseName: release.Title,
		Indexer:     release.IndexerName,
		ClientName:  grabResult.ClientName,
		DownloadID:  grabResult.DownloadID,
		Source:      "rss-sync",
		IsUpgrade:   isUpgrade,
	}
	if isUpgrade {
		data.NewQuality = qualityStr
	}
	sp := item.GetSearchParams()
	if slotID, ok := sp.Extra["targetSlotId"].(*int64); ok {
		data.SlotID = slotID
	}

	if err := s.historyService.LogAutoSearchDownload(ctx, mediaType, item.GetEntityID(), qualityStr, &data); err != nil {
		s.logger.Warn().Err(err).Msg("failed to log RSS sync grab history")
	}
}

func (s *Service) logGrabFailed(ctx context.Context, item module.SearchableItem, errMsg string) {
	if s.historyService == nil {
		return
	}

	mediaType := mapHistoryMediaType(item.GetMediaType())

	if err := s.historyService.LogAutoSearchFailed(ctx, mediaType, item.GetEntityID(), history.AutoSearchFailedData{
		Error:  errMsg,
		Source: "rss-sync",
	}); err != nil {
		s.logger.Warn().Err(err).Msg("failed to log RSS sync failure history")
	}
}

func mapHistoryMediaType(mediaType string) history.MediaType {
	switch mediaType {
	case mediaTypeSeason, "series":
		return history.MediaTypeSeason
	case mediaTypeEpisode:
		return history.MediaTypeEpisode
	default:
		return history.MediaTypeMovie
	}
}
