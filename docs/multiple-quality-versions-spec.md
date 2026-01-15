# Multiple Quality Versions

## Feature Overview

SlipStream supports maintaining multiple versions of the same movie or episode simultaneously (e.g., one 4K HDR version for local streaming and one high-compatibility 1080p version for remote streaming). This addresses a key limitation of Sonarr and Radarr.

---

## 1. Core Concepts

### 1.1 Version Slots

The system supports up to 3 global **version slots**. Each slot represents a distinct quality tier that can be maintained for every movie/episode.

| Requirement | Description |
|-------------|-------------|
| 1.1.1 | Maximum of 3 version slots supported |
| 1.1.2 | Slots are globally defined and shared between Movies and TV Series |
| 1.1.3 | Each slot has a user-defined custom name/label (e.g., "Local 4K", "Remote Streaming") |
| 1.1.4 | Each slot has an explicit enable/disable toggle independent of profile assignment |
| 1.1.5 | Each slot is assigned exactly one quality profile |
| 1.1.6 | Each slot has its own independent monitored status (per movie/episode) |

### 1.2 Master Toggle

| Requirement | Description |
|-------------|-------------|
| 1.2.1 | Global master toggle to enable/disable multi-version functionality |
| 1.2.2 | When disabled, system behaves as single-version (Slot 1 only, legacy behavior) |
| 1.2.3 | Enabling requires completing the migration dry-run preview first |

---

## 2. Quality Profile Extensions

Quality profiles are extended to include additional AV attributes beyond resolution and source.

### 2.1 New Profile Attributes

| Requirement | Description |
|-------------|-------------|
| 2.1.1 | Add profile-level HDR settings (applies to whole profile, not per quality-item) |
| 2.1.2 | Add profile-level video codec settings |
| 2.1.3 | Add profile-level audio codec settings |
| 2.1.4 | Add profile-level audio channel settings |
| 2.1.5 | Each attribute supports both "preferred" (scoring bonus) and "required" (hard filter) modes |

### 2.2 Attribute Values

**HDR Formats** (all individually selectable):
- Dolby Vision (DV)
- HDR10+
- HDR10
- HDR (generic)
- HLG
- SDR (no HDR)

**Video Codecs** (all supported):
- x264/H.264/AVC
- x265/H.265/HEVC
- AV1
- VP9
- XviD
- DivX
- MPEG2

**Audio Codecs** (all supported):
- TrueHD
- DTS-HD MA
- DTS-HD
- DTS
- Dolby Digital Plus (DDP)
- Dolby Digital (DD)
- AAC
- FLAC
- LPCM/PCM
- Opus
- MP3

**Audio Channels**:
- 7.1
- 5.1
- 2.0
- 1.0

### 2.3 Combo Format Handling

| Requirement | Description |
|-------------|-------------|
| 2.3.1 | Releases with combo HDR formats (e.g., "DV HDR10") are parsed as multiple separate formats |
| 2.3.2 | A release matches a profile if ANY of its HDR layers satisfies the profile requirement |
| 2.3.3 | Example: "DV HDR10" release matches both "DV required" AND "HDR10 required" profiles |

### 2.4 Multi-Track Audio Handling

| Requirement | Description |
|-------------|-------------|
| 2.4.1 | Releases with multiple audio tracks (common in Remux) match if ANY track satisfies profile requirements |

### 2.5 Unknown Attribute Handling

| Requirement | Description |
|-------------|-------------|
| 2.5.1 | If parser cannot determine an attribute, that attribute fails "required" checks |
| 2.5.2 | Unknown attributes pass "preferred" checks (no bonus, no penalty) |

---

## 3. Mutual Exclusivity

Profiles assigned to different slots must be mutually exclusive to prevent ambiguous matching.

### 3.1 Exclusivity Rules

