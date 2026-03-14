# Production Database Path Fix

SlipStream's orphan import path had bugs that caused files to be imported with incorrect naming (missing year in series folder and episode filenames). The bugs have been fixed in code but the production data needs manual correction.

**Stop the SlipStream service before performing any of these operations.**

The database is at: `C:\Users\jacks\AppData\Local\SlipStream\slipstream.db`

## Phase 1: File/Folder Operations (PowerShell)

### 1. The Pitt (2025)

Two folders exist on disk: `D:\Plex\Shows\The Pitt\` (wrong) and `D:\Plex\Shows\The Pitt (2025)\` (correct). Merge into the correct one.

```powershell
# Create Season 1 in correct folder
New-Item -ItemType Directory -Path "D:\Plex\Shows\The Pitt (2025)\Season 1" -Force

# Move and rename S01E01-E15 from old folder
1..15 | ForEach-Object {
    $ep = $_.ToString("D2")
    Move-Item "D:\Plex\Shows\The Pitt\Season 1\The Pitt - S01E$ep - WEBDL-1080p.mkv" `
              "D:\Plex\Shows\The Pitt (2025)\Season 1\The Pitt (2025) - S01E$ep - WEBDL-1080p.mkv"
}

# Move and rename S02E01-E03 from old folder
1..3 | ForEach-Object {
    $ep = $_.ToString("D2")
    Move-Item "D:\Plex\Shows\The Pitt\Season 2\The Pitt - S02E$ep - WEBDL-1080p.mkv" `
              "D:\Plex\Shows\The Pitt (2025)\Season 2\The Pitt (2025) - S02E$ep - WEBDL-1080p.mkv"
}

# Move and rename S02E10 from old folder
Move-Item "D:\Plex\Shows\The Pitt\Season 2\The Pitt - S02E10 - WEB-DL-2160p.mkv" `
          "D:\Plex\Shows\The Pitt (2025)\Season 2\The Pitt (2025) - S02E10 - WEB-DL-2160p.mkv"

# Rename S02E04-E06 already in correct folder (filename missing year)
4..6 | ForEach-Object {
    $ep = $_.ToString("D2")
    Rename-Item "D:\Plex\Shows\The Pitt (2025)\Season 2\The Pitt - S02E$ep - WEB-DL-2160p.mkv" `
                "The Pitt (2025) - S02E$ep - WEB-DL-2160p.mkv"
}

# Delete old empty folder
Remove-Item "D:\Plex\Shows\The Pitt" -Recurse -Force
```

### 2. True Detective (2014)

```powershell
# Rename series folder
Rename-Item "D:\Plex\Shows\True Detective" "True Detective (2014)"

