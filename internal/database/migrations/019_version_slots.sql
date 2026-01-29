-- +goose Up
-- +goose StatementBegin

-- Req 1.2.1: Global master toggle for multi-version functionality
-- Req 1.2.2: When disabled, system behaves as single-version (Slot 1 only)
CREATE TABLE IF NOT EXISTS multi_version_settings (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    enabled INTEGER NOT NULL DEFAULT 0,
    dry_run_completed INTEGER NOT NULL DEFAULT 0,
    last_migration_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Insert default row (disabled)
INSERT INTO multi_version_settings (id, enabled, dry_run_completed) VALUES (1, 0, 0);

-- Req 1.1.1: Maximum of 3 version slots supported
-- Req 1.1.2: Slots are globally defined and shared between Movies and TV Series
-- Req 1.1.3: Each slot has a user-defined custom name/label
-- Req 1.1.4: Each slot has an explicit enable/disable toggle independent of profile assignment
-- Req 1.1.5: Each slot is assigned exactly one quality profile
CREATE TABLE IF NOT EXISTS version_slots (
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

-- Insert 3 default slots (Req 1.1.1)
-- Slots 1 and 2 are always enabled (no toggle in UI), slot 3 is optional
INSERT INTO version_slots (slot_number, name, enabled, display_order) VALUES
    (1, 'Primary', 1, 1),
    (2, 'Secondary', 1, 2),
    (3, 'Tertiary', 0, 3);

-- Add slot_id to file tables for tracking which slot a file belongs to
ALTER TABLE movie_files ADD COLUMN slot_id INTEGER REFERENCES version_slots(id) ON DELETE SET NULL;
ALTER TABLE episode_files ADD COLUMN slot_id INTEGER REFERENCES version_slots(id) ON DELETE SET NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- SQLite doesn't support DROP COLUMN directly
-- The slot_id columns on movie_files and episode_files will remain but won't affect functionality

DROP TABLE IF EXISTS version_slots;
DROP TABLE IF EXISTS multi_version_settings;

-- +goose StatementEnd