| Requirement | Description |
|-------------|-------------|
| 3.1.1 | Two profiles are mutually exclusive if their **required** attributes conflict |
| 3.1.2 | Conflict means: Profile A requires attribute X, Profile B disallows attribute X |
| 3.1.3 | Preferred attributes do not affect exclusivity calculation |
| 3.1.4 | System prevents saving slot configuration if assigned profiles overlap |

### 3.2 Examples

**Mutually Exclusive:**
- Profile A: HDR required, 4K required
- Profile B: SDR required (HDR disallowed), 4K required
- *Conflict: HDR status*

**NOT Mutually Exclusive (blocked):**
- Profile A: 4K preferred, any HDR acceptable
- Profile B: 4K preferred, any HDR acceptable
- *No required attribute conflicts*

---

## 4. File Naming Validation

### 4.1 Differentiator Requirements

| Requirement | Description |
|-------------|-------------|
| 4.1.1 | Filename format must include tokens for attributes that differ between assigned slot profiles |
| 4.1.2 | Only **conflicting** differentiators are required (attributes where profiles have opposing requirements) |
| 4.1.3 | Example: If slots differ only on HDR status, HDR token must be in filename format |
| 4.1.4 | Validation occurs when saving slot configuration |
| 4.1.5 | If filename format is missing required differentiators, prompt user to update format before saving |

---

## 5. Slot Assignment Logic

### 5.1 Automatic Assignment (Auto-Search & Scan)

