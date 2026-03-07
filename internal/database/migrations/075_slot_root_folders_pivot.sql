-- +goose Up
-- +goose StatementBegin

PRAGMA foreign_keys = OFF;

-- 1. Create pivot table
CREATE TABLE slot_root_folders (
    slot_id INTEGER NOT NULL REFERENCES version_slots(id) ON DELETE CASCADE,
    module_type TEXT NOT NULL,
    root_folder_id INTEGER NOT NULL REFERENCES root_folders(id) ON DELETE CASCADE,
    PRIMARY KEY (slot_id, module_type)
);

-- 2. Migrate existing data
INSERT INTO slot_root_folders (slot_id, module_type, root_folder_id)
SELECT id, 'movie', movie_root_folder_id
FROM version_slots
WHERE movie_root_folder_id IS NOT NULL;

INSERT INTO slot_root_folders (slot_id, module_type, root_folder_id)
SELECT id, 'tv', tv_root_folder_id
FROM version_slots
WHERE tv_root_folder_id IS NOT NULL;

-- 3. Recreate version_slots without per-module columns
CREATE TABLE version_slots_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    slot_number INTEGER NOT NULL CHECK (slot_number >= 1 AND slot_number <= 3),
    name TEXT NOT NULL,
    enabled INTEGER NOT NULL DEFAULT 0,
    quality_profile_id INTEGER REFERENCES quality_profiles(id) ON DELETE SET NULL,
    display_order INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(slot_number)
);

INSERT INTO version_slots_new (id, slot_number, name, enabled, quality_profile_id, display_order, created_at, updated_at)
SELECT id, slot_number, name, enabled, quality_profile_id, display_order, created_at, updated_at
FROM version_slots;

DROP TABLE version_slots;
ALTER TABLE version_slots_new RENAME TO version_slots;

PRAGMA foreign_keys = ON;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

PRAGMA foreign_keys = OFF;

ALTER TABLE version_slots ADD COLUMN movie_root_folder_id INTEGER REFERENCES root_folders(id) ON DELETE SET NULL;
ALTER TABLE version_slots ADD COLUMN tv_root_folder_id INTEGER REFERENCES root_folders(id) ON DELETE SET NULL;

UPDATE version_slots SET
    movie_root_folder_id = (SELECT root_folder_id FROM slot_root_folders WHERE slot_id = version_slots.id AND module_type = 'movie'),
    tv_root_folder_id = (SELECT root_folder_id FROM slot_root_folders WHERE slot_id = version_slots.id AND module_type = 'tv');

DROP TABLE IF EXISTS slot_root_folders;

PRAGMA foreign_keys = ON;

-- +goose StatementEnd
