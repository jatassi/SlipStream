package autosearch

import (
	"context"
	"testing"

	"github.com/slipstream/slipstream/internal/config"
	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/testutil"
)

func newTestSearcher(t *testing.T) (*ScheduledSearcher, *sqlc.Queries) {
	t.Helper()
	tdb := testutil.NewTestDB(t)
	t.Cleanup(tdb.Close)

	svc := NewService(tdb.Conn, nil, nil, nil, tdb.Logger)
	cfg := &config.AutoSearchConfig{BackoffThreshold: 5, BaseDelayMs: 100}
	searcher := NewScheduledSearcher(svc, cfg, tdb.Logger)
	queries := sqlc.New(tdb.Conn)
	return searcher, queries
}

func TestShouldSkipItem(t *testing.T) {
	t.Run("NoRecord", func(t *testing.T) {
		searcher, _ := newTestSearcher(t)
		ctx := context.Background()

		skip, err := searcher.shouldSkipItem(ctx, "movie", 999, "missing")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if skip {
			t.Fatal("expected false for item with no status record")
		}
	})

	t.Run("BelowThreshold", func(t *testing.T) {
		searcher, queries := newTestSearcher(t)
		ctx := context.Background()

		for i := 0; i < 4; i++ {
			if err := queries.IncrementAutosearchFailure(ctx, sqlc.IncrementAutosearchFailureParams{
				ItemType: "movie", ItemID: 1, SearchType: "missing",
			}); err != nil {
				t.Fatalf("increment failed: %v", err)
			}
		}

		skip, err := searcher.shouldSkipItem(ctx, "movie", 1, "missing")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if skip {
			t.Fatal("expected false when failure_count (4) < threshold (5)")
		}
	})

	t.Run("AtThreshold", func(t *testing.T) {
		searcher, queries := newTestSearcher(t)
		ctx := context.Background()

		for i := 0; i < 5; i++ {
			if err := queries.IncrementAutosearchFailure(ctx, sqlc.IncrementAutosearchFailureParams{
				ItemType: "movie", ItemID: 1, SearchType: "missing",
			}); err != nil {
				t.Fatalf("increment failed: %v", err)
			}
		}

		skip, err := searcher.shouldSkipItem(ctx, "movie", 1, "missing")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !skip {
			t.Fatal("expected true when failure_count (5) == threshold (5)")
		}
	})

	t.Run("AboveThreshold", func(t *testing.T) {
		searcher, queries := newTestSearcher(t)
		ctx := context.Background()

		for i := 0; i < 7; i++ {
			if err := queries.IncrementAutosearchFailure(ctx, sqlc.IncrementAutosearchFailureParams{
				ItemType: "movie", ItemID: 1, SearchType: "missing",
			}); err != nil {
				t.Fatalf("increment failed: %v", err)
			}
		}

		skip, err := searcher.shouldSkipItem(ctx, "movie", 1, "missing")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !skip {
			t.Fatal("expected true when failure_count (7) > threshold (5)")
		}
	})

	t.Run("IndependentSearchTypes", func(t *testing.T) {
		searcher, queries := newTestSearcher(t)
		ctx := context.Background()

		// Push "missing" past threshold
		for i := 0; i < 6; i++ {
			if err := queries.IncrementAutosearchFailure(ctx, sqlc.IncrementAutosearchFailureParams{
				ItemType: "movie", ItemID: 1, SearchType: "missing",
			}); err != nil {
				t.Fatalf("increment failed: %v", err)
			}
		}

		// "upgrade" for the same item should be unaffected
		skip, err := searcher.shouldSkipItem(ctx, "movie", 1, "upgrade")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if skip {
			t.Fatal("expected false: 'upgrade' should be independent of 'missing' backoff")
		}
	})
}

