-- name: UpsertMovie :exec
INSERT INTO movie_movie (
    work_id, tmdb_id, imdb_id, title, title_english, original_title, original_language, year, primary_release,
    overview, tagline, runtime_minutes, status, certification, certification_country, studio_id, studio_name,
    collection_id, collection_name, trailer_site, trailer_key, awards
) VALUES (
    sqlc.arg(work_id), sqlc.arg(tmdb_id), sqlc.arg(imdb_id), sqlc.arg(title), sqlc.arg(title_english),
    sqlc.arg(original_title), sqlc.arg(original_language), sqlc.arg(year), sqlc.arg(primary_release),
    sqlc.arg(overview), sqlc.arg(tagline), sqlc.arg(runtime_minutes), sqlc.arg(status), sqlc.arg(certification),
    sqlc.arg(certification_country), sqlc.arg(studio_id), sqlc.arg(studio_name), sqlc.arg(collection_id),
    sqlc.arg(collection_name), sqlc.arg(trailer_site), sqlc.arg(trailer_key), sqlc.arg(awards)
)
ON CONFLICT (work_id) DO UPDATE SET imdb_id = excluded.imdb_id, title = excluded.title,
    title_english = excluded.title_english, original_title = excluded.original_title,
    original_language = excluded.original_language, year = excluded.year, primary_release = excluded.primary_release,
    overview = excluded.overview, tagline = excluded.tagline, runtime_minutes = excluded.runtime_minutes,
    status = excluded.status, certification = excluded.certification,
    certification_country = excluded.certification_country, studio_id = excluded.studio_id,
    studio_name = excluded.studio_name, collection_id = excluded.collection_id,
    collection_name = excluded.collection_name, trailer_site = excluded.trailer_site,
    trailer_key = excluded.trailer_key, awards = excluded.awards;

-- name: GetMovie :one
SELECT * FROM movie_movie WHERE work_id = sqlc.arg(work_id);

-- name: ListMovies :many
SELECT * FROM movie_movie WHERE work_id IN (sqlc.slice(work_ids)) ORDER BY work_id;

-- name: ListGenres :many
SELECT name FROM movie_genre WHERE work_id = sqlc.arg(work_id) ORDER BY position;

-- name: ListTitles :many
SELECT country, title, kind FROM movie_title WHERE work_id = sqlc.arg(work_id) ORDER BY country, title, kind;

-- name: ListReleases :many
SELECT * FROM movie_release WHERE work_id IN (sqlc.slice(work_ids))
ORDER BY work_id, country, kind, date, certification, note;

-- name: ListCredits :many
SELECT person_id, name, department, role, photo_path FROM movie_credit WHERE work_id = sqlc.arg(work_id) ORDER BY position;

-- name: DeleteGenres :exec
DELETE FROM movie_genre WHERE work_id = sqlc.arg(work_id);

-- name: DeleteTitles :exec
DELETE FROM movie_title WHERE work_id = sqlc.arg(work_id);

-- name: DeleteReleases :exec
DELETE FROM movie_release WHERE work_id = sqlc.arg(work_id);

-- name: DeleteCredits :exec
DELETE FROM movie_credit WHERE work_id = sqlc.arg(work_id);

-- name: InsertGenre :exec
INSERT INTO movie_genre (work_id, position, name) VALUES (sqlc.arg(work_id), sqlc.arg(position), sqlc.arg(name));

-- name: InsertTitle :exec
INSERT INTO movie_title (work_id, country, title, kind)
VALUES (sqlc.arg(work_id), sqlc.arg(country), sqlc.arg(title), sqlc.arg(kind));

-- name: InsertRelease :exec
INSERT INTO movie_release (work_id, country, kind, date, certification, note)
VALUES (sqlc.arg(work_id), sqlc.arg(country), sqlc.arg(kind), sqlc.arg(date), sqlc.arg(certification), sqlc.arg(note));

-- name: InsertCredit :exec
INSERT INTO movie_credit (work_id, position, person_id, name, department, role, photo_path)
VALUES (sqlc.arg(work_id), sqlc.arg(position), sqlc.arg(person_id), sqlc.arg(name), sqlc.arg(department), sqlc.arg(role), sqlc.arg(photo_path));
