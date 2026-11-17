-- +goose Up
CREATE INDEX movie_release_day ON movie_release (kind, date);
