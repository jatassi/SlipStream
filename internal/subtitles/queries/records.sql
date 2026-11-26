-- name: ListFilesIn :many
SELECT * FROM subtitles_file WHERE file_id IN (sqlc.slice(ids)) ORDER BY file_id;

-- name: ListWantsIn :many
SELECT * FROM subtitles_want WHERE file_id IN (sqlc.slice(ids)) ORDER BY file_id, language, forced;

-- name: ListSidecarsIn :many
SELECT * FROM subtitles_sidecar WHERE file_id IN (sqlc.slice(ids)) ORDER BY file_id, language, forced;

-- name: ListIntentsIn :many
SELECT * FROM subtitles_write WHERE file_id IN (sqlc.slice(ids)) ORDER BY file_id, language, forced;

-- name: ListIgnoredIn :many
SELECT * FROM subtitles_track_ignored WHERE file_id IN (sqlc.slice(ids)) ORDER BY file_id, stream_index;

-- name: DeleteIgnored :exec
DELETE FROM subtitles_track_ignored WHERE file_id = sqlc.arg(file_id) AND stream_index = sqlc.arg(stream_index);

-- name: PutFacts :exec
-- A new row is due a survey when its type has subtitles; an existing one
-- keeps its search and map state.
INSERT INTO subtitles_file (file_id, work_id, root_id, kind, wantless, audio, embedded, map_since, survey_due)
VALUES (sqlc.arg(file_id), sqlc.arg(work_id), sqlc.arg(root_id), sqlc.narg(kind), sqlc.arg(wantless), sqlc.arg(audio), sqlc.arg(embedded),
    sqlc.arg(map_since), sqlc.narg(kind) IS NOT NULL)
ON CONFLICT (file_id) DO UPDATE SET work_id = excluded.work_id, root_id = excluded.root_id, kind = excluded.kind,
    wantless = excluded.wantless, audio = excluded.audio, embedded = excluded.embedded;

-- name: PutWant :exec
INSERT INTO subtitles_want (file_id, language, forced, work_id, kind, unmet, due_at, gap_seconds)
VALUES (sqlc.arg(file_id), sqlc.arg(language), sqlc.arg(forced), sqlc.arg(work_id), sqlc.arg(kind),
    NOT EXISTS (SELECT 1 FROM subtitles_sidecar s WHERE s.file_id = sqlc.arg(file_id) AND s.language = sqlc.arg(language) AND s.forced = sqlc.arg(forced)),
    sqlc.narg(due_at), sqlc.arg(gap_seconds))
ON CONFLICT (file_id, language, forced) DO UPDATE SET work_id = excluded.work_id, kind = excluded.kind;

-- name: DeleteWant :exec
DELETE FROM subtitles_want WHERE file_id = sqlc.arg(file_id) AND language = sqlc.arg(language) AND forced = sqlc.arg(forced);

-- name: SetUnmet :exec
-- unmet is whether no Sidecar of the key exists. A final Sidecar ends
-- searching; a want that lost its Sidecar is due now unless it was due.
UPDATE subtitles_want SET
    unmet = NOT EXISTS (SELECT 1 FROM subtitles_sidecar s WHERE s.file_id = subtitles_want.file_id AND s.language = subtitles_want.language AND s.forced = subtitles_want.forced),
    due_at = CASE
        WHEN EXISTS (SELECT 1 FROM subtitles_sidecar s WHERE s.file_id = subtitles_want.file_id AND s.language = subtitles_want.language AND s.forced = subtitles_want.forced AND s.final = 1) THEN NULL
        ELSE coalesce(due_at, sqlc.arg(now)) END
WHERE subtitles_want.file_id = sqlc.arg(file_id) AND subtitles_want.language = sqlc.arg(language) AND subtitles_want.forced = sqlc.arg(forced);

-- name: PutSidecar :exec
INSERT INTO subtitles_sidecar (id, file_id, language, forced, hi, tail, source, provider, candidate_key, release, ai_translated, human_made,
    sibling_of, final, timed, offset_ms, ratio, segments, match, sha256, size, cues, timing_due, stray, added_at)
VALUES (sqlc.arg(id), sqlc.arg(file_id), sqlc.arg(language), sqlc.arg(forced), sqlc.arg(hi), sqlc.arg(tail), sqlc.arg(source), sqlc.narg(provider),
    sqlc.narg(candidate_key), sqlc.narg(release), sqlc.arg(ai_translated), sqlc.arg(human_made), sqlc.narg(sibling_of), sqlc.arg(final), sqlc.arg(timed),
    sqlc.narg(offset_ms), sqlc.narg(ratio), sqlc.narg(segments), sqlc.narg(match), sqlc.arg(sha256), sqlc.arg(size), sqlc.arg(cues), sqlc.arg(timing_due),
    sqlc.narg(stray), sqlc.arg(added_at))
ON CONFLICT (id) DO UPDATE SET hi = excluded.hi, tail = excluded.tail, source = excluded.source, provider = excluded.provider,
    candidate_key = excluded.candidate_key, release = excluded.release, ai_translated = excluded.ai_translated, human_made = excluded.human_made,
    sibling_of = excluded.sibling_of, final = excluded.final, timed = excluded.timed, offset_ms = excluded.offset_ms, ratio = excluded.ratio,
    segments = excluded.segments, match = excluded.match, sha256 = excluded.sha256, size = excluded.size, cues = excluded.cues,
    timing_due = excluded.timing_due, stray = excluded.stray;

-- name: DeleteSidecar :execrows
-- The Sidecar goes only while it still holds the bytes the caller read.
DELETE FROM subtitles_sidecar WHERE id = sqlc.arg(id) AND sha256 = sqlc.arg(sha256);

-- name: ListSidecarsOfFiles :many
-- Every Sidecar of the files file_ids names, a JSON array.
SELECT * FROM subtitles_sidecar
WHERE file_id IN (SELECT unhex(replace(value, '-', '')) FROM json_each(CAST(sqlc.arg(file_ids) AS TEXT)))
ORDER BY file_id, language, forced;
