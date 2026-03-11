package module_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/slipstream/slipstream/internal/module"
	"github.com/slipstream/slipstream/internal/testutil"
)

func TestDeleteEntity_Movie(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()
	ctx := context.Background()
	db := tdb.Conn

	// -- Seed prerequisite rows --
	mustExec(ctx, t, db,
		`INSERT INTO download_clients (name, type, host, port, use_ssl) VALUES ('test', 'qbittorrent', 'localhost', 8080, 0)`)
	mustExec(ctx, t, db,
		`INSERT INTO portal_users (username, password_hash, enabled) VALUES ('test', 'hash', 1)`)

	const targetMovieID int64 = 100
	const otherMovieID int64 = 200

	// -- Insert records for the target movie entity --
	mustExec(ctx, t, db,
		`INSERT INTO download_mappings (client_id, download_id, module_type, entity_type, entity_id) VALUES (1, 'dl-1', 'movie', 'movie', ?)`, targetMovieID)
	mustExec(ctx, t, db,
		`INSERT INTO queue_media (download_mapping_id, module_type, entity_type, entity_id) VALUES (1, 'movie', 'movie', ?)`, targetMovieID)
	mustExec(ctx, t, db,
		`INSERT INTO downloads (title, module_type, entity_type, entity_id) VALUES ('Test Movie', 'movie', 'movie', ?)`, targetMovieID)
	mustExec(ctx, t, db,
		`INSERT INTO history (event_type, module_type, entity_type, entity_id) VALUES ('grabbed', 'movie', 'movie', ?)`, targetMovieID)
	mustExec(ctx, t, db,
		`INSERT INTO autosearch_status (module_type, entity_type, entity_id, search_type) VALUES ('movie', 'movie', ?, 'missing')`, targetMovieID)
	mustExec(ctx, t, db,
		`INSERT INTO import_decisions (source_path, decision, module_type, entity_type, entity_id) VALUES ('/tmp/a.mkv', 'accepted', 'movie', 'movie', ?)`, targetMovieID)
	mustExec(ctx, t, db,
		`INSERT INTO requests (user_id, module_type, entity_type, title, media_id) VALUES (1, 'movie', 'movie', 'Test Movie', ?)`, targetMovieID)

	// -- Insert records for a different movie (should NOT be deleted) --
	mustExec(ctx, t, db,
		`INSERT INTO download_mappings (client_id, download_id, module_type, entity_type, entity_id) VALUES (1, 'dl-2', 'movie', 'movie', ?)`, otherMovieID)
	mustExec(ctx, t, db,
		`INSERT INTO downloads (title, module_type, entity_type, entity_id) VALUES ('Other Movie', 'movie', 'movie', ?)`, otherMovieID)
	mustExec(ctx, t, db,
		`INSERT INTO history (event_type, module_type, entity_type, entity_id) VALUES ('grabbed', 'movie', 'movie', ?)`, otherMovieID)
	mustExec(ctx, t, db,
		`INSERT INTO autosearch_status (module_type, entity_type, entity_id, search_type) VALUES ('movie', 'movie', ?, 'missing')`, otherMovieID)
	mustExec(ctx, t, db,
		`INSERT INTO import_decisions (source_path, decision, module_type, entity_type, entity_id) VALUES ('/tmp/b.mkv', 'accepted', 'movie', 'movie', ?)`, otherMovieID)
	mustExec(ctx, t, db,
		`INSERT INTO requests (user_id, module_type, entity_type, title, media_id) VALUES (1, 'movie', 'movie', 'Other Movie', ?)`, otherMovieID)

	// -- Delete the target movie --
	if err := module.DeleteEntity(ctx, db, module.TypeMovie, module.EntityMovie, targetMovieID); err != nil {
		t.Fatalf("DeleteEntity failed: %v", err)
	}

	// -- Verify target records are gone --
	assertCount(ctx, t, db, 0, `SELECT COUNT(*) FROM download_mappings WHERE module_type = 'movie' AND entity_type = 'movie' AND entity_id = ?`, targetMovieID)
	assertCount(ctx, t, db, 0, `SELECT COUNT(*) FROM queue_media WHERE module_type = 'movie' AND entity_type = 'movie' AND entity_id = ?`, targetMovieID)
	assertCount(ctx, t, db, 0, `SELECT COUNT(*) FROM downloads WHERE module_type = 'movie' AND entity_type = 'movie' AND entity_id = ?`, targetMovieID)
	assertCount(ctx, t, db, 0, `SELECT COUNT(*) FROM history WHERE module_type = 'movie' AND entity_type = 'movie' AND entity_id = ?`, targetMovieID)
	assertCount(ctx, t, db, 0, `SELECT COUNT(*) FROM autosearch_status WHERE module_type = 'movie' AND entity_type = 'movie' AND entity_id = ?`, targetMovieID)
	assertCount(ctx, t, db, 0, `SELECT COUNT(*) FROM import_decisions WHERE module_type = 'movie' AND entity_type = 'movie' AND entity_id = ?`, targetMovieID)
	assertCount(ctx, t, db, 0, `SELECT COUNT(*) FROM requests WHERE module_type = 'movie' AND entity_type = 'movie' AND media_id = ?`, targetMovieID)

	// -- Verify other movie records are still intact --
	assertCount(ctx, t, db, 1, `SELECT COUNT(*) FROM download_mappings WHERE module_type = 'movie' AND entity_type = 'movie' AND entity_id = ?`, otherMovieID)
	assertCount(ctx, t, db, 1, `SELECT COUNT(*) FROM downloads WHERE module_type = 'movie' AND entity_type = 'movie' AND entity_id = ?`, otherMovieID)
	assertCount(ctx, t, db, 1, `SELECT COUNT(*) FROM history WHERE module_type = 'movie' AND entity_type = 'movie' AND entity_id = ?`, otherMovieID)
	assertCount(ctx, t, db, 1, `SELECT COUNT(*) FROM autosearch_status WHERE module_type = 'movie' AND entity_type = 'movie' AND entity_id = ?`, otherMovieID)
	assertCount(ctx, t, db, 1, `SELECT COUNT(*) FROM import_decisions WHERE module_type = 'movie' AND entity_type = 'movie' AND entity_id = ?`, otherMovieID)
	assertCount(ctx, t, db, 1, `SELECT COUNT(*) FROM requests WHERE module_type = 'movie' AND entity_type = 'movie' AND media_id = ?`, otherMovieID)
}