# Rename episode files
1..8 | ForEach-Object {
    $ep = $_.ToString("D2")
    Rename-Item "D:\Plex\Shows\True Detective (2014)\Season 1\True Detective - S01E$ep - Remux-1080p.mkv" `
                "True Detective (2014) - S01E$ep - Remux-1080p.mkv"
}
```

### 3. The Dinosaurs (2026)

```powershell
# Rename series folder
Rename-Item "D:\Plex\Shows\The Dinosaurs" "The Dinosaurs (2026)"

# Rename episode files
1..4 | ForEach-Object {
    $ep = $_.ToString("D2")
    Rename-Item "D:\Plex\Shows\The Dinosaurs (2026)\Season 1\The Dinosaurs - S01E$ep - WEB-DL-2160p DV HDR.mkv" `
                "The Dinosaurs (2026) - S01E$ep - WEB-DL-2160p DV HDR.mkv"
}
```

### 4. Vanished (2026)

Folder is correct, only filenames need fixing.

```powershell
1..4 | ForEach-Object {
    $ep = $_.ToString("D2")
    Rename-Item "D:\Plex\Shows\Vanished (2026)\Season 1\Vanished - S01E$ep - WEB-DL-2160p.mkv" `
                "Vanished (2026) - S01E$ep - WEB-DL-2160p.mkv"
}
```

## Phase 2: Database Updates (sqlite3)

```sql
-- ============================================
-- Series paths
-- ============================================
UPDATE series SET path = 'D:/Plex/Shows/The Pitt (2025)' WHERE id = 110;
UPDATE series SET path = 'D:/Plex/Shows/True Detective (2014)' WHERE id = 251;
UPDATE series SET path = 'D:/Plex/Shows/The Dinosaurs (2026)' WHERE id = 252;
UPDATE series SET path = 'D:/Plex/Shows/Vanished (2026)' WHERE id = 248;

-- ============================================
-- The Pitt - S01E01-E15
-- ============================================
UPDATE episode_files SET path = 'D:/Plex/Shows/The Pitt (2025)/Season 1/The Pitt (2025) - S01E01 - WEBDL-1080p.mkv' WHERE id = 2016;
UPDATE episode_files SET path = 'D:/Plex/Shows/The Pitt (2025)/Season 1/The Pitt (2025) - S01E02 - WEBDL-1080p.mkv' WHERE id = 2017;
UPDATE episode_files SET path = 'D:/Plex/Shows/The Pitt (2025)/Season 1/The Pitt (2025) - S01E03 - WEBDL-1080p.mkv' WHERE id = 2018;
UPDATE episode_files SET path = 'D:/Plex/Shows/The Pitt (2025)/Season 1/The Pitt (2025) - S01E04 - WEBDL-1080p.mkv' WHERE id = 2019;
UPDATE episode_files SET path = 'D:/Plex/Shows/The Pitt (2025)/Season 1/The Pitt (2025) - S01E05 - WEBDL-1080p.mkv' WHERE id = 2020;
UPDATE episode_files SET path = 'D:/Plex/Shows/The Pitt (2025)/Season 1/The Pitt (2025) - S01E06 - WEBDL-1080p.mkv' WHERE id = 2021;
UPDATE episode_files SET path = 'D:/Plex/Shows/The Pitt (2025)/Season 1/The Pitt (2025) - S01E07 - WEBDL-1080p.mkv' WHERE id = 2022;
UPDATE episode_files SET path = 'D:/Plex/Shows/The Pitt (2025)/Season 1/The Pitt (2025) - S01E08 - WEBDL-1080p.mkv' WHERE id = 2023;
UPDATE episode_files SET path = 'D:/Plex/Shows/The Pitt (2025)/Season 1/The Pitt (2025) - S01E09 - WEBDL-1080p.mkv' WHERE id = 2024;
UPDATE episode_files SET path = 'D:/Plex/Shows/The Pitt (2025)/Season 1/The Pitt (2025) - S01E10 - WEBDL-1080p.mkv' WHERE id = 2025;
UPDATE episode_files SET path = 'D:/Plex/Shows/The Pitt (2025)/Season 1/The Pitt (2025) - S01E11 - WEBDL-1080p.mkv' WHERE id = 2026;
UPDATE episode_files SET path = 'D:/Plex/Shows/The Pitt (2025)/Season 1/The Pitt (2025) - S01E12 - WEBDL-1080p.mkv' WHERE id = 2027;
UPDATE episode_files SET path = 'D:/Plex/Shows/The Pitt (2025)/Season 1/The Pitt (2025) - S01E13 - WEBDL-1080p.mkv' WHERE id = 2028;
UPDATE episode_files SET path = 'D:/Plex/Shows/The Pitt (2025)/Season 1/The Pitt (2025) - S01E14 - WEBDL-1080p.mkv' WHERE id = 2029;
UPDATE episode_files SET path = 'D:/Plex/Shows/The Pitt (2025)/Season 1/The Pitt (2025) - S01E15 - WEBDL-1080p.mkv' WHERE id = 2030;

-- The Pitt - S02E01-E03
UPDATE episode_files SET path = 'D:/Plex/Shows/The Pitt (2025)/Season 2/The Pitt (2025) - S02E01 - WEBDL-1080p.mkv' WHERE id = 2031;
UPDATE episode_files SET path = 'D:/Plex/Shows/The Pitt (2025)/Season 2/The Pitt (2025) - S02E02 - WEBDL-1080p.mkv' WHERE id = 2032;
UPDATE episode_files SET path = 'D:/Plex/Shows/The Pitt (2025)/Season 2/The Pitt (2025) - S02E03 - WEBDL-1080p.mkv' WHERE id = 2033;

-- The Pitt - S02E04-E06 (already in correct folder, filename needs year)
UPDATE episode_files SET path = 'D:/Plex/Shows/The Pitt (2025)/Season 2/The Pitt (2025) - S02E04 - WEB-DL-2160p.mkv' WHERE id = 2034;
UPDATE episode_files SET path = 'D:/Plex/Shows/The Pitt (2025)/Season 2/The Pitt (2025) - S02E05 - WEB-DL-2160p.mkv' WHERE id = 2387;
UPDATE episode_files SET path = 'D:/Plex/Shows/The Pitt (2025)/Season 2/The Pitt (2025) - S02E06 - WEB-DL-2160p.mkv' WHERE id = 4400;

-- The Pitt - S02E10
UPDATE episode_files SET path = 'D:/Plex/Shows/The Pitt (2025)/Season 2/The Pitt (2025) - S02E10 - WEB-DL-2160p.mkv' WHERE id = 5440;

-- S02E07-E09 (ids 5415-5417) are already correct, no changes needed.

-- ============================================
-- True Detective - S01E01-E08
-- ============================================
UPDATE episode_files SET path = 'D:/Plex/Shows/True Detective (2014)/Season 1/True Detective (2014) - S01E01 - Remux-1080p.mkv' WHERE id = 5375;
UPDATE episode_files SET path = 'D:/Plex/Shows/True Detective (2014)/Season 1/True Detective (2014) - S01E02 - Remux-1080p.mkv' WHERE id = 5376;
UPDATE episode_files SET path = 'D:/Plex/Shows/True Detective (2014)/Season 1/True Detective (2014) - S01E03 - Remux-1080p.mkv' WHERE id = 5377;
UPDATE episode_files SET path = 'D:/Plex/Shows/True Detective (2014)/Season 1/True Detective (2014) - S01E04 - Remux-1080p.mkv' WHERE id = 5378;
UPDATE episode_files SET path = 'D:/Plex/Shows/True Detective (2014)/Season 1/True Detective (2014) - S01E05 - Remux-1080p.mkv' WHERE id = 5379;
UPDATE episode_files SET path = 'D:/Plex/Shows/True Detective (2014)/Season 1/True Detective (2014) - S01E06 - Remux-1080p.mkv' WHERE id = 5380;
UPDATE episode_files SET path = 'D:/Plex/Shows/True Detective (2014)/Season 1/True Detective (2014) - S01E07 - Remux-1080p.mkv' WHERE id = 5381;
UPDATE episode_files SET path = 'D:/Plex/Shows/True Detective (2014)/Season 1/True Detective (2014) - S01E08 - Remux-1080p.mkv' WHERE id = 5382;

-- ============================================
-- The Dinosaurs - S01E01-E04
-- ============================================
UPDATE episode_files SET path = 'D:/Plex/Shows/The Dinosaurs (2026)/Season 1/The Dinosaurs (2026) - S01E01 - WEB-DL-2160p DV HDR.mkv' WHERE id = 5426;
UPDATE episode_files SET path = 'D:/Plex/Shows/The Dinosaurs (2026)/Season 1/The Dinosaurs (2026) - S01E02 - WEB-DL-2160p DV HDR.mkv' WHERE id = 5427;
UPDATE episode_files SET path = 'D:/Plex/Shows/The Dinosaurs (2026)/Season 1/The Dinosaurs (2026) - S01E03 - WEB-DL-2160p DV HDR.mkv' WHERE id = 5428;
UPDATE episode_files SET path = 'D:/Plex/Shows/The Dinosaurs (2026)/Season 1/The Dinosaurs (2026) - S01E04 - WEB-DL-2160p DV HDR.mkv' WHERE id = 5429;

-- ============================================
-- Vanished - S01E01-E04 (filename only, folder correct)
-- ============================================
-- Imported records (backslash paths)
UPDATE episode_files SET path = 'D:/Plex/Shows/Vanished (2026)/Season 1/Vanished (2026) - S01E01 - WEB-DL-2160p.mkv' WHERE id = 4494;
UPDATE episode_files SET path = 'D:/Plex/Shows/Vanished (2026)/Season 1/Vanished (2026) - S01E02 - WEB-DL-2160p.mkv' WHERE id = 4492;
UPDATE episode_files SET path = 'D:/Plex/Shows/Vanished (2026)/Season 1/Vanished (2026) - S01E03 - WEB-DL-2160p.mkv' WHERE id = 4493;
UPDATE episode_files SET path = 'D:/Plex/Shows/Vanished (2026)/Season 1/Vanished (2026) - S01E04 - WEB-DL-2160p.mkv' WHERE id = 4495;

-- Delete duplicate scanned records (same files, detected twice)
DELETE FROM episode_files WHERE id IN (4502, 4503, 4504, 4505);
```

## Phase 3: Verify

After starting SlipStream, confirm each series shows the correct path in the UI and all episode files are detected.
