package rsssync

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/indexer/types"
	"github.com/slipstream/slipstream/internal/testutil"
)

func TestGetCacheBoundary_NoCacheExists(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()
	q := sqlc.New(tdb.Conn)
	ctx := context.Background()

	// Create an indexer so indexer_status can reference it
	createTestIndexer(t, q, 1, "TestIndexer")

	boundary, err := GetCacheBoundary(ctx, q, 1)
	// sql.ErrNoRows is expected when no indexer_status row exists
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("GetCacheBoundary: unexpected error: %v", err)
	}
	if boundary != nil {
		t.Errorf("expected nil boundary for fresh indexer, got %+v", boundary)
	}
}

func TestUpdateAndGetCacheBoundary(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()
	q := sqlc.New(tdb.Conn)
	ctx := context.Background()

	createTestIndexer(t, q, 1, "TestIndexer")

	pubDate := time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC)
	release := &types.TorrentInfo{
		ReleaseInfo: types.ReleaseInfo{
			DownloadURL: "https://example.com/release1.torrent",
			PublishDate: pubDate,
		},
	}

	if err := UpdateCacheBoundary(ctx, q, 1, release); err != nil {
		t.Fatalf("UpdateCacheBoundary: %v", err)
	}

	boundary, err := GetCacheBoundary(ctx, q, 1)
	if err != nil {
		t.Fatalf("GetCacheBoundary: %v", err)
	}
	if boundary == nil {
		t.Fatal("expected non-nil boundary after update")
	}
	if boundary.URL != "https://example.com/release1.torrent" {
		t.Errorf("expected URL 'https://example.com/release1.torrent', got %q", boundary.URL)
	}
	if !boundary.Date.Valid {
		t.Error("expected valid date")
	}
}

func TestIsAtCacheBoundary_MatchingURL(t *testing.T) {
	boundary := &CacheBoundary{
		URL:  "https://example.com/release1.torrent",
		Date: sql.NullTime{Time: time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC), Valid: true},
	}

	release := &types.TorrentInfo{
		ReleaseInfo: types.ReleaseInfo{
			DownloadURL: "https://example.com/release1.torrent",
			PublishDate: time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC),
		},
	}

	if !IsAtCacheBoundary(release, boundary) {
		t.Error("expected IsAtCacheBoundary=true for matching URL and date")
	}
}

func TestIsAtCacheBoundary_DifferentURL(t *testing.T) {
	boundary := &CacheBoundary{
		URL:  "https://example.com/release1.torrent",
		Date: sql.NullTime{Time: time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC), Valid: true},
	}

	release := &types.TorrentInfo{
		ReleaseInfo: types.ReleaseInfo{
			DownloadURL: "https://example.com/different.torrent",
			PublishDate: time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC),
		},
	}

	if IsAtCacheBoundary(release, boundary) {
		t.Error("expected IsAtCacheBoundary=false for different URL")
	}
}

func TestIsAtCacheBoundary_NilBoundary(t *testing.T) {
	release := &types.TorrentInfo{
		ReleaseInfo: types.ReleaseInfo{
			DownloadURL: "https://example.com/release1.torrent",
		},
	}

	if IsAtCacheBoundary(release, nil) {
		t.Error("expected IsAtCacheBoundary=false for nil boundary")
	}
}

func TestIsAtCacheBoundary_URLMatchNoDate(t *testing.T) {
	boundary := &CacheBoundary{
		URL:  "https://example.com/release1.torrent",
		Date: sql.NullTime{Valid: false},
	}

	release := &types.TorrentInfo{
		ReleaseInfo: types.ReleaseInfo{
			DownloadURL: "https://example.com/release1.torrent",
		},
	}

	if !IsAtCacheBoundary(release, boundary) {
		t.Error("expected IsAtCacheBoundary=true for matching URL with no date")
	}
}

func TestIsAtCacheBoundary_URLMatchReleaseDateAfterBoundary(t *testing.T) {
	boundary := &CacheBoundary{
		URL:  "https://example.com/release1.torrent",
		Date: sql.NullTime{Time: time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC), Valid: true},
	}

	// Release date is after boundary date — this still matches because URL matches and
	// the function checks !release.PublishDate.After(boundary.Date.Time)
	release := &types.TorrentInfo{
		ReleaseInfo: types.ReleaseInfo{
			DownloadURL: "https://example.com/release1.torrent",
			PublishDate: time.Date(2025, 1, 16, 12, 0, 0, 0, time.UTC),
		},
	}

	// Same URL but date is after boundary → should not be at boundary
	if IsAtCacheBoundary(release, boundary) {
		t.Error("expected IsAtCacheBoundary=false when release date is after boundary date")
	}
}

func TestUpdateCacheBoundary_NilRelease(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()
	q := sqlc.New(tdb.Conn)
	ctx := context.Background()

	if err := UpdateCacheBoundary(ctx, q, 1, nil); err != nil {
		t.Fatalf("UpdateCacheBoundary with nil release should not error: %v", err)
	}
}

// createTestIndexer inserts a minimal indexer so indexer_status can reference it.
func createTestIndexer(t *testing.T, q *sqlc.Queries, id int64, name string) {
	t.Helper()
	_, err := q.CreateIndexer(context.Background(), sqlc.CreateIndexerParams{
		Name:           name,
		DefinitionID:   "test",
		SupportsMovies: 1,
		SupportsTv:     1,
		Priority:       25,
		Enabled:        1,
		RssEnabled:     1,
	})
	if err != nil {
		t.Fatalf("create indexer: %v", err)
	}
}
