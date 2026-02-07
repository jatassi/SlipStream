-- +goose Up
ALTER TABLE movie_files ADD COLUMN audio_channels TEXT;
ALTER TABLE movie_files ADD COLUMN dynamic_range TEXT;
ALTER TABLE episode_files ADD COLUMN audio_channels TEXT;
ALTER TABLE episode_files ADD COLUMN dynamic_range TEXT;

-- +goose Down
