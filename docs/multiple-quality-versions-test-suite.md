# Multiple Quality Versions - Manual Test Suite

This document provides a comprehensive manual test suite to validate the Multiple Quality Versions feature implementation.

---

## Prerequisites

### Environment Setup
- [ ] SlipStream backend running on `:8080`
- [ ] SlipStream frontend running on `:3000`
- [ ] Developer mode enabled (`DEVELOPER_MODE=true` in `.env`)
- [ ] At least one indexer configured
- [ ] At least one download client configured
- [ ] At least one root folder configured
- [ ] TMDB API key configured

### Test Data Requirements
- [ ] At least 3-5 movies in library with files
- [ ] At least 1 TV series with multiple seasons/episodes
- [ ] Access to test release names (provided in test cases below)

### Quality Profiles to Create
Before testing, create the following quality profiles:

**Profile: "4K HDR"**
- Qualities: 4K Remux, 4K BluRay, 4K WEB-DL (ranked in order)
- Cutoff: 4K BluRay
- HDR Settings (per-item modes):
  - DV: Required
  - HDR10+: Required
  - HDR10: Required
  - HDR: Required
  - SDR: Not Allowed
- Video Codec: x265 = Preferred

**Profile: "1080p SDR"**
- Qualities: 1080p Remux, 1080p BluRay, 1080p WEB-DL (ranked in order)
- Cutoff: 1080p BluRay
- HDR Settings (per-item modes):
  - SDR: Required
  - DV: Not Allowed
  - HDR10: Not Allowed
- Video Codec: x264 = Preferred, x265 = Preferred

**Profile: "720p Compatibility"**
- Qualities: 720p BluRay, 720p WEB-DL, 720p HDTV (ranked in order)
- Cutoff: 720p WEB-DL
- HDR Settings: SDR = Required
- Audio Codec: AAC = Required, DD = Required

---

## Test Suite 1: Quality Profile Extensions (Group A)

### Test A1: Profile Attribute Settings UI
**Objective**: Verify all attribute settings are available in profile editor with per-item mode selection

**Steps**:
1. Navigate to Settings → Quality Profiles
2. Click "Add Profile" or edit existing profile
3. Scroll to "Attribute Filters" section

**Expected Results**:
- [ ] Four collapsible attribute sections: HDR Format, Video Codec, Audio Codec, Audio Channels
- [ ] Each section header shows summary badges (e.g., "2 required", "1 preferred", "1 blocked")
- [ ] Sections with no settings show "Any" text
- [ ] Expanding HDR Format section shows per-item mode dropdowns for: DV, HDR10+, HDR10, HDR, HLG, SDR
- [ ] Expanding Video Codec section shows per-item mode dropdowns for: x264, x265, AV1, VP9, XviD, DivX, MPEG2
- [ ] Expanding Audio Codec section shows per-item mode dropdowns for: TrueHD, DTS-HD MA, DTS-HD, DTS, DDP, DD, AAC, FLAC, LPCM, Opus, MP3
- [ ] Expanding Audio Channels section shows per-item mode dropdowns for: 7.1, 5.1, 2.0, 1.0
- [ ] Each item has dropdown with four options: Any, Preferred, Required, Not Allowed

### Test A2: Profile Attribute Persistence
**Objective**: Verify per-item attribute settings save and load correctly

**Steps**:
1. Create new profile "Test Profile A2"
2. Expand HDR Format section:
   - Set DV to Required
   - Set HDR10 to Required
   - Set SDR to Not Allowed
3. Expand Video Codec section:
   - Set x265 to Preferred
4. Expand Audio Codec section:
   - Set TrueHD to Required
   - Set DTS-HD MA to Required
5. Leave Audio Channels at defaults (all Any)
6. Save profile
7. Close dialog
8. Reopen profile for editing

**Expected Results**:
- [ ] HDR Format header shows "2 required", "1 blocked" badges
- [ ] Expanding HDR shows DV=Required, HDR10=Required, SDR=Not Allowed, others=Any
- [ ] Video Codec header shows "1 preferred" badge
- [ ] Expanding Video Codec shows x265=Preferred, others=Any
- [ ] Audio Codec header shows "2 required" badge
- [ ] Expanding Audio Codec shows TrueHD=Required, DTS-HD MA=Required, others=Any
- [ ] Audio Channels header shows "Any" text (no badges)

