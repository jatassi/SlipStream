package rsssync

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/indexer/types"
)

var ErrNoCacheBoundary = errors.New("no cache boundary")

// ProwlarrAggregatedID is a sentinel value indicating the Prowlarr aggregated feed.
// Cache boundaries for Prowlarr are stored in the settings table (not indexer_status)
// because indexer_status has a FK to the indexers table.
const ProwlarrAggregatedID int64 = 0

const prowlarrCacheKey = "prowlarr_rss_cache_boundary"

// CacheBoundary represents the last-seen release for an indexer.
type CacheBoundary struct {
	URL  string
	Date sql.NullTime
}

// prowlarrCacheJSON is the JSON-serializable form stored in the settings table.
type prowlarrCacheJSON struct {
	URL  string `json:"url"`
	Date string `json:"date,omitempty"`
}

// GetCacheBoundary retrieves the cache boundary for an indexer.
func GetCacheBoundary(ctx context.Context, queries *sqlc.Queries, indexerID int64) (*CacheBoundary, error) {
	if indexerID == ProwlarrAggregatedID {
		return getProwlarrCacheBoundary(ctx, queries)
	}

	row, err := queries.GetIndexerRssCache(ctx, indexerID)
	if err != nil {
		return nil, err
	}
	if !row.LastRssReleaseUrl.Valid || row.LastRssReleaseUrl.String == "" {
		return nil, ErrNoCacheBoundary
	}
	return &CacheBoundary{
		URL:  row.LastRssReleaseUrl.String,
		Date: row.LastRssReleaseDate,
	}, nil
}

// UpdateCacheBoundary stores the newest release as the cache boundary after a successful sync.
func UpdateCacheBoundary(ctx context.Context, queries *sqlc.Queries, indexerID int64, newest *types.TorrentInfo) error {
	if newest == nil {
		return nil
	}

	if indexerID == ProwlarrAggregatedID {
		return updateProwlarrCacheBoundary(ctx, queries, newest)
	}

	return queries.UpdateIndexerRssCache(ctx, sqlc.UpdateIndexerRssCacheParams{
		IndexerID:          indexerID,
		LastRssReleaseUrl:  sql.NullString{String: newest.DownloadURL, Valid: newest.DownloadURL != ""},
		LastRssReleaseDate: sql.NullTime{Time: newest.PublishDate, Valid: !newest.PublishDate.IsZero()},
	})
}

// IsAtCacheBoundary checks whether a release matches the cache boundary.
func IsAtCacheBoundary(release *types.TorrentInfo, boundary *CacheBoundary) bool {
	if boundary == nil {
		return false
	}
	if release.DownloadURL == boundary.URL {
		if !boundary.Date.Valid {
			return true
		}
		return !release.PublishDate.After(boundary.Date.Time)
	}
	return false
}

func getProwlarrCacheBoundary(ctx context.Context, queries *sqlc.Queries) (*CacheBoundary, error) {
	row, getErr := queries.GetSetting(ctx, prowlarrCacheKey)
	if getErr != nil {
		return nil, ErrNoCacheBoundary
	}

	var cached prowlarrCacheJSON
	if unmarshalErr := json.Unmarshal([]byte(row.Value), &cached); unmarshalErr != nil {
		return nil, ErrNoCacheBoundary
	}
	if cached.URL == "" {
		return nil, ErrNoCacheBoundary
	}

	boundary := &CacheBoundary{URL: cached.URL}
	if cached.Date != "" {
		if t, err := time.Parse(time.RFC3339, cached.Date); err == nil {
			boundary.Date = sql.NullTime{Time: t, Valid: true}
		}
	}
	return boundary, nil
}

func updateProwlarrCacheBoundary(ctx context.Context, queries *sqlc.Queries, newest *types.TorrentInfo) error {
	cached := prowlarrCacheJSON{
		URL: newest.DownloadURL,
	}
	if !newest.PublishDate.IsZero() {
		cached.Date = newest.PublishDate.Format(time.RFC3339)
	}

	data, err := json.Marshal(cached)
	if err != nil {
		return err
	}

	_, err = queries.SetSetting(ctx, sqlc.SetSettingParams{
		Key:   prowlarrCacheKey,
		Value: string(data),
	})
	return err
}
