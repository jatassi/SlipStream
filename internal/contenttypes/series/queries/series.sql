-- name: UpsertSeries :exec
INSERT INTO series_series (
    work_id, tvdb_id, tmdb_id, imdb_id, english_title, original_title, original_language, country, network,
    creators, credits, status, first_aired, last_aired, next_aired, average_runtime_seconds, certification,
    certification_country, genres, aliases, overview, episode_order, orders, trailer_site, trailer_key, awards,
    name_by_air_date
) VALUES (
    sqlc.arg(work_id), sqlc.arg(tvdb_id), sqlc.arg(tmdb_id), sqlc.arg(imdb_id), sqlc.arg(english_title),
    sqlc.arg(original_title), sqlc.arg(original_language), sqlc.arg(country), sqlc.arg(network),
    sqlc.arg(creators), sqlc.arg(credits), sqlc.arg(status), sqlc.arg(first_aired), sqlc.arg(last_aired),
    sqlc.arg(next_aired), sqlc.arg(average_runtime_seconds), sqlc.arg(certification),
    sqlc.arg(certification_country), sqlc.arg(genres), sqlc.arg(aliases), sqlc.arg(overview),
    sqlc.arg(episode_order), sqlc.arg(orders), sqlc.arg(trailer_site), sqlc.arg(trailer_key), sqlc.arg(awards),
    sqlc.arg(name_by_air_date)
)
ON CONFLICT (work_id) DO UPDATE SET tmdb_id = excluded.tmdb_id, imdb_id = excluded.imdb_id,
    english_title = excluded.english_title, original_title = excluded.original_title,
    original_language = excluded.original_language, country = excluded.country, network = excluded.network,
    creators = excluded.creators, credits = excluded.credits, status = excluded.status,
    first_aired = excluded.first_aired, last_aired = excluded.last_aired, next_aired = excluded.next_aired,
    average_runtime_seconds = excluded.average_runtime_seconds, certification = excluded.certification,
    certification_country = excluded.certification_country, genres = excluded.genres, aliases = excluded.aliases,
    overview = excluded.overview, episode_order = excluded.episode_order, orders = excluded.orders,
    trailer_site = excluded.trailer_site, trailer_key = excluded.trailer_key, awards = excluded.awards;

-- name: GetSeries :one
SELECT * FROM series_series WHERE work_id = sqlc.arg(work_id);

-- name: ListSeriesOf :many
SELECT * FROM series_series WHERE work_id IN (sqlc.slice(work_ids)) ORDER BY work_id;

-- name: WorksByTMDB :many
SELECT work_id, tvdb_id, tmdb_id FROM series_series WHERE tmdb_id IN (sqlc.slice(tmdb_ids)) ORDER BY tmdb_id;

-- name: UpsertSeason :exec
INSERT INTO series_season (level_id, work_id, tvdb_season_id, number, episode_order, name, overview)
VALUES (sqlc.arg(level_id), sqlc.arg(work_id), sqlc.arg(tvdb_season_id), sqlc.arg(number), sqlc.arg(episode_order),
    sqlc.arg(name), sqlc.arg(overview))
ON CONFLICT (level_id) DO UPDATE SET tvdb_season_id = excluded.tvdb_season_id, number = excluded.number,
    episode_order = excluded.episode_order, name = excluded.name, overview = excluded.overview;

-- name: ListSeasons :many
SELECT * FROM series_season WHERE work_id = sqlc.arg(work_id) ORDER BY number;

-- name: UpsertEpisode :exec
INSERT INTO series_episode (
    item_id, work_id, tvdb_episode_id, season_number, episode_number, absolute_number, aired_season, aired_number,
    air_date, runtime_seconds, overview, imdb_id
) VALUES (
    sqlc.arg(item_id), sqlc.arg(work_id), sqlc.arg(tvdb_episode_id), sqlc.arg(season_number),
    sqlc.arg(episode_number), sqlc.arg(absolute_number), sqlc.arg(aired_season), sqlc.arg(aired_number),
    sqlc.arg(air_date), sqlc.arg(runtime_seconds), sqlc.arg(overview), sqlc.arg(imdb_id)
)
ON CONFLICT (item_id) DO UPDATE SET tvdb_episode_id = excluded.tvdb_episode_id,
    season_number = excluded.season_number, episode_number = excluded.episode_number,
    absolute_number = excluded.absolute_number, aired_season = excluded.aired_season,
    aired_number = excluded.aired_number, air_date = excluded.air_date, runtime_seconds = excluded.runtime_seconds,
    overview = excluded.overview, imdb_id = excluded.imdb_id;

