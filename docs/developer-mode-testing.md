# Developer Mode Manual Testing Guide

This document provides step-by-step test flows for manually testing SlipStream's developer mode features through the frontend UI.

## Prerequisites

- SlipStream backend running (`make dev-backend` or `go run ./cmd/slipstream`)
- SlipStream frontend running (`make dev-frontend` or `cd web && bun run dev`)
- Access to `http://localhost:3000`

---

## 1. Developer Mode Activation

### 1.1 Enable Developer Mode

**Steps:**
1. Navigate to the SlipStream UI
2. Locate the hammer icon in the header (top right area)
3. Click the hammer icon to toggle developer mode ON

**Expected Results:**
- Hammer icon should indicate active state (highlighted/filled)
- Status endpoint (`/api/v1/status`) should return `developerMode: true`
- Application switches to development database (`slipstream_dev.db`)
- Mock services are automatically created:
  - Mock TMDB/TVDB metadata providers
  - Mock indexer
  - Mock download client
  - Mock notification provider
  - Mock root folders (`/mock/movies`, `/mock/tv`)

### 1.2 Disable Developer Mode

**Steps:**
1. Click the hammer icon again to toggle OFF

**Expected Results:**
- Hammer icon returns to inactive state
- Status endpoint returns `developerMode: false`
- Application switches back to production database
- Real metadata providers restored

---

## 2. Mock Metadata Provider Testing

### 2.1 Search for Movies

**Steps:**
1. Enable developer mode
2. Go to Movies > Add New
3. Search for "The Matrix"

**Expected Results:**
- Returns The Matrix (1999) with TMDB ID 603
- Shows poster image (real TMDB poster URL)
- Shows correct year, overview, and rating

**Additional Searches to Test:**
| Search Term | Expected Result |
|-------------|-----------------|
| "Inception" | Inception (2010), TMDB 27205 |
| "Dune" | Dune (2021) and Dune: Part Two (2024) |
| "Oppenheimer" | Oppenheimer (2023), TMDB 872585 |
| "Barbie" | Barbie (2023), TMDB 346698 |

### 2.2 Search for TV Series

**Steps:**
1. Go to Series > Add New
2. Search for "Breaking Bad"

**Expected Results:**
- Returns Breaking Bad with TVDB ID 81189
- Shows poster and overview
- Shows correct number of seasons (5)

**Additional Searches to Test:**
| Search Term | Expected Result |
|-------------|-----------------|
| "Game of Thrones" | TVDB 121361, 8 seasons |
| "Stranger Things" | TVDB 305288, 4 seasons |
| "The Mandalorian" | TVDB 361753, 3 seasons |
| "The Boys" | TVDB 355567 |

### 2.3 Fuzzy Search Fallback

**Steps:**
1. Search for a non-existent title like "xyznonexistent"

**Expected Results:**
- Should return top 10 fallback results from mock data
- Results should be valid movies/shows from the mock database

---

## 3. Mock Indexer Testing

### 3.1 Verify Mock Indexer Created

**Steps:**
1. Enable developer mode
2. Go to Settings > Indexers

**Expected Results:**
- "Mock Indexer" should appear in the list
- Type should show as "mock"
- Status should be enabled

### 3.2 Search for Movie Releases

**Steps:**
1. Add The Matrix (1999) to your library (if not already added)
2. Go to the movie detail page
3. Click "Search" or use the manual search feature

**Expected Results:**
- Returns multiple releases with varied quality:
  - 2160p UHD BluRay Remux with DV/TrueHD
  - 2160p WEB-DL with HDR
  - 1080p BluRay x264
  - 2160p AV1 SDR (new mock variant)
  - 1080p SDR (new mock variant)

### 3.3 Search for TV Releases

**Steps:**
1. Add Breaking Bad to your library
2. Go to the series detail page
3. Search for Season 1 Episode 1

**Expected Results:**
- Returns multiple releases:
  - 2160p Remux options
  - 1080p BluRay options
  - WEB-DL options

### 3.4 Quality Attribute Variety

**Verify the following quality attributes appear in search results:**

| Attribute Type | Values to Find |
|----------------|----------------|
| Resolution | 720p, 1080p, 2160p |
| HDR | DV, HDR10+, HDR10, HDR, SDR |
| Video Codec | x264, x265/HEVC, AV1 |
| Audio Codec | TrueHD, DTS-HD MA, DTS, DDP, DD, AAC |
| Audio Channels | 7.1, 5.1, 2.0 |
| Source | Remux, BluRay, WEB-DL |

