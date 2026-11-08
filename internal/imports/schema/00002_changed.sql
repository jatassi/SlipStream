-- +goose Up
-- changed counts the recorded files a root's last comparison found with
-- another size or modification time, for the root's scan line.
ALTER TABLE imports_root ADD COLUMN changed INTEGER NOT NULL DEFAULT 0 CHECK (changed >= 0);