-- name: ListEpisodes :many
SELECT * FROM series_episode WHERE work_id = sqlc.arg(work_id) ORDER BY season_number, episode_number;

-- name: ListEpisodesOf :many
SELECT * FROM series_episode WHERE work_id IN (sqlc.slice(work_ids)) ORDER BY work_id, season_number, episode_number;

-- name: GetSettings :one
SELECT * FROM series_settings WHERE id = 1;

-- name: SetSettings :exec
UPDATE series_settings SET air_delay_seconds = sqlc.arg(air_delay_seconds), year_slack = sqlc.arg(year_slack),
    add_name_by_air_date = sqlc.arg(add_name_by_air_date) WHERE id = 1;

-- name: SetNameByAirDate :exec
UPDATE series_series SET name_by_air_date = sqlc.arg(name_by_air_date) WHERE work_id = sqlc.arg(work_id);

-- name: ListEveryListed :many
SELECT work_id, network, certification, status, first_aired, genres FROM series_series ORDER BY work_id;

-- name: ListListedOf :many
SELECT work_id, network, certification, status, first_aired, genres FROM series_series
WHERE work_id IN (sqlc.slice(work_ids)) ORDER BY work_id;

-- name: ListStatusesOf :many
SELECT work_id, status FROM series_series WHERE work_id IN (sqlc.slice(work_ids)) ORDER BY work_id;

-- name: GetEpisode :one
SELECT * FROM series_episode WHERE item_id = sqlc.arg(item_id);

-- name: ListEveryTiming :many
SELECT e.item_id, e.air_date, coalesce(e.runtime_seconds, s.average_runtime_seconds) AS runtime_seconds
FROM series_episode e JOIN series_series s ON s.work_id = e.work_id ORDER BY e.item_id;

-- name: ListTimingsOf :many
SELECT e.item_id, e.air_date, coalesce(e.runtime_seconds, s.average_runtime_seconds) AS runtime_seconds
FROM series_episode e JOIN series_series s ON s.work_id = e.work_id
WHERE e.work_id IN (sqlc.slice(work_ids)) ORDER BY e.item_id;

-- name: ListSeasonRatingsRead :many
SELECT aired_season, read_at FROM series_season_rating WHERE work_id = sqlc.arg(work_id) ORDER BY aired_season;

-- name: MarkSeasonRatingsRead :exec
INSERT INTO series_season_rating (work_id, aired_season, read_at)
VALUES (sqlc.arg(work_id), sqlc.arg(aired_season), sqlc.arg(read_at))
ON CONFLICT (work_id, aired_season) DO UPDATE SET read_at = excluded.read_at;

-- name: ListClaimsOf :many
SELECT work_id, tvdb_id, tmdb_id, imdb_id, first_aired FROM series_series WHERE work_id IN (sqlc.slice(work_ids)) ORDER BY work_id;

-- name: ListEveryClaim :many
SELECT work_id, tvdb_id, tmdb_id, imdb_id, first_aired FROM series_series ORDER BY work_id;

-- name: ListNamesOf :many
SELECT work_id, tvdb_id, tmdb_id, imdb_id, english_title, first_aired, episode_order, name_by_air_date FROM series_series WHERE work_id IN (sqlc.slice(work_ids)) ORDER BY work_id;

-- name: ListEveryName :many
SELECT work_id, tvdb_id, tmdb_id, imdb_id, english_title, first_aired, episode_order, name_by_air_date FROM series_series ORDER BY work_id;

-- name: ListEpisodeNamesOf :many
SELECT item_id, work_id, season_number, episode_number, absolute_number, air_date FROM series_episode WHERE work_id IN (sqlc.slice(work_ids)) ORDER BY work_id, season_number, episode_number;

-- name: ListEveryEpisodeName :many
SELECT item_id, work_id, season_number, episode_number, absolute_number, air_date FROM series_episode ORDER BY work_id, season_number, episode_number;