func TestDeleteEntity_TV_CascadesChildRequests(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()
	ctx := context.Background()
	db := tdb.Conn

	mustExec(ctx, t, db,
		`INSERT INTO download_clients (name, type, host, port, use_ssl) VALUES ('test', 'qbittorrent', 'localhost', 8080, 0)`)
	mustExec(ctx, t, db,
		`INSERT INTO portal_users (username, password_hash, enabled) VALUES ('test', 'hash', 1)`)

	const seriesID int64 = 10
	const episodeID int64 = 50
	const otherSeriesID int64 = 20

	// -- Series-level shared table records --
	mustExec(ctx, t, db,
		`INSERT INTO download_mappings (client_id, download_id, module_type, entity_type, entity_id, is_season_pack, is_complete_series) VALUES (1, 'dl-tv-1', 'tv', 'series', ?, 0, 0)`, seriesID)
	mustExec(ctx, t, db,
		`INSERT INTO history (event_type, module_type, entity_type, entity_id) VALUES ('grabbed', 'tv', 'series', ?)`, seriesID)
	mustExec(ctx, t, db,
		`INSERT INTO autosearch_status (module_type, entity_type, entity_id, search_type) VALUES ('tv', 'series', ?, 'missing')`, seriesID)

	// -- Episode-level shared table records --
	mustExec(ctx, t, db,
		`INSERT INTO download_mappings (client_id, download_id, module_type, entity_type, entity_id, is_season_pack, is_complete_series) VALUES (1, 'dl-tv-2', 'tv', 'episode', ?, 0, 0)`, episodeID)
	mustExec(ctx, t, db,
		`INSERT INTO downloads (title, module_type, entity_type, entity_id) VALUES ('Episode 1', 'tv', 'episode', ?)`, episodeID)
	mustExec(ctx, t, db,
		`INSERT INTO history (event_type, module_type, entity_type, entity_id) VALUES ('grabbed', 'tv', 'episode', ?)`, episodeID)
	mustExec(ctx, t, db,
		`INSERT INTO autosearch_status (module_type, entity_type, entity_id, search_type) VALUES ('tv', 'episode', ?, 'missing')`, episodeID)
	mustExec(ctx, t, db,
		`INSERT INTO import_decisions (source_path, decision, module_type, entity_type, entity_id) VALUES ('/tmp/ep1.mkv', 'accepted', 'tv', 'episode', ?)`, episodeID)

	// -- Requests: series-level and season/episode-level all linked via media_id = seriesID --
	mustExec(ctx, t, db,
		`INSERT INTO requests (user_id, module_type, entity_type, title, media_id) VALUES (1, 'tv', 'series', 'Test Show', ?)`, seriesID)
	mustExec(ctx, t, db,
		`INSERT INTO requests (user_id, module_type, entity_type, title, media_id, season_number) VALUES (1, 'tv', 'season', 'Test Show S01', ?, 1)`, seriesID)
	mustExec(ctx, t, db,
		`INSERT INTO requests (user_id, module_type, entity_type, title, media_id, season_number, episode_number) VALUES (1, 'tv', 'episode', 'Test Show S01E01', ?, 1, 1)`, seriesID)

	// -- Other series records (should NOT be deleted) --
	mustExec(ctx, t, db,
		`INSERT INTO download_mappings (client_id, download_id, module_type, entity_type, entity_id, is_season_pack, is_complete_series) VALUES (1, 'dl-tv-3', 'tv', 'series', ?, 0, 0)`, otherSeriesID)
	mustExec(ctx, t, db,
		`INSERT INTO requests (user_id, module_type, entity_type, title, media_id) VALUES (1, 'tv', 'series', 'Other Show', ?)`, otherSeriesID)

	// -- Delete the series entity --
	if err := module.DeleteEntity(ctx, db, module.TypeTV, module.EntitySeries, seriesID); err != nil {
		t.Fatalf("DeleteEntity (series) failed: %v", err)
	}

	// Series-level records are deleted.
	assertCount(ctx, t, db, 0, `SELECT COUNT(*) FROM download_mappings WHERE module_type = 'tv' AND entity_type = 'series' AND entity_id = ?`, seriesID)
	assertCount(ctx, t, db, 0, `SELECT COUNT(*) FROM history WHERE module_type = 'tv' AND entity_type = 'series' AND entity_id = ?`, seriesID)
	assertCount(ctx, t, db, 0, `SELECT COUNT(*) FROM autosearch_status WHERE module_type = 'tv' AND entity_type = 'series' AND entity_id = ?`, seriesID)

	// Requests for series AND child entity types (season, episode) are deleted via the
	// "DELETE FROM requests WHERE module_type = ? AND media_id = ?" cascade.
	assertCount(ctx, t, db, 0, `SELECT COUNT(*) FROM requests WHERE module_type = 'tv' AND media_id = ?`, seriesID)

	// Episode-level records should still exist (DeleteEntity was called for series, not episode).
	assertCount(ctx, t, db, 1, `SELECT COUNT(*) FROM download_mappings WHERE module_type = 'tv' AND entity_type = 'episode' AND entity_id = ?`, episodeID)
	assertCount(ctx, t, db, 1, `SELECT COUNT(*) FROM downloads WHERE module_type = 'tv' AND entity_type = 'episode' AND entity_id = ?`, episodeID)
	assertCount(ctx, t, db, 1, `SELECT COUNT(*) FROM history WHERE module_type = 'tv' AND entity_type = 'episode' AND entity_id = ?`, episodeID)

	// Other series records are untouched.
	assertCount(ctx, t, db, 1, `SELECT COUNT(*) FROM download_mappings WHERE module_type = 'tv' AND entity_type = 'series' AND entity_id = ?`, otherSeriesID)
	assertCount(ctx, t, db, 1, `SELECT COUNT(*) FROM requests WHERE module_type = 'tv' AND media_id = ?`, otherSeriesID)
}

