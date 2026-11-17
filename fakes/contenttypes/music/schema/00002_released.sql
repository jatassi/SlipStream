-- +goose Up
CREATE INDEX music_album_released ON music_album (released) WHERE length(released) = 10;
