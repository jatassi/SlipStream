-- +goose Up
-- A file's quality, as its import judged it. ADD COLUMN NOT NULL needs a
-- default; the UPDATE below sets every row.
ALTER TABLE files_file ADD COLUMN quality TEXT NOT NULL DEFAULT 'sdtv' CHECK (quality IN (
    'sdtv', 'dvd', 'webrip-480p', 'hdtv-720p', 'webrip-720p', 'webdl-720p', 'bluray-720p',
    'hdtv-1080p', 'webrip-1080p', 'webdl-1080p', 'bluray-1080p', 'remux-1080p',
    'hdtv-2160p', 'webrip-2160p', 'webdl-2160p', 'bluray-2160p', 'remux-2160p', 'cam'));

-- Files that existed before take the quality release.File gives their
-- Grab's release title and their probe. The expression is
-- release.QualitySQL as go run ./internal/release/cmd/qualitysql printed it
-- when this migration was written, frozen with it; migration_test.go
-- checks the two agree.
UPDATE files_file SET quality =
(WITH n (t, w, h) AS (SELECT ' ' || replace(replace(replace(replace(replace(replace(replace(replace(replace(replace(replace(replace(replace(replace(replace(replace(replace(replace(replace(replace(replace(replace(replace(replace(replace(replace(replace(replace(replace(replace(replace(replace(replace(replace(replace(replace(lower((SELECT release_title FROM grabs_grab WHERE grabs_grab.id = files_file.grab_id)), '''', ''), '’', ''), '!', ' '), '"', ' '), '#', ' '), '$', ' '), '%', ' '), '&', ' '), '(', ' '), ')', ' '), '*', ' '), '+', ' '), ',', ' '), '-', ' '), '.', ' '), '/', ' '), ':', ' '), ';', ' '), '<', ' '), '=', ' '), '>', ' '), '?', ' '), '@', ' '), '[', ' '), '\', ' '), ']', ' '), '^', ' '), '_', ' '), '`', ' '), '{', ' '), '|', ' '), '}', ' '), '~', ' '), ' ', ' |'), '| ', ''), '|', '') || ' ', json_extract(files_file.probe, '$.width'), json_extract(files_file.probe, '$.height'))
 SELECT CASE
  WHEN w >= 3200 OR h >= 2000 THEN CASE
    WHEN instr(t, ' cam ') > 0 OR instr(t, ' hdcam ') > 0 OR instr(t, ' ts ') > 0 OR instr(t, ' hdts ') > 0 OR instr(t, ' telesync ') > 0 THEN 'cam'
    WHEN instr(t, ' remux ') > 0 OR instr(t, ' bdremux ') > 0 THEN 'remux-2160p'
    WHEN instr(t, ' bluray ') > 0 OR instr(t, ' blu ray ') > 0 OR instr(t, ' bdrip ') > 0 OR instr(t, ' brrip ') > 0 THEN 'bluray-2160p'
    WHEN instr(t, ' webrip ') > 0 OR instr(t, ' web rip ') > 0 THEN 'webrip-2160p'
    WHEN instr(t, ' webdl ') > 0 OR instr(t, ' web dl ') > 0 OR instr(t, ' web ') > 0 THEN 'webdl-2160p'
    WHEN instr(t, ' hdtv ') > 0 THEN 'hdtv-2160p'
    WHEN instr(t, ' sdtv ') > 0 OR instr(t, ' tvrip ') > 0 THEN 'hdtv-2160p'
    WHEN instr(t, ' dvd ') > 0 OR instr(t, ' dvdrip ') > 0 THEN 'hdtv-2160p'
    ELSE 'hdtv-2160p' END
  WHEN w >= 1800 OR h >= 1000 THEN CASE
    WHEN instr(t, ' cam ') > 0 OR instr(t, ' hdcam ') > 0 OR instr(t, ' ts ') > 0 OR instr(t, ' hdts ') > 0 OR instr(t, ' telesync ') > 0 THEN 'cam'
    WHEN instr(t, ' remux ') > 0 OR instr(t, ' bdremux ') > 0 THEN 'remux-1080p'
    WHEN instr(t, ' bluray ') > 0 OR instr(t, ' blu ray ') > 0 OR instr(t, ' bdrip ') > 0 OR instr(t, ' brrip ') > 0 THEN 'bluray-1080p'
    WHEN instr(t, ' webrip ') > 0 OR instr(t, ' web rip ') > 0 THEN 'webrip-1080p'
    WHEN instr(t, ' webdl ') > 0 OR instr(t, ' web dl ') > 0 OR instr(t, ' web ') > 0 THEN 'webdl-1080p'
    WHEN instr(t, ' hdtv ') > 0 THEN 'hdtv-1080p'
    WHEN instr(t, ' sdtv ') > 0 OR instr(t, ' tvrip ') > 0 THEN 'hdtv-1080p'
    WHEN instr(t, ' dvd ') > 0 OR instr(t, ' dvdrip ') > 0 THEN 'hdtv-1080p'
    ELSE 'hdtv-1080p' END
  WHEN w >= 1200 OR h >= 700 THEN CASE
    WHEN instr(t, ' cam ') > 0 OR instr(t, ' hdcam ') > 0 OR instr(t, ' ts ') > 0 OR instr(t, ' hdts ') > 0 OR instr(t, ' telesync ') > 0 THEN 'cam'
    WHEN instr(t, ' remux ') > 0 OR instr(t, ' bdremux ') > 0 THEN 'hdtv-720p'
    WHEN instr(t, ' bluray ') > 0 OR instr(t, ' blu ray ') > 0 OR instr(t, ' bdrip ') > 0 OR instr(t, ' brrip ') > 0 THEN 'bluray-720p'
    WHEN instr(t, ' webrip ') > 0 OR instr(t, ' web rip ') > 0 THEN 'webrip-720p'
    WHEN instr(t, ' webdl ') > 0 OR instr(t, ' web dl ') > 0 OR instr(t, ' web ') > 0 THEN 'webdl-720p'
    WHEN instr(t, ' hdtv ') > 0 THEN 'hdtv-720p'
    WHEN instr(t, ' sdtv ') > 0 OR instr(t, ' tvrip ') > 0 THEN 'hdtv-720p'
    WHEN instr(t, ' dvd ') > 0 OR instr(t, ' dvdrip ') > 0 THEN 'hdtv-720p'
    ELSE 'hdtv-720p' END
  WHEN w >= 1000 OR h >= 560 THEN CASE
    WHEN instr(t, ' cam ') > 0 OR instr(t, ' hdcam ') > 0 OR instr(t, ' ts ') > 0 OR instr(t, ' hdts ') > 0 OR instr(t, ' telesync ') > 0 THEN 'cam'
    WHEN instr(t, ' remux ') > 0 OR instr(t, ' bdremux ') > 0 THEN 'sdtv'
    WHEN instr(t, ' bluray ') > 0 OR instr(t, ' blu ray ') > 0 OR instr(t, ' bdrip ') > 0 OR instr(t, ' brrip ') > 0 THEN 'sdtv'
    WHEN instr(t, ' webrip ') > 0 OR instr(t, ' web rip ') > 0 THEN 'webrip-480p'
    WHEN instr(t, ' webdl ') > 0 OR instr(t, ' web dl ') > 0 OR instr(t, ' web ') > 0 THEN 'sdtv'
    WHEN instr(t, ' hdtv ') > 0 THEN 'sdtv'
    WHEN instr(t, ' sdtv ') > 0 OR instr(t, ' tvrip ') > 0 THEN 'sdtv'
    WHEN instr(t, ' dvd ') > 0 OR instr(t, ' dvdrip ') > 0 THEN 'dvd'
    ELSE 'sdtv' END
  ELSE CASE
    WHEN instr(t, ' cam ') > 0 OR instr(t, ' hdcam ') > 0 OR instr(t, ' ts ') > 0 OR instr(t, ' hdts ') > 0 OR instr(t, ' telesync ') > 0 THEN 'cam'
    WHEN instr(t, ' remux ') > 0 OR instr(t, ' bdremux ') > 0 THEN 'sdtv'
    WHEN instr(t, ' bluray ') > 0 OR instr(t, ' blu ray ') > 0 OR instr(t, ' bdrip ') > 0 OR instr(t, ' brrip ') > 0 THEN 'sdtv'
    WHEN instr(t, ' webrip ') > 0 OR instr(t, ' web rip ') > 0 THEN 'webrip-480p'
    WHEN instr(t, ' webdl ') > 0 OR instr(t, ' web dl ') > 0 OR instr(t, ' web ') > 0 THEN 'sdtv'
    WHEN instr(t, ' hdtv ') > 0 THEN 'sdtv'
    WHEN instr(t, ' sdtv ') > 0 OR instr(t, ' tvrip ') > 0 THEN 'sdtv'
    WHEN instr(t, ' dvd ') > 0 OR instr(t, ' dvdrip ') > 0 THEN 'dvd'
    ELSE 'sdtv' END
 END FROM n)
;

-- label is the Target name a file's {Target} renders: the first Target in
-- Profile order it fills, or for a Kept file the first it meets. extra is
-- N in " (extra N)", set exactly when the file fills nothing.
ALTER TABLE files_file ADD COLUMN label TEXT NOT NULL DEFAULT '';
ALTER TABLE files_file ADD COLUMN extra INTEGER NOT NULL DEFAULT 0 CHECK (extra >= 0);

UPDATE files_file SET label = coalesce((
    SELECT t.name FROM files_fill f
    JOIN targets_target t ON t.id = f.target_id
    JOIN targets_wants w ON w.work_id = files_file.work_id
    JOIN targets_profile_target pt ON pt.profile_id = w.profile_id AND pt.target_id = f.target_id
    WHERE f.file_id = files_file.id
    ORDER BY pt.position LIMIT 1), '');

-- A file whose bytes are not at its path yet: sits is where they are,
-- root-relative. The disk queue moves them through
-- .slipstream-incoming/<file id>.<ext>, a name only that file uses.
CREATE TABLE files_move (
    file_id BLOB PRIMARY KEY NOT NULL REFERENCES files_file (id) ON DELETE CASCADE,
    sits TEXT NOT NULL CHECK (sits <> '')
) STRICT;

-- A file gone from the library whose bytes the disk queue still removes:
-- at its path, or, when sits is set, wherever its move had taken them.
CREATE TABLE files_doomed (
    file_id BLOB PRIMARY KEY NOT NULL,
    root_id BLOB NOT NULL REFERENCES roots_root (id),
    path TEXT NOT NULL CHECK (path <> ''),
    sits TEXT CHECK (sits <> '')
) STRICT;

-- A file's own cut, which names it; the Edition a fill is for is the cut
-- a list wants, or any under any cut. Every Profile starts at any cut, so
-- every Standard fill is an any fill.
ALTER TABLE files_file RENAME COLUMN edition TO cut;
CREATE INDEX files_file_cut ON files_file (cut);
UPDATE files_fill SET edition = 'any' WHERE edition = 'standard';