### Test A3: Attribute API Endpoint
**Objective**: Verify attributes endpoint returns all options and modes

**Steps**:
1. Open browser dev tools or use curl
2. Call `GET /api/v1/qualityprofiles/attributes`

**Expected Results**:
```json
{
  "hdrFormats": ["DV", "HDR10+", "HDR10", "HDR", "HLG", "SDR"],
  "videoCodecs": ["x264", "x265", "AV1", "VP9", "XviD", "DivX", "MPEG2"],
  "audioCodecs": ["TrueHD", "DTS-HD MA", "DTS-HD", "DTS", "DDP", "DD", "AAC", "FLAC", "LPCM", "Opus", "MP3"],
  "audioChannels": ["7.1", "5.1", "2.0", "1.0"],
  "modes": ["any", "preferred", "required", "notAllowed"]
}
```

### Test A4: Not Allowed Mode Functionality
**Objective**: Verify "Not Allowed" mode rejects releases with blocked attributes

**Steps**:
1. Create profile with HDR Format: DV = Not Allowed
2. Use debug tools to test release: `Movie.2024.2160p.BluRay.DV.x265-GROUP`
3. Test another release: `Movie.2024.2160p.BluRay.HDR10.x265-GROUP`

**Expected Results**:
- [ ] First release (with DV) does NOT match profile
- [ ] Second release (with HDR10, no DV) DOES match profile
- [ ] Match failure reason indicates "DV is not allowed"

### Test A5: Mixed Per-Item Modes
**Objective**: Verify different modes work together on same attribute category

**Steps**:
1. Create profile with HDR Format:
   - DV = Required
   - HDR10 = Preferred
   - SDR = Not Allowed
2. Test releases with various HDR formats

**Expected Results**:
- [ ] Release with DV only: Matches (required satisfied)
- [ ] Release with DV + HDR10: Matches with bonus score (required + preferred)
- [ ] Release with HDR10 only: Does NOT match (required DV missing)
- [ ] Release with SDR: Does NOT match (blocked)

---

## Test Suite 2: Mutual Exclusivity (Group B)

### Test B1: Exclusivity Check - Conflicting HDR
**Objective**: Verify system detects HDR conflicts between profiles

**Steps**:
1. Navigate to Settings → Quality Profiles
2. Call `POST /api/v1/qualityprofiles/check-exclusivity` with:
```json
{
  "profileIds": [<4K HDR profile ID>, <1080p SDR profile ID>]
}
```

**Expected Results**:
- [ ] Response indicates profiles ARE mutually exclusive
- [ ] Conflict reason mentions HDR (one requires HDR formats, other requires SDR)

### Test B2: Exclusivity Check - Non-Conflicting (Should Fail)
**Objective**: Verify system rejects profiles without required conflicts

**Steps**:
1. Create two profiles with only "Preferred" attributes (no Required or Not Allowed)
   - Profile X: DV = Preferred
   - Profile Y: HDR10 = Preferred
2. Call exclusivity check API with both profile IDs

**Expected Results**:
- [ ] Response indicates profiles are NOT mutually exclusive
- [ ] Message explains preferred attributes don't create exclusivity

### Test B3: Preferred Attributes Don't Affect Exclusivity
**Objective**: Verify preferred mode is ignored in exclusivity calculation

**Steps**:
1. Create Profile X: DV = Required, x265 = Preferred
2. Create Profile Y: SDR = Required, x265 = Preferred
3. Check exclusivity between X and Y

**Expected Results**:
- [ ] Profiles ARE mutually exclusive (HDR conflict: DV required vs SDR required)
- [ ] Video codec (both prefer x265) does NOT prevent exclusivity

### Test B4: Required vs Not Allowed Creates Exclusivity
**Objective**: Verify required + notAllowed conflict detection

**Steps**:
1. Create Profile X: DV = Required
2. Create Profile Y: DV = Not Allowed, SDR = Required
3. Check exclusivity between X and Y

**Expected Results**:
- [ ] Profiles ARE mutually exclusive
- [ ] Conflict reason: Profile X requires DV, Profile Y blocks DV