func TestDeleteEntity_Episode(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()
	ctx := context.Background()
	db := tdb.Conn

	mustExec(ctx, t, db,
		`INSERT INTO download_clients (name, type, host, port, use_ssl) VALUES ('test', 'qbittorrent', 'localhost', 8080, 0)`)
	mustExec(ctx, t, db,
		`INSERT INTO portal_users (username, password_hash, enabled) VALUES ('test', 'hash', 1)`)

	const episodeID int64 = 77
	const otherEpisodeID int64 = 88

	// Target episode records.
	mustExec(ctx, t, db,
		`INSERT INTO download_mappings (client_id, download_id, module_type, entity_type, entity_id, is_season_pack, is_complete_series) VALUES (1, 'dl-ep-1', 'tv', 'episode', ?, 0, 0)`, episodeID)
	mustExec(ctx, t, db,
		`INSERT INTO queue_media (download_mapping_id, module_type, entity_type, entity_id) VALUES (1, 'tv', 'episode', ?)`, episodeID)
	mustExec(ctx, t, db,
		`INSERT INTO downloads (title, module_type, entity_type, entity_id) VALUES ('Ep', 'tv', 'episode', ?)`, episodeID)
	mustExec(ctx, t, db,
		`INSERT INTO history (event_type, module_type, entity_type, entity_id) VALUES ('grabbed', 'tv', 'episode', ?)`, episodeID)
	mustExec(ctx, t, db,
		`INSERT INTO autosearch_status (module_type, entity_type, entity_id, search_type) VALUES ('tv', 'episode', ?, 'missing')`, episodeID)
	mustExec(ctx, t, db,
		`INSERT INTO import_decisions (source_path, decision, module_type, entity_type, entity_id) VALUES ('/tmp/ep.mkv', 'accepted', 'tv', 'episode', ?)`, episodeID)

	// Other episode records.
	mustExec(ctx, t, db,
		`INSERT INTO download_mappings (client_id, download_id, module_type, entity_type, entity_id, is_season_pack, is_complete_series) VALUES (1, 'dl-ep-2', 'tv', 'episode', ?, 0, 0)`, otherEpisodeID)
	mustExec(ctx, t, db,
		`INSERT INTO downloads (title, module_type, entity_type, entity_id) VALUES ('Other Ep', 'tv', 'episode', ?)`, otherEpisodeID)

	if err := module.DeleteEntity(ctx, db, module.TypeTV, module.EntityEpisode, episodeID); err != nil {
		t.Fatalf("DeleteEntity (episode) failed: %v", err)
	}

	// Target episode records are gone.
	assertCount(ctx, t, db, 0, `SELECT COUNT(*) FROM download_mappings WHERE module_type = 'tv' AND entity_type = 'episode' AND entity_id = ?`, episodeID)
	assertCount(ctx, t, db, 0, `SELECT COUNT(*) FROM queue_media WHERE module_type = 'tv' AND entity_type = 'episode' AND entity_id = ?`, episodeID)
	assertCount(ctx, t, db, 0, `SELECT COUNT(*) FROM downloads WHERE module_type = 'tv' AND entity_type = 'episode' AND entity_id = ?`, episodeID)
	assertCount(ctx, t, db, 0, `SELECT COUNT(*) FROM history WHERE module_type = 'tv' AND entity_type = 'episode' AND entity_id = ?`, episodeID)
	assertCount(ctx, t, db, 0, `SELECT COUNT(*) FROM autosearch_status WHERE module_type = 'tv' AND entity_type = 'episode' AND entity_id = ?`, episodeID)
	assertCount(ctx, t, db, 0, `SELECT COUNT(*) FROM import_decisions WHERE module_type = 'tv' AND entity_type = 'episode' AND entity_id = ?`, episodeID)

	// Other episode records are intact.
	assertCount(ctx, t, db, 1, `SELECT COUNT(*) FROM download_mappings WHERE module_type = 'tv' AND entity_type = 'episode' AND entity_id = ?`, otherEpisodeID)
	assertCount(ctx, t, db, 1, `SELECT COUNT(*) FROM downloads WHERE module_type = 'tv' AND entity_type = 'episode' AND entity_id = ?`, otherEpisodeID)
}

// mustExec executes a SQL statement and fails the test on error.
func mustExec(ctx context.Context, t *testing.T, db *sql.DB, query string, args ...any) {
	t.Helper()
	if _, err := db.ExecContext(ctx, query, args...); err != nil {
		t.Fatalf("mustExec(%q): %v", query, err)
	}
}

// assertCount queries for a single integer count and asserts it matches expected.
func assertCount(ctx context.Context, t *testing.T, db *sql.DB, expected int, query string, args ...any) {
	t.Helper()
	var count int
	if err := db.QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
		t.Fatalf("assertCount(%q): %v", query, err)
	}
	if count != expected {
		t.Errorf("assertCount(%q): got %d, want %d", query, count, expected)
	}
}
