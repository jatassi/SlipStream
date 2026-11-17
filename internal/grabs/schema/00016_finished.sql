-- +goose Up
-- Finished Grabs by when they finished: the retention walk.
CREATE INDEX grabs_grab_finished ON grabs_grab (updated_at, id) WHERE stage IN ('imported', 'failed');