func TestIncrementFailureCount(t *testing.T) {
	t.Run("Movie", func(t *testing.T) {
		searcher, queries := newTestSearcher(t)
		ctx := context.Background()

		item := SearchableItem{MediaType: MediaTypeMovie, MediaID: 10}
		searcher.incrementFailureCount(ctx, item, "missing")

		status, err := queries.GetAutosearchStatus(ctx, sqlc.GetAutosearchStatusParams{
			ItemType: "movie", ItemID: 10, SearchType: "missing",
		})
		if err != nil {
			t.Fatalf("failed to get status: %v", err)
		}
		if status.FailureCount != 1 {
			t.Fatalf("expected failure_count=1, got %d", status.FailureCount)
		}
		if status.ItemType != "movie" {
			t.Fatalf("expected item_type='movie', got %q", status.ItemType)
		}
	})

	t.Run("Episode", func(t *testing.T) {
		searcher, queries := newTestSearcher(t)
		ctx := context.Background()

		item := SearchableItem{MediaType: MediaTypeEpisode, MediaID: 20}
		searcher.incrementFailureCount(ctx, item, "missing")

		status, err := queries.GetAutosearchStatus(ctx, sqlc.GetAutosearchStatusParams{
			ItemType: "episode", ItemID: 20, SearchType: "missing",
		})
		if err != nil {
			t.Fatalf("failed to get status: %v", err)
		}
		if status.ItemType != "episode" {
			t.Fatalf("expected item_type='episode', got %q", status.ItemType)
		}
	})

	t.Run("Season", func(t *testing.T) {
		searcher, queries := newTestSearcher(t)
		ctx := context.Background()

		// MediaTypeSeason should map to item_type="series" in the DB
		item := SearchableItem{MediaType: MediaTypeSeason, MediaID: 30}
		searcher.incrementFailureCount(ctx, item, "missing")

		status, err := queries.GetAutosearchStatus(ctx, sqlc.GetAutosearchStatusParams{
			ItemType: "series", ItemID: 30, SearchType: "missing",
		})
		if err != nil {
			t.Fatalf("failed to get status: %v", err)
		}
		if status.ItemType != "series" {
			t.Fatalf("expected item_type='series', got %q", status.ItemType)
		}
		if status.FailureCount != 1 {
			t.Fatalf("expected failure_count=1, got %d", status.FailureCount)
		}
	})

	t.Run("Accumulates", func(t *testing.T) {
		searcher, queries := newTestSearcher(t)
		ctx := context.Background()

		item := SearchableItem{MediaType: MediaTypeMovie, MediaID: 40}
		searcher.incrementFailureCount(ctx, item, "missing")
		searcher.incrementFailureCount(ctx, item, "missing")
		searcher.incrementFailureCount(ctx, item, "missing")

		status, err := queries.GetAutosearchStatus(ctx, sqlc.GetAutosearchStatusParams{
			ItemType: "movie", ItemID: 40, SearchType: "missing",
		})
		if err != nil {
			t.Fatalf("failed to get status: %v", err)
		}
		if status.FailureCount != 3 {
			t.Fatalf("expected failure_count=3, got %d", status.FailureCount)
		}
	})
}

