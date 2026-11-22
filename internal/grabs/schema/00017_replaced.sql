-- +goose Up
-- grabs_grab.replaced: every cell the Grab holds has a later Grab, so no
-- cell reads from it. Needs you walks the failed Grabs not replaced, so
-- its read is bounded by current failures, not by Grab history. Each new
-- cell weighs its own Grab again and marks the earlier Grabs of that cell.
-- Deletes never clear it: retention keeps a later Grab while an earlier
-- one would read in its place.
ALTER TABLE grabs_grab ADD COLUMN replaced INTEGER NOT NULL DEFAULT 0 CHECK (replaced IN (0, 1));

UPDATE grabs_grab SET replaced = 1 WHERE NOT EXISTS (
    SELECT 1 FROM grabs_target mine WHERE mine.grab_id = grabs_grab.id AND NOT EXISTS (
        SELECT 1 FROM grabs_target later JOIN grabs_grab n ON n.id = later.grab_id
        WHERE later.item_id = mine.item_id AND later.target_id = mine.target_id AND n.edition = grabs_grab.edition
            AND (n.created_at, n.id) > (grabs_grab.created_at, grabs_grab.id)));

-- +goose StatementBegin
CREATE TRIGGER grabs_target_replaces AFTER INSERT ON grabs_target BEGIN
    UPDATE grabs_grab SET replaced = NOT EXISTS (
        SELECT 1 FROM grabs_target mine WHERE mine.grab_id = grabs_grab.id AND NOT EXISTS (
            SELECT 1 FROM grabs_target later JOIN grabs_grab n ON n.id = later.grab_id
            WHERE later.item_id = mine.item_id AND later.target_id = mine.target_id AND n.edition = grabs_grab.edition
                AND (n.created_at, n.id) > (grabs_grab.created_at, grabs_grab.id)))
    WHERE id = new.grab_id;
    UPDATE grabs_grab SET replaced = 1
    WHERE replaced = 0
        AND id IN (SELECT grab_id FROM grabs_target WHERE item_id = new.item_id AND target_id = new.target_id AND grab_id <> new.grab_id)
        AND edition = (SELECT edition FROM grabs_grab WHERE id = new.grab_id)
        AND (created_at, id) < (SELECT created_at, id FROM grabs_grab WHERE id = new.grab_id)
        AND NOT EXISTS (
            SELECT 1 FROM grabs_target mine WHERE mine.grab_id = grabs_grab.id AND NOT EXISTS (
                SELECT 1 FROM grabs_target later JOIN grabs_grab n ON n.id = later.grab_id
                WHERE later.item_id = mine.item_id AND later.target_id = mine.target_id AND n.edition = grabs_grab.edition
                    AND (n.created_at, n.id) > (grabs_grab.created_at, grabs_grab.id)));
END;
-- +goose StatementEnd

CREATE INDEX grabs_grab_failing ON grabs_grab (created_at) WHERE stage = 'failed' AND replaced = 0;
