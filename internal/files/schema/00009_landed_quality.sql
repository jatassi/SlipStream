-- +goose Up
-- S39 (Home): the quality of the file each landing placed, which Recent's
-- "Upgraded to" names after that file is replaced or deleted. NULL on a
-- record written before this column whose file was already gone.
ALTER TABLE files_imported ADD COLUMN quality TEXT CHECK (quality IN (
    'sdtv', 'dvd', 'webrip-480p', 'hdtv-720p', 'webrip-720p', 'webdl-720p', 'bluray-720p',
    'hdtv-1080p', 'webrip-1080p', 'webdl-1080p', 'bluray-1080p', 'remux-1080p',
    'hdtv-2160p', 'webrip-2160p', 'webdl-2160p', 'bluray-2160p', 'remux-2160p', 'cam'));
UPDATE files_imported SET quality = (SELECT f.quality FROM files_file f WHERE f.id = files_imported.file_id)
WHERE file_id IS NOT NULL;
-- An Import's records, so Recent reads one Import's landings without a scan.
CREATE INDEX files_imported_import ON files_imported (import_id);
