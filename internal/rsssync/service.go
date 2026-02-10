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
	"github.com/slipstream/slipstream/internal/websocket"
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
	logger         zerolog.Logger

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
	logger zerolog.Logger,
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

	// 1. Fetch feeds from all RSS-enabled indexers
	feeds := s.fetcher.FetchAll(ctx)
	if len(feeds) == 0 {
		s.logger.Info().Msg("no RSS feeds fetched")
		s.setStatus(SyncStatus{LastRun: start})
		return nil
	}

	s.broadcast(EventStarted, StartedEvent{IndexerCount: len(feeds)})

	// 2. Collect wanted items
	collector := &decisioning.Collector{
		Queries:        s.queries,
		Logger:         s.logger,
		BackoffChecker: decisioning.NoBackoff{},
	}
	wantedItems, err := decisioning.CollectWantedItems(ctx, collector)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to collect wanted items")
		s.failSync(start, err.Error())
		return err
	}
	if len(wantedItems) == 0 {
		s.logger.Info().Msg("no wanted items, skipping RSS sync")
		s.setStatus(SyncStatus{LastRun: start})
		s.broadcast(EventCompleted, CompletedEvent{ElapsedMs: int(time.Since(start).Milliseconds())})
		return nil
	}

	// 3. Build WantedIndex
	index := BuildWantedIndex(wantedItems)
	s.logger.Info().Int("wantedItems", len(wantedItems)).Msg("built wanted index")

	// 4. Process each feed
	var (
		totalReleases int
		allMatches    []MatchResult
	)

	for _, feed := range feeds {
		if feed.Error != nil {
			s.reportFeedHealth(feed.IndexerName, feed.Error)
			continue
		}

		matcher := NewMatcher(index, s.queries, s.logger)
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

	s.logger.Info().
		Int("totalReleases", totalReleases).
		Int("matched", len(allMatches)).
		Msg("RSS matching complete")

	// 5. Group matches by target wanted item
	groups := groupMatches(allMatches)

	// 6. Score, select, and grab
	grabbed := s.scoreAndGrab(ctx, groups)

	// 7. Update cache boundaries for successful feeds
	for _, feed := range feeds {
		if feed.Error != nil || len(feed.Releases) == 0 {
			continue
		}
		newest := &feed.Releases[0]
		if err := UpdateCacheBoundary(ctx, s.queries, feed.IndexerID, newest); err != nil {
			s.logger.Warn().Err(err).Int64("indexerID", feed.IndexerID).Msg("failed to update RSS cache boundary")
		}
	}

	elapsed := int(time.Since(start).Milliseconds())
	s.setStatus(SyncStatus{
		LastRun:       start,
		TotalReleases: totalReleases,
		Matched:       len(allMatches),
		Grabbed:       grabbed,
		ElapsedMs:     elapsed,
	})

	s.broadcast(EventCompleted, CompletedEvent{
		TotalReleases: totalReleases,
		Matched:       len(allMatches),
		Grabbed:       grabbed,
		ElapsedMs:     elapsed,
	})

	s.logger.Info().
		Int("totalReleases", totalReleases).
		Int("matched", len(allMatches)).
		Int("grabbed", grabbed).
		Int("elapsedMs", elapsed).
		Msg("RSS sync completed")

	return nil
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

		results := matcher.Match(ctx, *release)
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
	item     decisioning.SearchableItem
	releases []types.TorrentInfo
	isSeason bool
}

