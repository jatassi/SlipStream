-- name: NameTaken :one
SELECT EXISTS (SELECT 1 FROM indexers_indexer WHERE name = sqlc.arg(name) AND id IS NOT sqlc.arg(except));

-- name: InsertIndexer :exec
INSERT INTO indexers_indexer (
    id, name, kind, home_node_id, is_on, auto_search, rss, allow_failover, priority, categories, media,
    search_limit, search_window, grab_limit, grab_window, definition_id, definition_sha, base_address, plain,
    secrets_sealed, feed_host, tracker_name, allow_no_size, rss_reading, caps, notes, revision, created_at, updated_at)
VALUES (
    sqlc.arg(id), sqlc.arg(name), sqlc.arg(kind), sqlc.arg(home_node_id), sqlc.arg(is_on), sqlc.arg(auto_search), sqlc.arg(rss),
    sqlc.arg(allow_failover), sqlc.arg(priority), sqlc.narg(categories), sqlc.narg(media), sqlc.arg(search_limit),
    sqlc.arg(search_window), sqlc.arg(grab_limit), sqlc.arg(grab_window), sqlc.narg(definition_id), sqlc.narg(definition_sha),
    sqlc.narg(base_address), sqlc.arg(plain), sqlc.arg(secrets_sealed), sqlc.narg(feed_host), sqlc.narg(tracker_name),
    sqlc.arg(allow_no_size), sqlc.narg(rss_reading), sqlc.arg(caps), sqlc.arg(notes), sqlc.arg(revision), sqlc.arg(created_at),
    sqlc.arg(updated_at));

-- name: UpdateIndexer :execrows
UPDATE indexers_indexer SET
    name = sqlc.arg(name), home_node_id = sqlc.arg(home_node_id), is_on = sqlc.arg(is_on), auto_search = sqlc.arg(auto_search),
    rss = sqlc.arg(rss), allow_failover = sqlc.arg(allow_failover), priority = sqlc.arg(priority),
    categories = sqlc.narg(categories), media = sqlc.narg(media), search_limit = sqlc.arg(search_limit),
    search_window = sqlc.arg(search_window), grab_limit = sqlc.arg(grab_limit), grab_window = sqlc.arg(grab_window),
    base_address = sqlc.narg(base_address), plain = sqlc.arg(plain), secrets_sealed = sqlc.arg(secrets_sealed),
    feed_host = sqlc.narg(feed_host), tracker_name = sqlc.narg(tracker_name), allow_no_size = sqlc.arg(allow_no_size),
    caps = sqlc.arg(caps), revision = sqlc.arg(revision), updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id);

-- name: ListIndexers :many
SELECT * FROM indexers_indexer ORDER BY name;

-- name: GetIndexer :one
SELECT * FROM indexers_indexer WHERE id = sqlc.arg(id);

-- name: DeleteIndexer :execrows
DELETE FROM indexers_indexer WHERE id = sqlc.arg(id);

-- name: CountHomedOn :one
SELECT count(*) FROM indexers_indexer WHERE home_node_id = sqlc.arg(home_node_id);

-- name: Rehome :exec
UPDATE indexers_indexer SET home_node_id = sqlc.arg(heir) WHERE home_node_id = sqlc.arg(home_node_id);

-- name: GetStatus :one
SELECT * FROM indexers_status WHERE indexer_id = sqlc.arg(indexer_id);

-- name: ListStatuses :many
SELECT * FROM indexers_status ORDER BY indexer_id;

-- name: PutStatus :exec
INSERT INTO indexers_status (
    indexer_id, state, rung, until_at, since_at, cause, text, node_id, solver_tried, last_failure, last_failure_at, success_at, revision)
VALUES (
    sqlc.arg(indexer_id), sqlc.arg(state), sqlc.arg(rung), sqlc.narg(until_at), sqlc.narg(since_at), sqlc.narg(cause), sqlc.narg(text),
    sqlc.narg(node_id), sqlc.arg(solver_tried), sqlc.narg(last_failure), sqlc.narg(last_failure_at), sqlc.narg(success_at), sqlc.arg(revision))
ON CONFLICT (indexer_id) DO UPDATE SET
    state = excluded.state, rung = excluded.rung, until_at = excluded.until_at, since_at = excluded.since_at, cause = excluded.cause,
    text = excluded.text, node_id = excluded.node_id, solver_tried = excluded.solver_tried, last_failure = excluded.last_failure,
    last_failure_at = excluded.last_failure_at, success_at = excluded.success_at, revision = excluded.revision;

-- name: OfferSession :execrows
INSERT INTO indexers_session (indexer_id, node_id, cookies_sealed, user_agent, pasted, earned_at)
VALUES (sqlc.arg(indexer_id), sqlc.arg(node_id), sqlc.arg(cookies_sealed), sqlc.arg(user_agent), sqlc.arg(pasted), sqlc.arg(earned_at))
ON CONFLICT (indexer_id, node_id) DO UPDATE SET
    cookies_sealed = excluded.cookies_sealed, user_agent = excluded.user_agent, pasted = excluded.pasted, earned_at = excluded.earned_at
WHERE excluded.earned_at > indexers_session.earned_at;

-- name: GetSession :one
SELECT * FROM indexers_session WHERE indexer_id = sqlc.arg(indexer_id) AND node_id = sqlc.arg(node_id);

-- name: ListSessions :many
SELECT indexer_id, node_id, pasted, earned_at FROM indexers_session ORDER BY indexer_id, earned_at DESC;