---

## Test Suite 3: Slot Infrastructure (Group C)

### Test C1: Slot Configuration Page
**Objective**: Verify slot configuration UI exists and functions

**Steps**:
1. Navigate to Settings → Version Slots (or Media Management)
2. Observe the slot configuration interface

**Expected Results**:
- [ ] Master toggle for "Enable Multi-Version Mode" visible
- [ ] 3 slot cards displayed (Slot 1, Slot 2, Slot 3)
- [ ] Each slot has: Name field, Enable toggle, Profile dropdown
- [ ] Slot 1 enabled by default, Slots 2-3 disabled

### Test C2: Slot Name Customization
**Objective**: Verify slot names can be customized

**Steps**:
1. On slot configuration page, change Slot 1 name to "Local 4K"
2. Change Slot 2 name to "Remote Streaming"
3. Save configuration
4. Refresh page

**Expected Results**:
- [ ] Slot names persist after refresh
- [ ] Names appear in slot cards

### Test C3: Slot Profile Assignment
**Objective**: Verify profiles can be assigned to slots

**Steps**:
1. Assign "4K HDR" profile to Slot 1
2. Assign "1080p SDR" profile to Slot 2
3. Enable Slot 2
4. Attempt to save

**Expected Results**:
- [ ] Profile dropdowns show all available profiles
- [ ] Profiles assign correctly to slots
- [ ] If profiles are mutually exclusive, save succeeds
- [ ] If profiles overlap, error message displayed

### Test C4: Master Toggle - Dry Run Requirement
**Objective**: Verify dry run is required before enabling multi-version

**Steps**:
1. With multi-version disabled, try to enable the master toggle
2. Do NOT run dry run preview first

**Expected Results**:
- [ ] System prevents enabling
- [ ] Message indicates dry run preview must be completed first

### Test C5: File Naming Validation
**Objective**: Verify filename format validation for differentiators

**Steps**:
1. Configure Slot 1 with "4K HDR" profile (HDR required)
2. Configure Slot 2 with "1080p SDR" profile (SDR required)
3. Enable both slots
4. Attempt to validate/save configuration

**Expected Results**:
- [ ] If filename format lacks HDR token, warning displayed
- [ ] Warning suggests adding `{MediaInfo VideoDynamicRange}` token
- [ ] System identifies which differentiators are missing

---

## Test Suite 4: Debug Tools (Developer Mode)

### Test D1: Parse Release Tester
**Objective**: Verify release parsing debug tool works

**Steps**:
1. Ensure Developer Mode is enabled
2. Navigate to Settings → Version Slots
3. Find "Parse Release" debug panel (may need to expand)
4. Enter release name: `Movie.Name.2024.2160p.UHD.BluRay.Remux.DV.HDR10.HEVC.TrueHD.Atmos.7.1-GROUP`
5. Click Parse

**Expected Results**:
- [ ] Debug panel visible only in developer mode
- [ ] Parsed results show:
  - Resolution: 2160p/4K
  - Source: BluRay Remux
  - HDR: [DV, HDR10]
  - Video Codec: x265/HEVC
  - Audio Codec: TrueHD
  - Audio Channels: 7.1

### Test D2: Profile Match Tester
**Objective**: Verify profile matching debug tool works

**Steps**:
1. In debug panel, find "Profile Match Tester"
2. Enter release: `Movie.2024.1080p.WEB-DL.x264.AAC.5.1-GROUP`
3. Select "1080p SDR" profile
4. Click Test Match

**Expected Results**:
- [ ] Shows match result (should match)
- [ ] Displays attribute match breakdown
- [ ] Shows quality score calculation

### Test D3: Simulate Import
**Objective**: Verify import simulation debug tool works

**Steps**:
1. In debug panel, find "Simulate Import"
2. Enter release: `Movie.2024.2160p.WEB-DL.DV.x265.DDP.5.1-GROUP`
3. Click Simulate

**Expected Results**:
- [ ] Shows which slot the release would be assigned to
- [ ] Shows match scores for each enabled slot
- [ ] Indicates if it would be new fill or upgrade

---

## Test Suite 5: Migration (Group I)

