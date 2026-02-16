package rsssync

import (
	"context"
	"fmt"
	"sync"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/indexer"
	"github.com/slipstream/slipstream/internal/indexer/genericrss"
	indexerTypes "github.com/slipstream/slipstream/internal/indexer/types"
	"github.com/slipstream/slipstream/internal/prowlarr"
)

const (
	maxResultsPerIndexer     = 1000
	rssBackoffThreshold  int = 3
)

// IndexerFeed holds the fetched releases from a single indexer.
type IndexerFeed struct {
	IndexerID   int64
	IndexerName string
	Releases    []indexerTypes.TorrentInfo
	Error       error
}

// FeedFetcher fetches RSS feeds from indexers.
type FeedFetcher struct {
	indexerService  *indexer.Service
	prowlarrService *prowlarr.Service
	modeManager     *prowlarr.ModeManager
	queries         *sqlc.Queries
	logger          *zerolog.Logger

	// In-memory backoff: consecutive failure counts per indexer ID.
	// Resets on successful fetch or server restart.
	failMu        sync.Mutex
	failureCounts map[int64]int
}

// NewFeedFetcher creates a new FeedFetcher.
func NewFeedFetcher(
	indexerService *indexer.Service,
	prowlarrService *prowlarr.Service,
	modeManager *prowlarr.ModeManager,
	queries *sqlc.Queries,
	logger *zerolog.Logger,
) *FeedFetcher {
	return &FeedFetcher{
		indexerService:  indexerService,
		prowlarrService: prowlarrService,
		modeManager:     modeManager,
		queries:         queries,
		logger:          logger,
	}
}

// FetchAll fetches feeds from all RSS-enabled indexers.
func (f *FeedFetcher) FetchAll(ctx context.Context) []IndexerFeed {
	isProwlarr, err := f.modeManager.IsProwlarrMode(ctx)
	if err != nil {
		f.logger.Error().Err(err).Msg("failed to determine indexer mode")
		return nil
	}

	if isProwlarr {
		feeds := f.fetchProwlarr(ctx)
		// Also fetch generic-rss indexers (not managed by Prowlarr)
		feeds = append(feeds, f.fetchGenericRSS(ctx)...)
		return feeds
	}
	return f.fetchNative(ctx)
}

// fetchNative fetches feeds from native (Cardigann) indexers.
func (f *FeedFetcher) fetchNative(ctx context.Context) []IndexerFeed {
	indexers, err := f.queries.ListRssEnabledIndexers(ctx)
	if err != nil {
		f.logger.Error().Err(err).Msg("failed to list RSS-enabled indexers")
		return nil
	}

	var feeds []IndexerFeed
	for _, idx := range indexers {
		if f.isBackedOff(idx.ID) {
			f.logger.Debug().Str("indexer", idx.Name).Msg("skipping RSS fetch: indexer backed off due to repeated failures")
			feeds = append(feeds, IndexerFeed{
				IndexerID:   idx.ID,
				IndexerName: idx.Name,
				Error:       fmt.Errorf("backed off after %d consecutive failures", rssBackoffThreshold),
			})
			continue
		}

		feed := f.fetchNativeIndexer(ctx, idx)
		if feed.Error != nil {
			f.recordFailure(idx.ID)
		} else {
			f.recordSuccess(idx.ID)
		}
		feeds = append(feeds, feed)
	}
	return feeds
}

// fetchNativeIndexer fetches the RSS feed from a single native indexer.
func (f *FeedFetcher) fetchNativeIndexer(ctx context.Context, idx *sqlc.Indexer) IndexerFeed {
	feed := IndexerFeed{
		IndexerID:   idx.ID,
		IndexerName: idx.Name,
	}

	client, err := f.indexerService.GetClient(ctx, idx.ID)
	if err != nil {
		feed.Error = fmt.Errorf("failed to get indexer client: %w", err)
		f.logger.Warn().Err(err).Str("indexer", idx.Name).Msg("RSS fetch: failed to get client")
		return feed
	}

	// Fetch with empty criteria to get recent items (RSS mode)
	criteria := indexerTypes.SearchCriteria{
		Type:  "search",
		Limit: maxResultsPerIndexer,
	}

	// Try torrent-specific search first
	if ti, ok := client.(indexer.TorrentIndexer); ok {
		results, err := ti.SearchTorrents(ctx, &criteria)
		if err != nil {
			feed.Error = fmt.Errorf("RSS fetch failed: %w", err)
			f.logger.Warn().Err(err).Str("indexer", idx.Name).Msg("RSS fetch failed")
			return feed
		}
		feed.Releases = results
	} else {
		results, err := client.Search(ctx, &criteria)
		if err != nil {
			feed.Error = fmt.Errorf("RSS fetch failed: %w", err)
			f.logger.Warn().Err(err).Str("indexer", idx.Name).Msg("RSS fetch failed")
			return feed
		}
		for i := range results {
			r := &results[i]
			feed.Releases = append(feed.Releases, indexerTypes.TorrentInfo{ReleaseInfo: *r})
		}
	}

	if len(feed.Releases) > maxResultsPerIndexer {
		feed.Releases = feed.Releases[:maxResultsPerIndexer]
	}

	f.logger.Info().Str("indexer", idx.Name).Int("releases", len(feed.Releases)).Msg("RSS feed fetched")
	return feed
}