func TestResetFailureCount(t *testing.T) {
	t.Run("ResetsToZero", func(t *testing.T) {
		searcher, queries := newTestSearcher(t)
		ctx := context.Background()

		item := SearchableItem{MediaType: MediaTypeMovie, MediaID: 50}
		// Build up failures
		for i := 0; i < 5; i++ {
			searcher.incrementFailureCount(ctx, item, "missing")
		}

		searcher.resetFailureCount(ctx, item, "missing")

		status, err := queries.GetAutosearchStatus(ctx, sqlc.GetAutosearchStatusParams{
			ItemType: "movie", ItemID: 50, SearchType: "missing",
		})
		if err != nil {
			t.Fatalf("failed to get status: %v", err)
		}
		if status.FailureCount != 0 {
			t.Fatalf("expected failure_count=0 after reset, got %d", status.FailureCount)
		}
	})

	t.Run("SeasonUsesSeriesType", func(t *testing.T) {
		searcher, queries := newTestSearcher(t)
		ctx := context.Background()

		item := SearchableItem{MediaType: MediaTypeSeason, MediaID: 60}
		searcher.incrementFailureCount(ctx, item, "upgrade")
		searcher.incrementFailureCount(ctx, item, "upgrade")

		searcher.resetFailureCount(ctx, item, "upgrade")

		status, err := queries.GetAutosearchStatus(ctx, sqlc.GetAutosearchStatusParams{
			ItemType: "series", ItemID: 60, SearchType: "upgrade",
		})
		if err != nil {
			t.Fatalf("failed to get status: %v", err)
		}
		if status.FailureCount != 0 {
			t.Fatalf("expected failure_count=0 after season reset, got %d", status.FailureCount)
		}
	})

	t.Run("OnlyResetsSpecificSearchType", func(t *testing.T) {
		searcher, queries := newTestSearcher(t)
		ctx := context.Background()

		item := SearchableItem{MediaType: MediaTypeMovie, MediaID: 70}
		// Increment both search types
		for i := 0; i < 3; i++ {
			searcher.incrementFailureCount(ctx, item, "missing")
			searcher.incrementFailureCount(ctx, item, "upgrade")
		}

		// Reset only "missing"
		searcher.resetFailureCount(ctx, item, "missing")

		missingStatus, err := queries.GetAutosearchStatus(ctx, sqlc.GetAutosearchStatusParams{
			ItemType: "movie", ItemID: 70, SearchType: "missing",
		})
		if err != nil {
			t.Fatalf("failed to get missing status: %v", err)
		}
		if missingStatus.FailureCount != 0 {
			t.Fatalf("expected missing failure_count=0, got %d", missingStatus.FailureCount)
		}

		upgradeStatus, err := queries.GetAutosearchStatus(ctx, sqlc.GetAutosearchStatusParams{
			ItemType: "movie", ItemID: 70, SearchType: "upgrade",
		})
		if err != nil {
			t.Fatalf("failed to get upgrade status: %v", err)
		}
		if upgradeStatus.FailureCount != 3 {
			t.Fatalf("expected upgrade failure_count=3 (untouched), got %d", upgradeStatus.FailureCount)
		}
	})
}

func TestBackoffIDConsistency(t *testing.T) {
	t.Run("SeasonIncrementThenCheck", func(t *testing.T) {
		searcher, _ := newTestSearcher(t)
		ctx := context.Background()

		seriesID := int64(100)
		item := SearchableItem{MediaType: MediaTypeSeason, MediaID: seriesID}

		// Push past threshold via incrementFailureCount
		for i := 0; i < 6; i++ {
			searcher.incrementFailureCount(ctx, item, "missing")
		}

		// shouldSkipItem checks "series" + seriesID — must match what increment wrote
		skip, err := searcher.shouldSkipItem(ctx, "series", seriesID, "missing")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !skip {
			t.Fatal("expected skip=true: season increment writes 'series' type, check should find it")
		}
	})

	t.Run("SeasonBackoffIndependentFromEpisodes", func(t *testing.T) {
		searcher, _ := newTestSearcher(t)
		ctx := context.Background()

		seriesID := int64(200)
		episodeID := int64(2001)

		// Push season (series-level) backoff past threshold
		seasonItem := SearchableItem{MediaType: MediaTypeSeason, MediaID: seriesID}
		for i := 0; i < 6; i++ {
			searcher.incrementFailureCount(ctx, seasonItem, "missing")
		}

		// Episode-level check for a different ID should not be affected
		skip, err := searcher.shouldSkipItem(ctx, "episode", episodeID, "missing")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if skip {
			t.Fatal("expected skip=false: episode backoff is independent from series-level backoff")
		}
	})
}