| Requirement | Description |
|-------------|-------------|
| 5.1.1 | Evaluate release against all enabled slot profiles |
| 5.1.2 | Calculate match score for each slot's profile |
| 5.1.3 | Assign to the slot with the **best quality match** (closest to profile's target) |
| 5.1.4 | If scores are equal, assign to whichever slot is empty (or first slot if both empty) |
| 5.1.5 | Mutual exclusivity should prevent true ties in normal operation |

### 5.2 Manual Import

| Requirement | Description |
|-------------|-------------|
| 5.2.1 | When manually importing, if file could fit multiple slots, prompt user to choose |
| 5.2.2 | Show recommendation based on best match score |
| 5.2.3 | Allow user to override auto-detected target slot |

### 5.3 Import Edge Cases - File Below All Profile Requirements

| Requirement | Description |
|-------------|-------------|
| 5.3.1 | **All slots empty**: Accept to closest-matching slot (fallback to Slot 1); slot shows as "upgrade needed" |
| 5.3.2 | **Some slots filled, some empty**: Reject import |
| 5.3.3 | **All slots filled**: Reject import |

---

## 6. Status Determination

### 6.1 Missing Status

| Requirement | Description |
|-------------|-------------|
| 6.1.1 | A movie/episode is "missing" if ANY **monitored** slot is empty |
| 6.1.2 | Unmonitored empty slots do not affect missing status |

### 6.2 Upgrade Status

| Requirement | Description |
|-------------|-------------|
| 6.2.1 | Each slot independently tracks upgrade eligibility based on its profile's cutoff |
| 6.2.2 | Slot is "upgrade needed" if file exists but quality is below profile cutoff |

---

## 7. Auto-Search Behavior

### 7.1 Search Execution

| Requirement | Description |
|-------------|-------------|
| 7.1.1 | When auto-search runs for a movie with multiple empty monitored slots, search in **parallel** |
| 7.1.2 | May grab multiple releases simultaneously for different slots |
| 7.1.3 | Each slot's search is independent |

### 7.2 Upgrade Handling

| Requirement | Description |
|-------------|-------------|
| 7.2.1 | When a slot has a file but finds a better match (upgrade), **replace and delete** the old file |
| 7.2.2 | Standard upgrade behavior per slot; no cross-slot file movement |

---

## 8. Monitoring

### 8.1 Per-Slot Monitoring

| Requirement | Description |
|-------------|-------------|
| 8.1.1 | Each slot has its own monitored toggle per movie/episode |
| 8.1.2 | A slot can be monitored independently (e.g., monitor 4K slot but not 1080p slot) |
| 8.1.3 | Auto-search only runs for monitored slots |

---

## 9. File Organization

### 9.1 Directory Structure

| Requirement | Description |
|-------------|-------------|
| 9.1.1 | Multiple versions of same movie/episode stored in same directory (existing structure) |
| 9.1.2 | Files differentiated by quality suffix in filename (existing naming) |
| 9.1.3 | No slot identifiers or subdirectories added to paths |

---

## 10. Download Queue Integration

### 10.1 Queue Display

| Requirement | Description |
|-------------|-------------|
| 10.1.1 | Queue shows raw downloads from client with mapped media (movie or episode/season) |
| 10.1.2 | Target slot info shown inline with each queue item |

### 10.2 Failed Downloads

| Requirement | Description |
|-------------|-------------|
| 10.2.1 | If download fails or is rejected, slot reverts to "empty" status immediately |
| 10.2.2 | No pending/retry state; waits for next search or manual action |

---

## 11. Search Results Display

| Requirement | Description |
|-------------|-------------|
| 11.1.1 | Search results indicate which slot each release would fill |
| 11.1.2 | Show whether grab would be an upgrade vs new fill |
| 11.1.3 | Allow user to override auto-detected slot when grabbing |

---

## 12. Deletion Behavior

### 12.1 File Deletion

| Requirement | Description |
|-------------|-------------|
| 12.1.1 | Deleting a file from a slot does NOT trigger automatic search for replacement |
| 12.1.2 | Slot becomes empty; waits for next scheduled search |

### 12.2 Slot Disabled

| Requirement | Description |
|-------------|-------------|
| 12.2.1 | When user disables a slot that has files assigned, prompt for action |
| 12.2.2 | Options: delete files, keep unassigned, or cancel |

---

## 13. Library Scanning

### 13.1 Scan Behavior

| Requirement | Description |
|-------------|-------------|
| 13.1.1 | Scanner discovers all files in movie/episode directories |
| 13.1.2 | Auto-assign each file to best-matching slot |
| 13.1.3 | Extra files (more than slot count) queued for user review |

---

## 14. Migration (Enabling Multi-Version)

### 14.1 Dry Run Preview

| Requirement | Description |
|-------------|-------------|
| 14.1.1 | Dry run preview is **required** before enabling multi-version |
| 14.1.2 | Preview organized by type (Movies, TV Shows), then per-item |
| 14.1.3 | TV shows show per-series, per-season breakdown with collapsible headers |
| 14.1.4 | Show proposed slot assignment for each file |
| 14.1.5 | Show conflicts and files that can't be matched |

### 14.2 Assignment Logic

| Requirement | Description |
|-------------|-------------|
| 14.2.1 | Intelligently assign existing files to slots based on quality profile matching |
| 14.2.2 | Files that can't be matched to any slot go to review queue |
| 14.2.3 | Quality profile must be assigned to slot before saving configuration |

### 14.3 Review Queue

| Requirement | Description |
|-------------|-------------|
| 14.3.1 | Dedicated full review page for files needing manual disposition |
| 14.3.2 | Show file details, detected quality, and available slot options |
| 14.3.3 | Allow assignment to specific slot or deletion |
| 14.3.4 | Movies with more files than enabled slots: extras go to review queue |

---

## 15. Profile Configuration Changes

### 15.1 Changing Slot's Profile

| Requirement | Description |
|-------------|-------------|
| 15.1.1 | When user changes a slot's quality profile after files are assigned, prompt for action |
| 15.1.2 | Options: keep current assignments, re-evaluate (may queue non-matches), or cancel |

---

## 16. TV-Specific Behavior

### 16.1 Episode Independence

| Requirement | Description |
|-------------|-------------|
| 16.1.1 | Each episode independently tracks which slots are filled |
| 16.1.2 | Different episodes may have different slot fills (e.g., S01 has 4K, S02 only 1080p) |

### 16.2 Season Packs

| Requirement | Description |
|-------------|-------------|
| 16.2.1 | Season pack assigned to the slot it best matches |
| 16.2.2 | May result in mixed slots across seasons (acceptable) |
| 16.2.3 | Each episode from pack individually assessed |

---

## 17. History & Activity Logging

| Requirement | Description |
|-------------|-------------|
| 17.1.1 | Log all slot-related events: assignments, reassignments, deletions |
| 17.1.2 | History entries include slot information |

---

## 18. API

### 18.1 Slot Management

| Requirement | Description |
|-------------|-------------|
| 18.1.1 | Full CRUD endpoints for slot assignments on files |
| 18.1.2 | Endpoints to view, assign, reassign, and unassign slots |

### 18.2 Grab Requests

| Requirement | Description |
|-------------|-------------|
| 18.2.1 | API grab requests accept optional `target_slot` parameter |
| 18.2.2 | If omitted, auto-detect best slot |

---

## 19. UI Location

| Requirement | Description |
|-------------|-------------|
| 19.1.1 | Version slot configuration located in Media Management settings page |
| 19.1.2 | Movie/episode detail page shows files in table/list format with slot assignments |

---

## 20. Testing Requirements

Comprehensive testing is critical due to the many permutations involved.

### 20.1 Backend Test Matrix

The following dimensions should be crossed for full coverage:

**Slot States** (per slot):
- Empty
- Filled with file meeting profile requirements
- Filled with file below profile requirements

**Operations**:
- Import (manual)
- Import (automatic/scan)
- Auto-search grab
- Manual grab
- Upgrade
- Delete
- Reassign

**Profile Configurations**:
- Single slot enabled
- Two slots, mutually exclusive profiles
- Three slots, various exclusivity patterns
- Profile with required vs preferred attributes

**File Types**:
- Standard quality (meets one profile clearly)
- Ambiguous quality (could match multiple profiles)
- Below all profiles
- Unknown attributes

**Scenarios to Test**:
| ID | Scenario | Expected Behavior |
|----|----------|-------------------|
| T1 | Import file matching Slot 1, all slots empty | Assign to Slot 1 |
| T2 | Import file matching Slot 2, all slots empty | Assign to Slot 2 |
| T3 | Import file matching both slots equally, all empty | Assign to first empty |
| T4 | Import file matching Slot 1, Slot 1 filled, Slot 2 empty | Prompt user (upgrade vs assign to Slot 2) |
| T5 | Import file below all profiles, all empty | Accept to closest slot |
| T6 | Import file below all profiles, some filled | Reject |
| T7 | Upgrade within slot (better file for same profile) | Replace, delete old |
| T8 | Auto-search with 2 empty monitored slots | Parallel search, grab for each |
| T9 | Auto-search with 1 monitored, 1 unmonitored empty | Only search for monitored |
| T10 | Disable slot with files | Prompt for action |
| T11 | Change slot profile with files assigned | Prompt for action |
| T12 | Scan folder with more files than slots | Assign best matches, queue extras |
| T13 | DV+HDR10 file, Slot 1 requires DV, Slot 2 requires HDR10 | Assign to Slot 1 (DV higher priority) |
| T14 | Multi-audio Remux, profile requires TrueHD | Match if any track is TrueHD |
| T15 | Unknown codec, profile requires x265 | Fail match |
| T16 | TV season pack, episodes exist in different slot | Import to matching slot per episode |
| T17 | Missing status with 1 monitored slot empty, 1 filled | Show as missing |
| T18 | Missing status with 1 unmonitored slot empty | Show as available |

### 20.2 Frontend Debug Features (Developer Mode)

| Requirement | Description |
|-------------|-------------|
| 20.2.1 | **Slot State Viewer**: Panel showing raw slot assignments and matching scores for any movie/episode |
| 20.2.2 | **Mock File Import**: Simulate file imports with custom quality attributes to test matching |
| 20.2.3 | **Profile Matching Tester**: Input release attributes, see which slots would match and scores |
| 20.2.4 | **Migration Simulator**: Preview migration results without actually enabling multi-version |
| 20.2.5 | All debug features gated behind `developerMode` |

---

## 21. Requirement Groups (Dependency Order)

For implementation planning, requirements are grouped by dependency:

### Group A: Quality Profile Extensions (Foundation)
- 2.1.x (new attributes)
- 2.2 (attribute values)
- 2.3.x (combo handling)
- 2.4.x (multi-track audio)
- 2.5.x (unknown handling)

### Group B: Mutual Exclusivity (Depends on A)
- 3.1.x (exclusivity rules)

### Group C: Slot Infrastructure (Depends on B)
- 1.1.x (slot definition)
- 1.2.x (master toggle)
- 4.1.x (filename validation)

### Group D: Assignment Logic (Depends on C)
- 5.1.x (auto assignment)
- 5.2.x (manual import)
- 5.3.x (edge cases)

### Group E: Status & Monitoring (Depends on D)
- 6.1.x (missing status)
- 6.2.x (upgrade status)
- 8.1.x (per-slot monitoring)

### Group F: Search Integration (Depends on E)
- 7.1.x (search execution)
- 7.2.x (upgrade handling)
- 11.1.x (search results display)

### Group G: File Operations (Depends on D)
- 9.1.x (directory structure)
- 12.1.x (file deletion)
- 12.2.x (slot disabled)
- 13.1.x (library scanning)

### Group H: Queue & History (Depends on F, G)
- 10.1.x (queue display)
- 10.2.x (failed downloads)
- 17.1.x (history logging)

### Group I: Migration (Depends on all above)
- 14.1.x (dry run preview)
- 14.2.x (assignment logic)
- 14.3.x (review queue)
- 15.1.x (profile changes)

### Group J: TV-Specific (Depends on D, E)
- 16.1.x (episode independence)
- 16.2.x (season packs)

### Group K: API (Depends on D)
- 18.1.x (slot management)
- 18.2.x (grab requests)

### Group L: UI (Depends on all above)
- 19.1.x (settings location)

### Group M: Testing (Parallel with implementation)
- 20.1 (backend test matrix)
- 20.2.x (debug features)

---

## 22. Slot Root Folders

Slots can optionally have dedicated root folders, allowing different quality versions to be stored on different storage devices.

### 22.1 Root Folder Assignment

| Requirement | Description |
|-------------|-------------|
| 22.1.1 | Each slot can have a dedicated root folder for movies |
| 22.1.2 | Each slot can have a dedicated root folder for TV series |
| 22.1.3 | Root folder assignment is optional per slot |
| 22.1.4 | If slot has no root folder, falls back to media item's root folder |

### 22.2 Import Behavior

| Requirement | Description |
|-------------|-------------|
| 22.2.1 | In multi-version mode, check target slot's root folder first |
| 22.2.2 | If slot root folder is set, use it for file destination |
| 22.2.3 | If slot root folder is not set, use media item's root folder |

### 22.3 UI Integration

| Requirement | Description |
|-------------|-------------|
| 22.3.1 | Root folder selectors shown in slot configuration when multi-version enabled |
| 22.3.2 | Movie and TV root folders selectable independently per slot |
| 22.3.3 | Default option shows "Use media default" behavior |

---

## Requirement Groups Update

### Group C: Slot Infrastructure (Updated)
- 1.1.x (slot definition)
- 1.2.x (master toggle)
- 4.1.x (filename validation)
- **22.1.x (slot root folders)**

### Group D: Assignment Logic (Updated)
- 5.1.x (auto assignment)
- 5.2.x (manual import)
- 5.3.x (edge cases)
- **22.2.x (slot root folder import behavior)**