---

## 4. Mock Download Client Testing

### 4.1 Verify Mock Download Client Created

**Steps:**
1. Enable developer mode
2. Go to Settings > Download Clients

**Expected Results:**
- "Mock Download Client" should appear in the list
- Type should show as "mock"
- Status should be enabled

### 4.2 Grab a Release

**Steps:**
1. Search for releases (see Section 3.2)
2. Click the download/grab button on any release
3. Go to Queue page

**Expected Results:**
- Release appears in queue
- Status shows "Queued" initially
- After ~2 seconds, status changes to "Downloading"
- Progress increases over ~15 seconds
- After completion, status shows "Seeding"

### 4.3 Download Progress Simulation

**Steps:**
1. Grab a release
2. Watch the queue page

**Expected Timeline:**
- 0-2s: Queued state
- 2-17s: Downloading (progress 0% → 100%)
- 17s+: Seeding (completed)

### 4.4 Pause/Resume Download

**Steps:**
1. While a download is in progress, click Pause
2. Wait a few seconds
3. Click Resume

**Expected Results:**
- Progress pauses when paused
- Progress resumes from same point when resumed
- Time tracking accounts for paused duration

---

## 5. Mock Notification Testing

### 5.1 Verify Mock Notification Created

**Steps:**
1. Enable developer mode
2. Go to Settings > Notifications

**Expected Results:**
- "Mock Notification" should appear in the list
- Should be subscribed to all event types
- Status should be enabled

### 5.2 Test Notification on Grab

**Steps:**
1. Open browser developer tools (F12) > Console
2. Grab a release from search results
3. Watch console for WebSocket messages

**Expected Results:**
- `notification:mock` WebSocket event received
- Event contains:
  - `eventType: "grab"`
  - `title` with release name
  - `data.release` with release info
  - `data.slot` with slot info (if multi-version mode enabled)

### 5.3 Test Notification on Download Complete

**Steps:**
1. Wait for a mock download to complete (seeding state)
2. Watch console for WebSocket messages

**Expected Results:**
- `notification:mock` WebSocket event with `eventType: "download"`
- Contains movie/episode info and quality details

---

## 6. Mock Virtual Filesystem Testing

### 6.1 Browse Mock Root Folders

**Steps:**
1. Enable developer mode
2. Go to Settings > Media Management > Root Folders
3. Click "Add Root Folder"
4. Browse to `/mock`

**Expected Results:**
- Shows three directories: `movies`, `tv`, `downloads`
- Can navigate into each directory
- `/mock/movies` shows movie folders
- `/mock/tv` shows TV show folders

### 6.2 Verify Multi-Version Movie Files

**Steps:**
1. Browse to `/mock/movies/The Matrix (1999)/`

**Expected Results:**
- Shows 3 files:
  - `The.Matrix.1999.2160p.UHD.BluRay.Remux.HEVC.DV.TrueHD.7.1.Atmos-GROUP.mkv` (65GB)
  - `The.Matrix.1999.1080p.BluRay.x264.DTS-HD.MA.5.1-GROUP.mkv` (12GB)
  - `The.Matrix.1999.720p.WEB-DL.x264.AAC.2.0-GROUP.mkv` (4GB)

### 6.3 Verify Multi-Version TV Files

**Steps:**
1. Browse to `/mock/tv/Breaking Bad/Season 01/`

**Expected Results:**
- Shows 14 files (7 episodes × 2 quality tiers):
  - 7 files at 2160p Remux quality
  - 7 files at 1080p BluRay quality

### 6.4 Verify Single-Version Content

**Steps:**
1. Browse to `/mock/movies/Pulp Fiction (1994)/`

**Expected Results:**
- Shows only 1 file (single 1080p version)

### 6.5 Verify Partial TV Content

**Steps:**
1. Browse to `/mock/tv/Stranger Things/Season 04/`

**Expected Results:**
- Shows only 5 episodes (partial season - episodes 1-5 of 9)

---

## 7. Multi-Version (Slots) Mode Testing

### 7.1 Enable Multi-Version Mode

**Steps:**
1. Go to Settings > General (or Multi-Version settings)
2. Enable multi-version mode
3. Configure slots:
   - Slot 1: "Local 4K" - assign a 4K quality profile
   - Slot 2: "Remote 1080p" - assign a 1080p quality profile

### 7.2 Migration Dry-Run Preview