// groupMatches groups match results by their target wanted item.
func groupMatches(matches []MatchResult) map[string]*matchGroup {
	groups := make(map[string]*matchGroup)
	for _, m := range matches {
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
		if g.item.MediaType == decisioning.MediaTypeEpisode {
			sKey := seasonKeyForEpisode(g.item)
			if seasonGrabs[sKey] {
				s.logger.Debug().
					Str("title", g.item.Title).
					Int("season", g.item.SeasonNumber).
					Int("episodeNumber", g.item.EpisodeNumber).
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
	profile, err := s.qualityService.Get(ctx, g.item.QualityProfileID)
	if err != nil {
		s.logger.Warn().Err(err).Int64("profileID", g.item.QualityProfileID).Msg("failed to load quality profile")
		return false
	}

	// Score all releases
	scoringCtx := scoring.ScoringContext{
		QualityProfile: profile,
	}
	if g.item.MediaType == decisioning.MediaTypeMovie {
		scoringCtx.SearchYear = g.item.Year
	} else {
		scoringCtx.SearchSeason = g.item.SeasonNumber
		scoringCtx.SearchEpisode = g.item.EpisodeNumber
	}

	for i := range g.releases {
		scorer.ScoreTorrent(&g.releases[i], scoringCtx)
	}

	// Sort by score (highest first)
	sort.Slice(g.releases, func(i, j int) bool {
		return g.releases[i].Score > g.releases[j].Score
	})

	// Select best acceptable release
	best := decisioning.SelectBestRelease(g.releases, profile, g.item, s.logger)
	if best == nil {
		return false
	}

	// History check: reject if recently grabbed
	hasRecent, err := s.queries.HasRecentGrab(ctx, sqlc.HasRecentGrabParams{
		MediaType: string(g.item.MediaType),
		MediaID:   g.item.MediaID,
	})
	if err != nil {
		s.logger.Warn().Err(err).Msg("failed to check recent grab history")
	}
	// For season packs, also check if any individual episode in the season was recently grabbed.
	// For individual episodes, also check if a season-level grab occurred.
	if !hasRecent && g.isSeason {
		hasRecent, err = s.queries.HasRecentSeasonGrab(ctx, sqlc.HasRecentSeasonGrabParams{
			SeriesID:     g.item.SeriesID,
			SeasonNumber: int64(g.item.SeasonNumber),
		})
		if err != nil {
			s.logger.Warn().Err(err).Msg("failed to check recent season grab history")
		}
	}
	if !hasRecent && g.item.MediaType == decisioning.MediaTypeEpisode {
		hasRecent, err = s.queries.HasRecentGrab(ctx, sqlc.HasRecentGrabParams{
			MediaType: "season",
			MediaID:   g.item.SeriesID,
		})
		if err != nil {
			s.logger.Warn().Err(err).Msg("failed to check recent season-level grab history")
		}
	}
	if hasRecent {
		s.logger.Debug().
			Str("title", best.Title).
			Str("mediaType", string(g.item.MediaType)).
			Int64("mediaID", g.item.MediaID).
			Msg("skipping: recently grabbed")
		return false
	}

	// Acquire grab lock
	lockKey := decisioning.Key(g.item.MediaType, g.item.MediaID)
	if !s.grabLock.TryAcquire(lockKey) {
		s.logger.Debug().Str("key", lockKey).Msg("skipping: grab lock held")
		return false
	}
	defer s.grabLock.Release(lockKey)

	// Build grab request
	req := grab.GrabRequest{
		Release:   &best.ReleaseInfo,
		Source:    "rss-sync",
		MediaType: string(g.item.MediaType),
		MediaID:   g.item.MediaID,
	}

	if g.isSeason {
		req.IsSeasonPack = true
		req.SeriesID = g.item.SeriesID
		req.SeasonNumber = g.item.SeasonNumber
	} else if g.item.MediaType == decisioning.MediaTypeEpisode {
		req.SeriesID = g.item.SeriesID
		req.SeasonNumber = g.item.SeasonNumber
	}

	if g.item.TargetSlotID != nil {
		req.TargetSlotID = g.item.TargetSlotID
	}

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

	isUpgrade := g.item.HasFile && g.item.CurrentQualityID > 0

	s.logger.Info().
		Str("title", best.Title).
		Str("mediaType", string(g.item.MediaType)).
		Int64("mediaID", g.item.MediaID).
		Str("quality", best.Quality).
		Bool("isUpgrade", isUpgrade).
		Msg("RSS sync grabbed release")

	s.logGrabSuccess(ctx, g.item, best, result, isUpgrade)

	return true
}

func (s *Service) setStatus(status SyncStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.status = status
}

func (s *Service) failSync(start time.Time, errMsg string) {
	s.setStatus(SyncStatus{
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
	if err := s.hub.Broadcast(eventType, payload); err != nil {
		s.logger.Warn().Err(err).Str("event", eventType).Msg("failed to broadcast RSS sync event")
	}
}

func (s *Service) logGrabSuccess(ctx context.Context, item decisioning.SearchableItem, release *types.TorrentInfo, grabResult *grab.GrabResult, isUpgrade bool) {
	if s.historyService == nil {
		return
	}

	mediaType := history.MediaTypeMovie
	if item.MediaType == decisioning.MediaTypeEpisode || item.MediaType == decisioning.MediaTypeSeason {
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
		Source:      "rss-sync",
		IsUpgrade:   isUpgrade,
	}
	if isUpgrade {
		data.NewQuality = qualityStr
	}
	if item.TargetSlotID != nil {
		data.SlotID = item.TargetSlotID
	}

	if err := s.historyService.LogAutoSearchDownload(ctx, mediaType, item.MediaID, qualityStr, data); err != nil {
		s.logger.Warn().Err(err).Msg("failed to log RSS sync grab history")
	}
}

func (s *Service) logGrabFailed(ctx context.Context, item decisioning.SearchableItem, errMsg string) {
	if s.historyService == nil {
		return
	}

	mediaType := history.MediaTypeMovie
	if item.MediaType == decisioning.MediaTypeEpisode || item.MediaType == decisioning.MediaTypeSeason {
		mediaType = history.MediaTypeEpisode
	}

	if err := s.historyService.LogAutoSearchFailed(ctx, mediaType, item.MediaID, history.AutoSearchFailedData{
		Error:  errMsg,
		Source: "rss-sync",
	}); err != nil {
		s.logger.Warn().Err(err).Msg("failed to log RSS sync failure history")
	}
}