// fetchProwlarr fetches feeds from Prowlarr-managed indexers.
func (f *FeedFetcher) fetchProwlarr(ctx context.Context) []IndexerFeed {
	if f.prowlarrService == nil {
		f.logger.Warn().Msg("Prowlarr service not available for RSS fetch")
		return nil
	}

	// Clear the search cache to avoid stale results from auto-search
	f.prowlarrService.ClearSearchCache()

	// Prowlarr aggregates all indexers — make two calls for broad coverage:
	// one for movies, one for TV.
	var feeds []IndexerFeed

	// Movie feed
	movieResults, err := f.prowlarrService.Search(ctx, &indexerTypes.SearchCriteria{
		Type:  "movie",
		Limit: maxResultsPerIndexer,
	})
	if err != nil {
		f.logger.Warn().Err(err).Msg("Prowlarr RSS movie fetch failed")
	}

	// TV feed
	tvResults, err := f.prowlarrService.Search(ctx, &indexerTypes.SearchCriteria{
		Type:  "tvsearch",
		Limit: maxResultsPerIndexer,
	})
	if err != nil {
		f.logger.Warn().Err(err).Msg("Prowlarr RSS TV fetch failed")
	}

	// Combine into a single feed since Prowlarr aggregates across indexers
	combined := make([]indexerTypes.TorrentInfo, 0, len(movieResults)+len(tvResults))
	combined = append(combined, movieResults...)
	combined = append(combined, tvResults...)

	seen := make(map[string]bool)
	var deduped []indexerTypes.TorrentInfo
	for i := range combined {
		r := &combined[i]
		key := r.DownloadURL
		if key == "" {
			key = r.GUID
		}
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		deduped = append(deduped, *r)
	}

	if len(deduped) > 0 {
		feeds = append(feeds, IndexerFeed{
			IndexerID:   0, // Prowlarr aggregated feed
			IndexerName: "Prowlarr",
			Releases:    deduped,
		})
	}

	f.logger.Info().
		Int("movieResults", len(movieResults)).
		Int("tvResults", len(tvResults)).
		Int("combined", len(deduped)).
		Msg("Prowlarr RSS feeds fetched")

	return feeds
}

func (f *FeedFetcher) isBackedOff(indexerID int64) bool {
	f.failMu.Lock()
	defer f.failMu.Unlock()
	return f.failureCounts[indexerID] >= rssBackoffThreshold
}

func (f *FeedFetcher) recordFailure(indexerID int64) {
	f.failMu.Lock()
	defer f.failMu.Unlock()
	if f.failureCounts == nil {
		f.failureCounts = make(map[int64]int)
	}
	f.failureCounts[indexerID]++
}

func (f *FeedFetcher) recordSuccess(indexerID int64) {
	f.failMu.Lock()
	defer f.failMu.Unlock()
	delete(f.failureCounts, indexerID)
}

// ResetBackoff clears the backoff state for all indexers.
func (f *FeedFetcher) ResetBackoff() {
	f.failMu.Lock()
	defer f.failMu.Unlock()
	f.failureCounts = nil
}

// fetchGenericRSS fetches feeds from generic-rss indexers (used in Prowlarr mode
// since Prowlarr doesn't manage these).
func (f *FeedFetcher) fetchGenericRSS(ctx context.Context) []IndexerFeed {
	indexers, err := f.queries.ListRssEnabledIndexers(ctx)
	if err != nil {
		f.logger.Error().Err(err).Msg("failed to list RSS-enabled indexers for generic-rss")
		return nil
	}

	var feeds []IndexerFeed
	for _, idx := range indexers {
		if idx.DefinitionID != genericrss.DefinitionID {
			continue
		}
		if f.isBackedOff(idx.ID) {
			f.logger.Debug().Str("indexer", idx.Name).Msg("skipping generic-rss fetch: indexer backed off due to repeated failures")
			feeds = append(feeds, IndexerFeed{
				IndexerID:   idx.ID,
				IndexerName: idx.Name,
				Error:       fmt.Errorf("backed off after %d consecutive failures", rssBackoffThreshold),
			})
			continue
		}

		feed := f.fetchNativeIndexer(ctx, idx)
		if feed.Error != nil {
			f.recordFailure(idx.ID)
		} else {
			f.recordSuccess(idx.ID)
		}
		feeds = append(feeds, feed)
	}
	return feeds
}
