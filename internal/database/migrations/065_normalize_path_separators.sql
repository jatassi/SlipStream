-- +goose Up
UPDATE series SET path = REPLACE(path, '\', '/') WHERE path LIKE '%\%';
UPDATE movies SET path = REPLACE(path, '\', '/') WHERE path LIKE '%\%';
UPDATE movie_files SET path = REPLACE(path, '\', '/') WHERE path LIKE '%\%';
UPDATE movie_files SET original_path = REPLACE(original_path, '\', '/') WHERE original_path LIKE '%\%';
UPDATE episode_files SET path = REPLACE(path, '\', '/') WHERE path LIKE '%\%';
UPDATE episode_files SET original_path = REPLACE(original_path, '\', '/') WHERE original_path LIKE '%\%';
UPDATE root_folders SET path = REPLACE(path, '\', '/') WHERE path LIKE '%\%';

-- +goose Down
-- No-op: cannot reverse path normalization (original separators unknown)