### Test M1: Migration Dry Run Preview
**Objective**: Verify migration preview shows correct information

**Steps**:
1. Configure slots with mutually exclusive profiles
2. Have existing movies with files in library
3. Click "Preview Migration" or equivalent

**Expected Results**:
- [ ] Preview organized by Movies and TV Shows sections
- [ ] Each movie shows current files with proposed slot assignment
- [ ] Match scores displayed for each file
- [ ] Conflicts highlighted (files that don't match any slot)
- [ ] TV shows show per-series, per-season breakdown

### Test M2: Migration Execution
**Objective**: Verify migration assigns files to slots correctly

**Steps**:
1. After successful dry run preview
2. Click "Execute Migration" or equivalent
3. Verify file assignments

**Expected Results**:
- [ ] Files assigned to appropriate slots based on quality match
- [ ] Unmatched files sent to review queue
- [ ] Multi-version mode can now be enabled
- [ ] Movie/episode detail pages show slot assignments

### Test M3: Review Queue
**Objective**: Verify review queue displays and resolves items

**Steps**:
1. If any files were unmatched during migration
2. Navigate to review queue
3. Select a file

**Expected Results**:
- [ ] Review queue shows all unmatched files
- [ ] File details displayed (quality, size, path)
- [ ] Available slot options shown
- [ ] Can assign to slot or delete file

---

## Test Suite 6: Assignment Logic (Group D) - Test Matrix

### Test T1: Import File Matching Slot 1, All Slots Empty
**Objective**: Verify file matching Slot 1 profile is assigned to Slot 1

**Setup**:
- Movie with no files
- Slot 1: "4K HDR" profile, enabled
- Slot 2: "1080p SDR" profile, enabled

**Steps**:
1. Import/scan file: `Movie.2024.2160p.BluRay.Remux.DV.x265.TrueHD-GROUP`

**Expected Results**:
- [ ] File assigned to Slot 1
- [ ] Slot 1 shows as filled
- [ ] Slot 2 remains empty

### Test T2: Import File Matching Slot 2, All Slots Empty
**Objective**: Verify file matching Slot 2 profile is assigned to Slot 2

**Setup**:
- Movie with no files
- Slot 1: "4K HDR" profile, enabled
- Slot 2: "1080p SDR" profile, enabled

**Steps**:
1. Import/scan file: `Movie.2024.1080p.BluRay.x264.DTS-GROUP`

**Expected Results**:
- [ ] File assigned to Slot 2
- [ ] Slot 2 shows as filled
- [ ] Slot 1 remains empty

### Test T3: Import File Matching Both Slots Equally, All Empty
**Objective**: Verify equal scores assign to first empty slot

**Setup**:
- Two profiles with identical "any" settings (hard to create truly equal)
- Alternative: Use debug tools to verify scoring

**Steps**:
1. Use "Simulate Import" debug tool
2. Enter release that could match multiple slots equally

**Expected Results**:
- [ ] File assigned to first empty slot (Slot 1)

### Test T4: Import File Matching Slot 1, Slot 1 Filled, Slot 2 Empty
**Objective**: Verify prompt for upgrade vs new slot decision

**Setup**:
- Movie with existing file in Slot 1 (lower quality)
- Slot 1: "4K HDR" profile, enabled
- Slot 2: "1080p SDR" profile, enabled

**Steps**:
1. Manually import file that matches Slot 1 profile but is better quality
2. Observe system behavior

**Expected Results**:
- [ ] System prompts user: Upgrade Slot 1 or Assign to different slot
- [ ] User can choose action

### Test T5: Import File Below All Profiles, All Slots Empty
**Objective**: Verify below-profile files accepted to closest slot when all empty

**Setup**:
- Movie with no files
- Both slots require quality higher than test file

**Steps**:
1. Import file: `Movie.2024.480p.WEB-DL.x264-GROUP` (below all profiles)

**Expected Results**:
- [ ] File accepted to Slot 1 (closest match)
- [ ] Slot shows "Upgrade Needed" indicator
- [ ] Missing status may still show if upgrade needed

### Test T6: Import File Below All Profiles, Some Slots Filled
**Objective**: Verify below-profile files rejected when some slots filled

**Setup**:
- Movie with file in Slot 1
- Slot 2 empty

**Steps**:
1. Try to import file: `Movie.2024.480p.WEB-DL.x264-GROUP` (below all profiles)

**Expected Results**:
- [ ] Import rejected
- [ ] Error message indicates file doesn't meet any profile requirements

### Test T7: Upgrade Within Slot
**Objective**: Verify upgrade replaces and deletes old file

**Setup**:
- Movie with 1080p file in Slot 2
- Slot 2 profile cutoff is BluRay

**Steps**:
1. Find/grab better quality 1080p release (e.g., BluRay vs WEB-DL)
2. Complete download and import

**Expected Results**:
- [ ] New file replaces old in Slot 2
- [ ] Old file deleted from disk
- [ ] History shows upgrade event

### Test T8: Auto-Search with 2 Empty Monitored Slots
**Objective**: Verify parallel search for multiple slots

**Setup**:
- Movie with both slots empty and monitored
- Indexer configured

**Steps**:
1. Trigger auto-search for movie (or wait for scheduled search)
2. Monitor activity/logs

**Expected Results**:
- [ ] Search executes for both slots
- [ ] May grab releases for both slots simultaneously
- [ ] Queue shows items with different target slots

### Test T9: Auto-Search with 1 Monitored, 1 Unmonitored Empty Slot
**Objective**: Verify only monitored slots are searched

**Setup**:
- Movie with Slot 1 empty and monitored
- Slot 2 empty but unmonitored

**Steps**:
1. Trigger auto-search for movie

**Expected Results**:
- [ ] Search only executes for Slot 1
- [ ] No search attempts for Slot 2

### Test T10: Disable Slot with Files Assigned
**Objective**: Verify prompt when disabling slot with files

**Setup**:
- Movie with file in Slot 2
- Slot 2 currently enabled

**Steps**:
1. Attempt to disable Slot 2 in settings

**Expected Results**:
- [ ] System prompts for action
- [ ] Options: Delete files, Keep unassigned, Cancel
- [ ] Choosing "Keep" leaves files but removes slot assignment

### Test T11: Change Slot Profile with Files Assigned
**Objective**: Verify prompt when changing profile after files assigned

**Setup**:
- Movies with files assigned to Slot 1
- Slot 1 has "4K HDR" profile

**Steps**:
1. Change Slot 1 profile to different profile

**Expected Results**:
- [ ] System prompts for action
- [ ] Options: Keep current assignments, Re-evaluate files, Cancel
- [ ] Re-evaluate may move files to review queue if no longer matching

### Test T12: Scan Folder with More Files Than Slots
**Objective**: Verify extra files go to review queue

**Setup**:
- Movie folder with 4 video files (more than 3 slots)
- All slots enabled

**Steps**:
1. Trigger library scan or rescan movie

**Expected Results**:
- [ ] 3 best-matching files assigned to slots
- [ ] 4th file appears in review queue
- [ ] User can manually assign or delete extra file

### Test T13: DV+HDR10 File, Slot 1 Requires DV, Slot 2 Requires HDR10
**Objective**: Verify combo HDR file matches profiles requiring either format

**Setup**:
- Slot 1 profile: DV = Required
- Slot 2 profile: HDR10 = Required

**Steps**:
1. Use debug tool to test: `Movie.2024.2160p.BluRay.DV.HDR10.x265-GROUP`

**Expected Results**:
- [ ] File matches BOTH slots (combo format contains both)
- [ ] Assigned to higher priority slot (Slot 1 for DV)

### Test T14: Multi-Audio Remux, Profile Requires TrueHD
**Objective**: Verify multi-track audio matches if ANY track satisfies

**Setup**:
- Profile with TrueHD = Required

**Steps**:
1. Test release: `Movie.2024.2160p.Remux.DTS-HD.MA.TrueHD.Atmos-GROUP`

**Expected Results**:
- [ ] File matches profile (TrueHD track present)
- [ ] Match succeeds despite DTS-HD MA also being present

### Test T15: Unknown Codec, Profile Requires x265
**Objective**: Verify unknown attributes fail required checks

**Setup**:
- Profile with x265 = Required

**Steps**:
1. Test release with no codec info: `Movie.2024.2160p.BluRay-GROUP`

**Expected Results**:
- [ ] File does NOT match profile
- [ ] Unknown codec fails required check

### Test T19: Not Allowed Blocks Release
**Objective**: Verify Not Allowed mode rejects matching releases

**Setup**:
- Profile with DV = Not Allowed, HDR10 = Required

**Steps**:
1. Test release: `Movie.2024.2160p.BluRay.DV.HDR10.x265-GROUP`

**Expected Results**:
- [ ] File does NOT match profile (DV is blocked even though HDR10 is present)
- [ ] Match failure indicates "DV is not allowed"

### Test T16: TV Season Pack with Episodes in Different Slots
**Objective**: Verify season pack episodes individually assessed

**Setup**:
- Series with some episodes already in Slot 1
- Season pack for same season

**Steps**:
1. Grab season pack release
2. Import season pack

**Expected Results**:
- [ ] Each episode evaluated individually
- [ ] Episodes may go to different slots based on existing files
- [ ] Summary shows mixed slot assignment

### Test T17: Missing Status with 1 Monitored Slot Empty, 1 Filled
**Objective**: Verify missing status when any monitored slot empty

**Setup**:
- Movie with Slot 1 filled, Slot 2 empty
- Both slots monitored

**Steps**:
1. View movie in library
2. Check status indicators

**Expected Results**:
- [ ] Movie shows as "Missing" or partial
- [ ] Missing indicator reflects empty monitored slot

### Test T18: Missing Status with 1 Unmonitored Slot Empty
**Objective**: Verify unmonitored empty slots don't affect missing status

**Setup**:
- Movie with Slot 1 filled, Slot 2 empty
- Slot 1 monitored, Slot 2 unmonitored

**Steps**:
1. View movie in library
2. Check status indicators

**Expected Results**:
- [ ] Movie shows as "Available" (not missing)
- [ ] Unmonitored empty slot doesn't trigger missing status

---

## Test Suite 7: Search Results Display (Group F)

### Test S1: Search Results Show Target Slot
**Objective**: Verify search results indicate which slot each release would fill

**Steps**:
1. Open movie detail page
2. Click Search/Manual Search
3. View search results

**Expected Results**:
- [ ] Each result shows target slot badge (e.g., "Slot 1: Local 4K")
- [ ] Slot icon (layers) visible
- [ ] Tooltip explains slot assignment

### Test S2: Search Results Show Upgrade vs New Fill
**Objective**: Verify upgrade/new indicators in search results

**Setup**:
- Movie with file in Slot 1
- Slot 2 empty

**Steps**:
1. Search for movie
2. Observe results for both slots

**Expected Results**:
- [ ] Results for Slot 1 show "Upgrade" indicator (up arrow)
- [ ] Results for Slot 2 show "New" indicator (green badge)

### Test S3: Override Target Slot When Grabbing
**Objective**: Verify user can override auto-detected slot

**Steps**:
1. Search for movie
2. Find release auto-assigned to Slot 1
3. Grab with override to Slot 2 (if UI supports)

**Expected Results**:
- [ ] Grab request accepts target slot parameter
- [ ] Download queued with specified slot target
- [ ] File imports to overridden slot

---

## Test Suite 8: Queue Display (Group H)

### Test Q1: Queue Shows Target Slot Info
**Objective**: Verify download queue displays slot information

**Steps**:
1. Grab releases for different slots
2. Navigate to Queue/Activity page

**Expected Results**:
- [ ] Each queue item shows target slot name
- [ ] Slot displayed inline with download info
- [ ] Different slots clearly distinguished

### Test Q2: Failed Download Clears Slot
**Objective**: Verify failed downloads revert slot to empty

**Setup**:
- Grab release for empty slot
- Slot shows as "downloading"

**Steps**:
1. Simulate download failure (remove from client, reject import)
2. Check slot status

**Expected Results**:
- [ ] Slot reverts to "empty" status
- [ ] No pending/retry state shown
- [ ] Slot available for next search

---

## Test Suite 9: History Logging

### Test H1: Slot Assignment Logged
**Objective**: Verify slot assignments appear in history

**Steps**:
1. Import file to slot
2. Navigate to History page

**Expected Results**:
- [ ] History entry for slot assignment
- [ ] Entry shows slot name and file info
- [ ] Event type indicates "Slot Assigned"

### Test H2: Slot Reassignment Logged
**Objective**: Verify slot reassignments appear in history

**Steps**:
1. Reassign file from Slot 1 to Slot 2 (via movie detail)
2. Check history

**Expected Results**:
- [ ] History entry for reassignment
- [ ] Shows previous slot and new slot
- [ ] Event type indicates "Slot Reassigned"

---

## Test Suite 10: TV-Specific Features (Group J)

### Test TV1: Episode Independence
**Objective**: Verify episodes track slots independently

**Steps**:
1. Navigate to series with multiple episodes
2. Import files to different slots for different episodes

**Expected Results**:
- [ ] Episode 1 can have Slot 1 filled, Slot 2 empty
- [ ] Episode 2 can have Slot 2 filled, Slot 1 empty
- [ ] Season list shows mixed slot status

### Test TV2: Season Pack Mixed Slots
**Objective**: Verify season pack can result in mixed slot assignments

**Setup**:
- Configure slots with different quality profiles

**Steps**:
1. Grab season pack
2. Import season pack
3. View episode assignments

**Expected Results**:
- [ ] Episodes from pack may go to different slots
- [ ] Based on individual episode quality matching
- [ ] No requirement for all episodes in same slot

### Test TV3: Per-Episode Slot Monitoring
**Objective**: Verify slots can be monitored per-episode

**Steps**:
1. Navigate to series detail
2. Find slot monitoring controls
3. Monitor Slot 1 for Episode 1, unmonitor for Episode 2

**Expected Results**:
- [ ] Monitoring toggles work per-episode per-slot
- [ ] Auto-search respects per-episode monitoring

---

## Test Suite 11: UI Display (Group L)

### Test UI1: Movie Detail Slot Display
**Objective**: Verify movie detail shows files with slot assignments

**Steps**:
1. Navigate to movie with files in multiple slots
2. View files section

**Expected Results**:
- [ ] Files displayed in table/list format
- [ ] Slot column shows slot name for each file
- [ ] Can reassign file to different slot via dropdown
- [ ] Slot status cards show filled/empty state

### Test UI2: Series Detail Slot Display
**Objective**: Verify series detail shows slot information

**Steps**:
1. Navigate to series with episodes
2. Expand season to view episodes

**Expected Results**:
- [ ] Episode table shows slot status
- [ ] Files associated with correct slots
- [ ] Can manage slot assignments per episode

### Test UI3: Settings Page Location
**Objective**: Verify slot configuration in correct location

**Steps**:
1. Navigate to Settings
2. Find Version Slots / Media Management section

**Expected Results**:
- [ ] Slot configuration accessible from settings
- [ ] Clear navigation path
- [ ] All slot management features available

---

## Test Suite 12: API Endpoints (Group K)

### Test API1: Slot CRUD
**Objective**: Verify slot management API endpoints

**Steps**:
```bash
# List slots
curl -X GET http://localhost:8080/api/v1/slots

# Get single slot
curl -X GET http://localhost:8080/api/v1/slots/1

# Update slot name
curl -X PUT http://localhost:8080/api/v1/slots/1 \
  -H "Content-Type: application/json" \
  -d '{"name": "Test Name"}'

# Enable slot
curl -X PUT http://localhost:8080/api/v1/slots/1/enabled \
  -H "Content-Type: application/json" \
  -d '{"enabled": true}'

# Set profile
curl -X PUT http://localhost:8080/api/v1/slots/1/profile \
  -H "Content-Type: application/json" \
  -d '{"qualityProfileId": 1}'
```

**Expected Results**:
- [ ] All endpoints return appropriate responses
- [ ] Updates persist correctly
- [ ] Validation errors returned when appropriate

### Test API2: Movie Slot Operations
**Objective**: Verify movie slot assignment API

**Steps**:
```bash
# Get movie slot assignments
curl -X GET http://localhost:8080/api/v1/slots/movies/1/assignments

# Get movie slot status
curl -X GET http://localhost:8080/api/v1/slots/movies/1/status

# Assign file to slot
curl -X POST http://localhost:8080/api/v1/slots/movies/1/slots/1/assign \
  -H "Content-Type: application/json" \
  -d '{"fileId": 1}'

# Set slot monitoring
curl -X PUT http://localhost:8080/api/v1/slots/movies/1/slots/1/monitored \
  -H "Content-Type: application/json" \
  -d '{"monitored": true}'
```

**Expected Results**:
- [ ] Assignments returned correctly
- [ ] Status reflects current state
- [ ] File assignment works
- [ ] Monitoring toggle works

### Test API3: Grab with Target Slot
**Objective**: Verify grab API accepts target slot

**Steps**:
```bash
curl -X POST http://localhost:8080/api/v1/search/grab \
  -H "Content-Type: application/json" \
  -d '{
    "guid": "release-guid",
    "indexerId": 1,
    "movieId": 1,
    "targetSlotId": 2
  }'
```

**Expected Results**:
- [ ] Grab succeeds with target slot
- [ ] Download mapping includes slot info
- [ ] Queue item shows target slot

---

## Test Completion Checklist

### Group A: Quality Profile Extensions
- [ ] A1: Attribute settings UI (per-item dropdowns, collapsible sections)
- [ ] A2: Attribute persistence (per-item modes)
- [ ] A3: Attributes API endpoint (includes modes)
- [ ] A4: Not Allowed mode functionality
- [ ] A5: Mixed per-item modes

### Group B: Mutual Exclusivity
- [ ] B1: Conflicting HDR detection
- [ ] B2: Non-conflicting rejection
- [ ] B3: Preferred attributes ignored
- [ ] B4: Required vs Not Allowed conflict

### Group C: Slot Infrastructure
- [ ] C1: Configuration page
- [ ] C2: Name customization
- [ ] C3: Profile assignment
- [ ] C4: Dry run requirement
- [ ] C5: Filename validation

### Group D: Debug Tools
- [ ] D1: Parse Release tester
- [ ] D2: Profile Match tester
- [ ] D3: Simulate Import

### Group I: Migration
- [ ] M1: Dry run preview
- [ ] M2: Migration execution
- [ ] M3: Review queue

### Test Matrix (T1-T19)
- [ ] T1: Slot 1 match, all empty
- [ ] T2: Slot 2 match, all empty
- [ ] T3: Equal match, all empty
- [ ] T4: Slot 1 match, Slot 1 filled
- [ ] T5: Below profile, all empty
- [ ] T6: Below profile, some filled
- [ ] T7: Upgrade within slot
- [ ] T8: Auto-search 2 monitored
- [ ] T9: Auto-search 1 monitored
- [ ] T10: Disable slot with files
- [ ] T11: Change profile with files
- [ ] T12: More files than slots
- [ ] T13: Combo HDR matching
- [ ] T14: Multi-audio matching
- [ ] T15: Unknown codec
- [ ] T16: Season pack mixed
- [ ] T17: Missing with monitored empty
- [ ] T18: Missing with unmonitored empty
- [ ] T19: Not Allowed blocks release

### Group F: Search Results
- [ ] S1: Target slot display
- [ ] S2: Upgrade vs new indicator
- [ ] S3: Override target slot

### Group H: Queue
- [ ] Q1: Queue slot info
- [ ] Q2: Failed download cleanup

### History
- [ ] H1: Assignment logged
- [ ] H2: Reassignment logged

### Group J: TV-Specific
- [ ] TV1: Episode independence
- [ ] TV2: Season pack mixed
- [ ] TV3: Per-episode monitoring

### Group L: UI
- [ ] UI1: Movie detail display
- [ ] UI2: Series detail display
- [ ] UI3: Settings location

### Group K: API
- [ ] API1: Slot CRUD
- [ ] API2: Movie operations
- [ ] API3: Grab with target slot

---

## Test Report Template

**Tester Name**: _______________
**Date**: _______________
**SlipStream Version**: _______________
**Environment**: _______________

| Test ID | Status | Notes |
|---------|--------|-------|
| A1 | Pass/Fail | |
| A2 | Pass/Fail | |
| ... | ... | ... |

**Issues Found**:
1.
2.
3.

**Overall Assessment**: Pass / Fail / Partial