-- name: DeleteSessions :exec
DELETE FROM indexers_session WHERE indexer_id = sqlc.arg(indexer_id);

-- name: InsertQuery :exec
INSERT INTO indexers_query (id, indexer_id, use, reserved_at, revision, sha)
VALUES (sqlc.arg(id), sqlc.arg(indexer_id), sqlc.arg(use), sqlc.arg(reserved_at), sqlc.arg(revision), sqlc.narg(sha));

-- name: CompleteQuery :execrows
UPDATE indexers_query SET
    started_at = sqlc.narg(started_at), completed_at = sqlc.arg(completed_at), node_id = sqlc.arg(node_id),
    outcome = sqlc.arg(outcome), releases = sqlc.narg(releases)
WHERE id = sqlc.arg(id) AND completed_at IS NULL;

-- name: ReservedSince :many
SELECT indexer_id, use, reserved_at FROM indexers_query
WHERE use IN ('search', 'recent', 'fetch') AND reserved_at > sqlc.arg(since)
ORDER BY reserved_at;

-- name: ReservedOfSince :many
SELECT reserved_at FROM indexers_query
WHERE indexer_id = sqlc.arg(indexer_id) AND reserved_at > sqlc.arg(since) AND use IN (sqlc.slice(uses))
ORDER BY reserved_at;

-- name: ListQueries :many
SELECT * FROM indexers_query WHERE indexer_id = sqlc.arg(indexer_id)
ORDER BY reserved_at DESC, id DESC
LIMIT sqlc.arg(lim) OFFSET sqlc.arg(off);

-- name: SetCaps :exec
UPDATE indexers_indexer SET caps = sqlc.arg(caps) WHERE id = sqlc.arg(id);

-- name: GetCatalogState :one
SELECT * FROM indexers_catalog_state WHERE one = 1;

-- name: SetCatalogRefreshed :exec
UPDATE indexers_catalog_state SET refreshed_at = sqlc.arg(refreshed_at), last_modified = sqlc.narg(last_modified) WHERE one = 1;

-- name: SetCatalogAddress :exec
UPDATE indexers_catalog_state SET address = sqlc.narg(address), last_modified = NULL WHERE one = 1;

-- name: ListCatalog :many
SELECT c.id, c.file, c.sha, c.name, c.description, c.privacy, c.language, c.categories, d.refusal, d.fetched_at
FROM indexers_catalog c JOIN indexers_definition d ON d.id = c.id AND d.sha = c.sha
ORDER BY c.name, c.id;

-- name: GetCatalogEntry :one
SELECT * FROM indexers_catalog WHERE id = sqlc.arg(id);

-- name: ClearCatalog :exec
DELETE FROM indexers_catalog;

-- name: InsertCatalogEntry :exec
INSERT INTO indexers_catalog (id, file, sha, name, description, privacy, language, categories)
VALUES (sqlc.arg(id), sqlc.arg(file), sqlc.arg(sha), sqlc.arg(name), sqlc.arg(description), sqlc.arg(privacy), sqlc.arg(language), sqlc.arg(categories));

-- name: ListDefinitionKeys :many
SELECT id, sha FROM indexers_definition ORDER BY id, sha;

-- name: InsertDefinition :exec
INSERT INTO indexers_definition (id, sha, file, origin, yaml, summary, refusal, fetched_at)
VALUES (sqlc.arg(id), sqlc.arg(sha), sqlc.arg(file), sqlc.arg(origin), sqlc.arg(yaml), sqlc.narg(summary), sqlc.narg(refusal), sqlc.arg(fetched_at))
ON CONFLICT (id, sha) DO NOTHING;

-- name: GetDefinition :one
SELECT * FROM indexers_definition WHERE id = sqlc.arg(id) AND sha = sqlc.arg(sha);

-- name: ListCustom :many
SELECT * FROM indexers_definition WHERE origin = 'custom' ORDER BY id, fetched_at DESC, sha;

-- name: DeleteDefinition :exec
DELETE FROM indexers_definition WHERE id = sqlc.arg(id) AND sha = sqlc.arg(sha);

-- name: PruneDefinitions :exec
DELETE FROM indexers_definition
WHERE origin = 'catalog'
    AND NOT EXISTS (SELECT 1 FROM indexers_catalog c WHERE c.id = indexers_definition.id AND c.sha = indexers_definition.sha)
    AND NOT EXISTS (SELECT 1 FROM indexers_indexer i WHERE i.definition_id = indexers_definition.id AND i.definition_sha = indexers_definition.sha);

-- name: ListPins :many
SELECT i.id, i.name, i.definition_id, i.definition_sha, i.plain, i.secrets_sealed, d.origin, d.fetched_at
FROM indexers_indexer i JOIN indexers_definition d ON d.id = i.definition_id AND d.sha = i.definition_sha
ORDER BY i.name;

-- name: CountPinned :one
SELECT count(*) FROM indexers_indexer WHERE definition_id = sqlc.arg(definition_id) AND definition_sha = sqlc.arg(definition_sha);

-- name: MovePin :execrows
UPDATE indexers_indexer SET
    definition_sha = sqlc.arg(sha), plain = sqlc.arg(plain), secrets_sealed = sqlc.arg(secrets_sealed), notes = sqlc.arg(notes),
    updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id) AND definition_sha = sqlc.arg(from_sha);

-- name: GetCustom :one
SELECT * FROM indexers_definition WHERE origin = 'custom' AND id = sqlc.arg(id)
ORDER BY fetched_at DESC, sha
LIMIT 1;