**Steps:**
1. Before enabling, click "Preview Migration"
2. Review the proposed file assignments

**Expected Results:**
- The Matrix shows 3 files with proposed slot assignments
- 2160p file → Slot 1 (Local 4K)
- 1080p file → Slot 2 (Remote 1080p)
- 720p file → Review Queue (more files than slots)

### 7.3 Slot Assignment Testing

**Steps:**
1. Enable multi-version mode
2. Run migration
3. Go to The Matrix movie detail page

**Expected Results:**
- Shows slot status for each configured slot
- Slot 1 shows 4K file assigned
- Slot 2 shows 1080p file assigned
- 720p file in review queue

### 7.4 Grab with Slot Target

**Steps:**
1. Add a movie without files (e.g., Oppenheimer)
2. Search for releases
3. Grab a release for Slot 1

**Expected Results:**
- Queue shows download with target slot indicator
- Notification includes slot context (`slot.id`, `slot.name`)

### 7.5 Slot-Specific Search

**Steps:**
1. Go to a movie with empty slots
2. Click "Search" for a specific slot

**Expected Results:**
- Results filtered/scored for slot's quality profile
- Best matches for that slot's requirements shown first

### 7.6 Review Queue Testing

**Steps:**
1. Use The Matrix (has 3 files but only 2 slots)
2. Go to Multi-Version > Review Queue

**Expected Results:**
- 720p file appears in review queue
- Options to:
  - Assign to a slot (replacing existing)
  - Delete the file
  - Ignore

---

## 8. Integration Flow Testing

### 8.1 Full Download Flow

**Test the complete flow from search to import:**

1. Enable developer mode
2. Add a new movie (e.g., "Oppenheimer")
3. Search for releases
4. Grab a 2160p release
5. Watch queue for progress
6. Verify notification on grab
7. Wait for download completion (~17 seconds)
8. Verify notification on download complete
9. Check movie now shows as "downloaded"

### 8.2 Upgrade Flow

**Test upgrading an existing file:**

1. Have a movie with 1080p file in Slot 2
2. Search for releases
3. Grab a 2160p release targeting Slot 1
4. Wait for download
5. Verify upgrade notification includes slot context
6. Verify Slot 1 now has 4K file

### 8.3 TV Episode Flow

**Test TV-specific functionality:**

1. Add Breaking Bad series
2. Go to Season 1
3. Verify episodes show existing files (from mock VFS)
4. Search for missing/upgrade releases
5. Grab a release for a specific episode
6. Verify download and import

---

## 9. Error Scenarios

### 9.1 Indexer Disabled State

**Steps:**
1. Note: Mock indexer doesn't support disable simulation
2. Test with real indexer in production mode if needed

### 9.2 Download Client Failure

**Steps:**
1. Note: Mock client always succeeds
2. To test failures, use real clients or manipulate database

---

## 10. Cleanup

### 10.1 Disable Developer Mode

**Steps:**
1. Click hammer icon to disable
2. Verify switch back to production database
3. Verify mock services no longer appear in settings

### 10.2 Clear Dev Database

**Steps:**
1. Delete `slipstream_dev.db` file manually if needed
2. Re-enable developer mode for fresh start

---

## Quick Reference: Mock Data Summary

### Movies with Multi-Version Files
| Movie | Quality Tiers |
|-------|---------------|
| The Matrix (1999) | 4K DV, 1080p, 720p |
| Inception (2010) | 4K HDR10, 1080p |
| Dune (2021) | 4K DV+HDR10, 1080p |
| Pulp Fiction (1994) | 1080p only |
| Fight Club (1999) | 1080p only |

### Movies Without Files (Search Testing)
- Oppenheimer (2023)
- Barbie (2023)
- Dune: Part Two (2024)

### TV with Multi-Version Episodes
| Series | Season | Quality Tiers |
|--------|--------|---------------|
| Breaking Bad | S01 | 4K Remux, 1080p |
| Breaking Bad | S02-S05 | 1080p only |
| Game of Thrones | S01 | 4K HDR10, 1080p |
| Game of Thrones | S02-S03 | 1080p only |

### TV with Partial/Missing Content
| Series | Status |
|--------|--------|
| Stranger Things S04 | 5 of 9 episodes |
| The Mandalorian S03 | Missing entirely |
| The Boys | No files (search only) |
| The Simpsons | No files (search only) |

### Indexer Quality Variants
- Standard: HDR (DV, HDR10+, HDR10), x264, x265
- New: AV1 codec, explicit SDR variants
