package module

import (
	"context"
	"database/sql"
	"fmt"
)

// DBTX is the common interface satisfied by both *sql.DB and *sql.Tx.
// This allows DeleteEntity to work within an existing transaction or create its own.
type DBTX interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// DeleteEntity cascades entity deletion to all shared tables.
// Modules MUST call this when deleting entities. Modules must not delete shared table
// records directly — all shared-table cleanup flows through this function.
//
// The db parameter accepts *sql.DB or *sql.Tx. When called with *sql.DB, no wrapping
// transaction is created — the caller is responsible for transaction management if needed.
// When called with *sql.Tx, the deletes participate in the caller's transaction.
//
// IMPORTANT: This function replaces the ON DELETE CASCADE behavior that was previously
// provided by FKs on download_mappings (movie_id, series_id, episode_id) and
// queue_media (movie_id, episode_id). All code paths that delete movies, series,
// seasons, or episodes MUST call this function.
func DeleteEntity(ctx context.Context, db DBTX, moduleType Type, entityType EntityType, entityID int64) error {
	mt := string(moduleType)
	et := string(entityType)

	// Delete from all shared tables that use the discriminator pattern.
	// Note: queue_media rows also cascade-delete via FK when their parent
	// download_mapping is deleted, but we delete them explicitly here first
	// to catch any queue_media rows whose entity differs from the mapping's
	// entity (e.g., episode-level queue rows under a season-pack mapping).
	standardDeletes := []string{
		"DELETE FROM download_mappings WHERE module_type = ? AND entity_type = ? AND entity_id = ?",
		"DELETE FROM queue_media WHERE module_type = ? AND entity_type = ? AND entity_id = ?",
		"DELETE FROM downloads WHERE module_type = ? AND entity_type = ? AND entity_id = ?",
		"DELETE FROM history WHERE module_type = ? AND entity_type = ? AND entity_id = ?",
		"DELETE FROM autosearch_status WHERE module_type = ? AND entity_type = ? AND entity_id = ?",
		"DELETE FROM import_decisions WHERE module_type = ? AND entity_type = ? AND entity_id = ?",
	}

	for _, query := range standardDeletes {
		if _, err := db.ExecContext(ctx, query, mt, et, entityID); err != nil {
			return fmt.Errorf("delete shared data: %w", err)
		}
	}

	// Requests table uses (module_type, entity_type, media_id) — note media_id not entity_id.
	// media_id stores the library entity ID once a request is fulfilled.
	// For root entities (movies, series), also delete child-level requests.
	if _, err := db.ExecContext(ctx,
		"DELETE FROM requests WHERE module_type = ? AND entity_type = ? AND media_id = ?",
		mt, et, entityID,
	); err != nil {
		return fmt.Errorf("delete requests: %w", err)
	}
	// For root entities, cascade to child request types too (e.g., deleting a series
	// should also delete season/episode requests linked to that series via media_id).
	// This is safe because media_id for TV requests always holds the series.id,
	// so this catches all season/episode requests associated with the deleted entity.
	if _, err := db.ExecContext(ctx,
		"DELETE FROM requests WHERE module_type = ? AND media_id = ?",
		mt, entityID,
	); err != nil {
		return fmt.Errorf("delete child requests: %w", err)
	}

	return nil
}
