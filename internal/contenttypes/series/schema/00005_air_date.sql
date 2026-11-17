-- +goose Up
CREATE INDEX series_episode_air ON series_episode (air_date) WHERE air_date <> '';
