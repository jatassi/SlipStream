-- +goose Up
-- Finished Imports whose Grab is gone: exactly what the retention step may delete.
CREATE INDEX imports_import_prunable ON imports_import (finished_at, id) WHERE finished_at IS NOT NULL AND grab_id IS NULL;
-- Every Import of a Grab, so deleting a Grab SET NULLs its Imports without a scan.
-- imports_import_grab, the partial unique index, stays.
CREATE INDEX imports_import_grab_any ON imports_import (grab_id);
