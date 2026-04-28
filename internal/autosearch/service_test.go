package autosearch

import (
	"context"
	"database/sql"
	"testing"

	"github.com/slipstream/slipstream/internal/config"
	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/library/status"
	"github.com/slipstream/slipstream/internal/testutil"
)

// TestSearchMovie_SkipsUnreleased verifies that SearchMovie short-circuits when
// the movie is in `unreleased` status, without invoking the search service.
// This guards against the regression where a portal request for an in-theater
// movie triggered an auto-search that grabbed a TELESYNC.
func TestSearchMovie_SkipsUnreleased(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	t.Cleanup(tdb.Close)

	queries := sqlc.New(tdb.Conn)
	ctx := context.Background()

	movie, err := queries.CreateMovie(ctx, sqlc.CreateMovieParams{
		Title:     "Michael",
		SortTitle: "michael",
		Year:      sql.NullInt64{Int64: 2026, Valid: true},
		TmdbID:    sql.NullInt64{Int64: 936075, Valid: true},
		Monitored: true,
		Status:    status.Unreleased,
	})
	if err != nil {
		t.Fatalf("create movie: %v", err)
	}

	// nil search/grab/quality services prove the unreleased short-circuit fires
	// before any search work — invoking them would nil-panic.
	svc := NewService(tdb.Conn, nil, nil, nil, &tdb.Logger, nil, nil, nil)
	cfg := &config.AutoSearchConfig{BackoffThreshold: 5, BaseDelayMs: 100}
	_ = NewScheduledSearcher(svc, cfg, &tdb.Logger)

	result, err := svc.SearchMovie(ctx, movie.ID, SearchSourceRequest)
	if err != nil {
		t.Fatalf("SearchMovie returned error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Found {
		t.Errorf("expected Found=false for unreleased movie, got Found=true")
	}
	if result.Downloaded {
		t.Errorf("expected Downloaded=false for unreleased movie, got Downloaded=true")
	}
}
