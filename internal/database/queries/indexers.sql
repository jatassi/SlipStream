-- name: GetIndexer :one
SELECT * FROM indexers WHERE id = ? LIMIT 1;

-- name: GetIndexerByDefinitionID :one
SELECT * FROM indexers WHERE definition_id = ? LIMIT 1;

-- name: ListIndexers :many
SELECT * FROM indexers ORDER BY priority, name;

-- name: ListEnabledIndexers :many
SELECT * FROM indexers WHERE enabled = 1 ORDER BY priority, name;

-- name: ListEnabledMovieIndexers :many
SELECT * FROM indexers WHERE enabled = 1 AND supports_movies = 1 ORDER BY priority, name;

-- name: ListEnabledTVIndexers :many
SELECT * FROM indexers WHERE enabled = 1 AND supports_tv = 1 ORDER BY priority, name;

-- name: ListIndexersByDefinition :many
SELECT * FROM indexers WHERE definition_id = ? ORDER BY priority, name;

-- name: CreateIndexer :one
INSERT INTO indexers (
    name, definition_id, settings, categories, supports_movies, supports_tv, priority, enabled, auto_search_enabled
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: UpdateIndexer :one
UPDATE indexers SET
    name = ?,
    definition_id = ?,
    settings = ?,
    categories = ?,
    supports_movies = ?,
    supports_tv = ?,
    priority = ?,
    enabled = ?,
    auto_search_enabled = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING *;

-- name: DeleteIndexer :exec
DELETE FROM indexers WHERE id = ?;

-- name: CountIndexers :one
SELECT COUNT(*) FROM indexers;

-- name: CountEnabledIndexers :one
SELECT COUNT(*) FROM indexers WHERE enabled = 1;

-- Indexer Status queries

-- name: GetIndexerStatus :one
SELECT * FROM indexer_status WHERE indexer_id = ? LIMIT 1;

-- name: UpsertIndexerStatus :one
INSERT INTO indexer_status (indexer_id, initial_failure, most_recent_failure, escalation_level, disabled_till, last_rss_sync)
VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT(indexer_id) DO UPDATE SET
    initial_failure = excluded.initial_failure,
    most_recent_failure = excluded.most_recent_failure,
    escalation_level = excluded.escalation_level,
    disabled_till = excluded.disabled_till,
    last_rss_sync = excluded.last_rss_sync
RETURNING *;

-- name: UpdateIndexerLastRssSync :exec
INSERT INTO indexer_status (indexer_id, last_rss_sync)
VALUES (?, CURRENT_TIMESTAMP)
ON CONFLICT(indexer_id) DO UPDATE SET
    last_rss_sync = CURRENT_TIMESTAMP;

-- name: ClearIndexerFailure :exec
UPDATE indexer_status SET
    initial_failure = NULL,
    most_recent_failure = NULL,
    escalation_level = 0,
    disabled_till = NULL
WHERE indexer_id = ?;

-- name: ListDisabledIndexers :many
SELECT i.*, s.disabled_till FROM indexers i
JOIN indexer_status s ON i.id = s.indexer_id
WHERE s.disabled_till IS NOT NULL AND s.disabled_till > CURRENT_TIMESTAMP;

-- Indexer History queries

-- name: CreateIndexerHistoryEvent :one
INSERT INTO indexer_history (
    indexer_id, event_type, successful, query, categories, results_count, elapsed_ms, data
) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: ListIndexerHistory :many
SELECT * FROM indexer_history
WHERE indexer_id = ?
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- name: ListRecentIndexerHistory :many
SELECT * FROM indexer_history
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- name: CountIndexerQueries :one
SELECT COUNT(*) FROM indexer_history
WHERE indexer_id = ? AND event_type = 'query' AND created_at > ?;

-- name: CountIndexerGrabs :one
SELECT COUNT(*) FROM indexer_history
WHERE indexer_id = ? AND event_type = 'grab' AND created_at > ?;

-- name: GetIndexerStats :one
SELECT
    COUNT(*) as total_events,
    SUM(CASE WHEN successful = 1 THEN 1 ELSE 0 END) as successful_events,
    SUM(CASE WHEN event_type = 'query' THEN 1 ELSE 0 END) as total_queries,
    SUM(CASE WHEN event_type = 'grab' THEN 1 ELSE 0 END) as total_grabs,
    AVG(CASE WHEN event_type = 'query' THEN elapsed_ms ELSE NULL END) as avg_query_time_ms
FROM indexer_history
WHERE indexer_id = ?;

-- name: DeleteOldIndexerHistory :exec
DELETE FROM indexer_history WHERE created_at < ?;

-- Definition Metadata queries

-- name: GetDefinitionMetadata :one
SELECT * FROM definition_metadata WHERE id = ? LIMIT 1;

-- name: ListDefinitionMetadata :many
SELECT * FROM definition_metadata ORDER BY name;

-- name: ListDefinitionMetadataByProtocol :many
SELECT * FROM definition_metadata WHERE protocol = ? ORDER BY name;

-- name: ListDefinitionMetadataByPrivacy :many
SELECT * FROM definition_metadata WHERE privacy = ? ORDER BY name;

-- name: UpsertDefinitionMetadata :one
INSERT INTO definition_metadata (id, name, description, privacy, language, protocol, cached_at, file_path)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
    name = excluded.name,
    description = excluded.description,
    privacy = excluded.privacy,
    language = excluded.language,
    protocol = excluded.protocol,
    cached_at = excluded.cached_at,
    file_path = excluded.file_path
RETURNING *;

-- name: DeleteDefinitionMetadata :exec
DELETE FROM definition_metadata WHERE id = ?;

-- name: ClearDefinitionMetadata :exec
DELETE FROM definition_metadata;

-- name: CountDefinitionMetadata :one
SELECT COUNT(*) FROM definition_metadata;

-- Auto-search enabled indexer queries

-- name: ListAutoSearchEnabledMovieIndexers :many
SELECT * FROM indexers WHERE enabled = 1 AND auto_search_enabled = 1 AND supports_movies = 1 ORDER BY priority, name;

-- name: ListAutoSearchEnabledTVIndexers :many
SELECT * FROM indexers WHERE enabled = 1 AND auto_search_enabled = 1 AND supports_tv = 1 ORDER BY priority, name;

-- name: ListAutoSearchEnabledIndexers :many
SELECT * FROM indexers WHERE enabled = 1 AND auto_search_enabled = 1 ORDER BY priority, name;

-- name: UpdateIndexerAutoSearchEnabled :exec
UPDATE indexers SET auto_search_enabled = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?;

-- Cookie management queries

-- name: GetIndexerCookies :one
SELECT cookies, cookies_expiration FROM indexer_status WHERE indexer_id = ? LIMIT 1;

-- name: UpdateIndexerCookies :exec
INSERT INTO indexer_status (indexer_id, cookies, cookies_expiration)
VALUES (?, ?, ?)
ON CONFLICT(indexer_id) DO UPDATE SET
    cookies = excluded.cookies,
    cookies_expiration = excluded.cookies_expiration;

-- name: ClearIndexerCookies :exec
UPDATE indexer_status SET cookies = NULL, cookies_expiration = NULL WHERE indexer_id = ?;
