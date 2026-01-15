# Multiple Quality Versions - Implementation Plan

This plan implements the Multiple Quality Versions feature as specified in `docs/multiple-quality-versions.md`.

---

## Implementation Status

| Phase | Name | Status |
|-------|------|--------|
| 1 | Quality Profile Extensions | ✅ Complete |
| 2 | Mutual Exclusivity | ✅ Complete |
| 3 | Slot Infrastructure | ✅ Complete |
| 4 | Assignment Logic | ✅ Complete |
| 5 | Status & Monitoring | ✅ Complete |
| 6 | Search Integration | ✅ Complete |
| 7 | File Operations | ✅ Complete |
| 8 | Queue & History | ✅ Complete |
| 9 | Migration System | ✅ Complete |
| 10 | TV-Specific | ✅ Complete |
| 11 | API Endpoints | ✅ Complete |
| 12 | UI Components | ✅ Complete |
| 13 | Debug & Testing | ✅ Complete |

---

## Requirement Traceability Matrix

This table maps every specification requirement to its implementation phase.

| Req ID | Description | Phase |
|--------|-------------|-------|
| **1. Core Concepts** |||
| 1.1.1 | Maximum of 3 version slots supported | Phase 3 |
| 1.1.2 | Slots are globally defined and shared between Movies and TV Series | Phase 3 |
| 1.1.3 | Each slot has a user-defined custom name/label | Phase 3 |
| 1.1.4 | Each slot has an explicit enable/disable toggle independent of profile assignment | Phase 3 |
| 1.1.5 | Each slot is assigned exactly one quality profile | Phase 3 |
| 1.1.6 | Each slot has its own independent monitored status (per movie/episode) | Phase 5 |
| 1.2.1 | Global master toggle to enable/disable multi-version functionality | Phase 3 |
| 1.2.2 | When disabled, system behaves as single-version (Slot 1 only, legacy behavior) | Phase 3 |
| 1.2.3 | Enabling requires completing the migration dry-run preview first | Phase 9 |
| **2. Quality Profile Extensions** |||
| 2.1.1 | Add profile-level HDR settings | Phase 1 |
| 2.1.2 | Add profile-level video codec settings | Phase 1 |
| 2.1.3 | Add profile-level audio codec settings | Phase 1 |
| 2.1.4 | Add profile-level audio channel settings | Phase 1 |
| 2.1.5 | Each attribute supports both "preferred" and "required" modes | Phase 1 |
| 2.2 | Attribute values (HDR, Video Codecs, Audio Codecs, Audio Channels) | Phase 1 |
| 2.3.1 | Releases with combo HDR formats parsed as multiple separate formats | Phase 1 |
| 2.3.2 | A release matches a profile if ANY of its HDR layers satisfies requirement | Phase 1 |
| 2.3.3 | Example: "DV HDR10" release matches both "DV required" AND "HDR10 required" | Phase 1 |
| 2.4.1 | Releases with multiple audio tracks match if ANY track satisfies requirements | Phase 1 |
| 2.5.1 | If parser cannot determine attribute, that attribute fails "required" checks | Phase 1 |
| 2.5.2 | Unknown attributes pass "preferred" checks (no bonus, no penalty) | Phase 1 |
| **3. Mutual Exclusivity** |||
| 3.1.1 | Two profiles are mutually exclusive if their required attributes conflict | Phase 2 |
| 3.1.2 | Conflict means: Profile A requires attribute X, Profile B disallows attribute X | Phase 2 |
| 3.1.3 | Preferred attributes do not affect exclusivity calculation | Phase 2 |
| 3.1.4 | System prevents saving slot configuration if assigned profiles overlap | Phase 2 |
| **4. File Naming Validation** |||
| 4.1.1 | Filename format must include tokens for attributes that differ between slot profiles | Phase 3 |
| 4.1.2 | Only conflicting differentiators are required | Phase 3 |
| 4.1.3 | Example: If slots differ only on HDR status, HDR token must be in filename | Phase 3 |
| 4.1.4 | Validation occurs when saving slot configuration | Phase 3 |
| 4.1.5 | If filename format missing required differentiators, prompt user to update | Phase 3 |
| **5. Slot Assignment Logic** |||
| 5.1.1 | Evaluate release against all enabled slot profiles | Phase 4 |
| 5.1.2 | Calculate match score for each slot's profile | Phase 4 |
| 5.1.3 | Assign to the slot with the best quality match | Phase 4 |
| 5.1.4 | If scores equal, assign to whichever slot is empty (or first if both empty) | Phase 4 |
| 5.1.5 | Mutual exclusivity should prevent true ties in normal operation | Phase 4 |
| 5.2.1 | When manually importing, if file could fit multiple slots, prompt user | Phase 4 |
| 5.2.2 | Show recommendation based on best match score | Phase 4 |
| 5.2.3 | Allow user to override auto-detected target slot | Phase 4 |
| 5.3.1 | All slots empty: Accept to closest-matching slot; show "upgrade needed" | Phase 4 |
| 5.3.2 | Some slots filled, some empty: Reject import | Phase 4 |
| 5.3.3 | All slots filled: Reject import | Phase 4 |
| **6. Status Determination** |||
| 6.1.1 | Movie/episode is "missing" if ANY monitored slot is empty | Phase 5 |
| 6.1.2 | Unmonitored empty slots do not affect missing status | Phase 5 |
| 6.2.1 | Each slot independently tracks upgrade eligibility based on profile cutoff | Phase 5 |
| 6.2.2 | Slot is "upgrade needed" if file exists but quality below profile cutoff | Phase 5 |
| **7. Auto-Search Behavior** |||
| 7.1.1 | Auto-search with multiple empty monitored slots: search in parallel | Phase 6 |
| 7.1.2 | May grab multiple releases simultaneously for different slots | Phase 6 |
| 7.1.3 | Each slot's search is independent | Phase 6 |
| 7.2.1 | When slot has file but finds better match (upgrade), replace and delete old | Phase 6 |
| 7.2.2 | Standard upgrade behavior per slot; no cross-slot file movement | Phase 6 |
| **8. Monitoring** |||
| 8.1.1 | Each slot has its own monitored toggle per movie/episode | Phase 5 |
| 8.1.2 | A slot can be monitored independently | Phase 5 |
| 8.1.3 | Auto-search only runs for monitored slots | Phase 6 |
| **9. File Organization** |||
| 9.1.1 | Multiple versions stored in same directory (existing structure) | Phase 7 |
| 9.1.2 | Files differentiated by quality suffix in filename (existing naming) | Phase 7 |
| 9.1.3 | No slot identifiers or subdirectories added to paths | Phase 7 |
| **10. Download Queue Integration** |||
| 10.1.1 | Queue shows raw downloads from client with mapped media | Phase 8 |
| 10.1.2 | Target slot info shown inline with each queue item | Phase 8 |
| 10.2.1 | If download fails or rejected, slot reverts to "empty" immediately | Phase 8 |
| 10.2.2 | No pending/retry state; waits for next search or manual action | Phase 8 |
| **11. Search Results Display** |||
| 11.1.1 | Search results indicate which slot each release would fill | Phase 6 |
| 11.1.2 | Show whether grab would be upgrade vs new fill | Phase 6 |
| 11.1.3 | Allow user to override auto-detected slot when grabbing | Phase 6 |
| **12. Deletion Behavior** |||
| 12.1.1 | Deleting file from slot does NOT trigger automatic search | Phase 7 |
| 12.1.2 | Slot becomes empty; waits for next scheduled search | Phase 7 |
| 12.2.1 | When user disables slot with files assigned, prompt for action | Phase 7 |
| 12.2.2 | Options: delete files, keep unassigned, or cancel | Phase 7 |
| **13. Library Scanning** |||
| 13.1.1 | Scanner discovers all files in movie/episode directories | Phase 7 |
| 13.1.2 | Auto-assign each file to best-matching slot | Phase 7 |
| 13.1.3 | Extra files (more than slot count) queued for user review | Phase 7 |
| **14. Migration** |||
| 14.1.1 | Dry run preview is required before enabling multi-version | Phase 9 |
| 14.1.2 | Preview organized by type (Movies, TV Shows), then per-item | Phase 9 |
| 14.1.3 | TV shows show per-series, per-season breakdown with collapsible headers | Phase 9 |
| 14.1.4 | Show proposed slot assignment for each file | Phase 9 |
| 14.1.5 | Show conflicts and files that can't be matched | Phase 9 |
| 14.2.1 | Intelligently assign existing files to slots based on quality profile matching | Phase 9 |
| 14.2.2 | Files that can't be matched to any slot go to review queue | Phase 9 |
| 14.2.3 | Quality profile must be assigned to slot before saving configuration | Phase 9 |
| 14.3.1 | Dedicated full review page for files needing manual disposition | Phase 9 |
| 14.3.2 | Show file details, detected quality, and available slot options | Phase 9 |
| 14.3.3 | Allow assignment to specific slot or deletion | Phase 9 |
| 14.3.4 | Movies with more files than enabled slots: extras go to review queue | Phase 9 |
| **15. Profile Configuration Changes** |||
| 15.1.1 | When user changes slot's profile after files assigned, prompt for action | Phase 9 |
| 15.1.2 | Options: keep current assignments, re-evaluate, or cancel | Phase 9 |
| **16. TV-Specific Behavior** |||
| 16.1.1 | Each episode independently tracks which slots are filled | Phase 10 |
| 16.1.2 | Different episodes may have different slot fills | Phase 10 |
| 16.2.1 | Season pack assigned to the slot it best matches | Phase 10 |
| 16.2.2 | May result in mixed slots across seasons (acceptable) | Phase 10 |
| 16.2.3 | Each episode from pack individually assessed | Phase 10 |
| **17. History & Activity Logging** |||
| 17.1.1 | Log all slot-related events: assignments, reassignments, deletions | Phase 8 |
| 17.1.2 | History entries include slot information | Phase 8 |
| **18. API** |||
| 18.1.1 | Full CRUD endpoints for slot assignments on files | Phase 11 |
| 18.1.2 | Endpoints to view, assign, reassign, and unassign slots | Phase 11 |
| 18.2.1 | API grab requests accept optional `target_slot` parameter | Phase 11 |
| 18.2.2 | If omitted, auto-detect best slot | Phase 11 |
| **19. UI Location** |||
| 19.1.1 | Version slot configuration in Media Management settings page | Phase 12 |
| 19.1.2 | Movie/episode detail page shows files with slot assignments | Phase 12 |
| **20. Testing Requirements** |||
| 20.1 | Backend Test Matrix (T1-T18 scenarios) | Phase 13 |
| 20.2.1 | Slot State Viewer: Panel showing raw slot assignments and matching scores | Phase 13 |
| 20.2.2 | Mock File Import: Simulate file imports with custom quality attributes | Phase 13 |
| 20.2.3 | Profile Matching Tester: Input release attributes, see which slots match | Phase 13 |
| 20.2.4 | Migration Simulator: Preview migration without enabling | Phase 13 |
| 20.2.5 | All debug features gated behind `developerMode` | Phase 13 |

---

## Phase Overview

| Phase | Name | Spec Groups | Requirements |
|-------|------|-------------|--------------|
| 1 | Quality Profile Extensions | Group A | 2.1.1-2.1.5, 2.2, 2.3.1-2.3.3, 2.4.1, 2.5.1-2.5.2 |
| 2 | Mutual Exclusivity | Group B | 3.1.1-3.1.4 |
| 3 | Slot Infrastructure | Group C | 1.1.1-1.1.5, 1.2.1-1.2.2, 4.1.1-4.1.5 |
| 4 | Assignment Logic | Group D | 5.1.1-5.1.5, 5.2.1-5.2.3, 5.3.1-5.3.3 |
| 5 | Status & Monitoring | Group E | 1.1.6, 6.1.1-6.1.2, 6.2.1-6.2.2, 8.1.1-8.1.2 |
| 6 | Search Integration | Group F | 7.1.1-7.1.3, 7.2.1-7.2.2, 8.1.3, 11.1.1-11.1.3 |
| 7 | File Operations | Group G | 9.1.1-9.1.3, 12.1.1-12.1.2, 12.2.1-12.2.2, 13.1.1-13.1.3 |
| 8 | Queue & History | Group H | 10.1.1-10.1.2, 10.2.1-10.2.2, 17.1.1-17.1.2 |
| 9 | Migration System | Group I | 1.2.3, 14.1.1-14.1.5, 14.2.1-14.2.3, 14.3.1-14.3.4, 15.1.1-15.1.2 |
| 10 | TV-Specific | Group J | 16.1.1-16.1.2, 16.2.1-16.2.3 |
| 11 | API Endpoints | Group K | 18.1.1-18.1.2, 18.2.1-18.2.2 |
| 12 | UI Components | Group L | 19.1.1-19.1.2 |
| 13 | Debug & Testing | Group M | 20.1, 20.2.1-20.2.5 |

---

## Phase 1: Quality Profile Extensions (Foundation)

**Spec Group:** A (Quality Profile Extensions)
**Requirements:** 2.1.1-2.1.5, 2.2, 2.3.1-2.3.3, 2.4.1, 2.5.1-2.5.2

### 1.1 Database Changes

**Migration: `XXX_quality_profile_attributes.sql`**

```sql
-- Req 2.1.1-2.1.4: Add profile-level attribute settings
ALTER TABLE quality_profiles ADD COLUMN hdr_settings TEXT DEFAULT '{}';
ALTER TABLE quality_profiles ADD COLUMN video_codec_settings TEXT DEFAULT '{}';
ALTER TABLE quality_profiles ADD COLUMN audio_codec_settings TEXT DEFAULT '{}';
ALTER TABLE quality_profiles ADD COLUMN audio_channel_settings TEXT DEFAULT '{}';
```

**JSON Schema for each attribute (Req 2.1.5):**
```json
{
  "mode": "any" | "preferred" | "required",
  "values": ["DV", "HDR10+", "HDR10", "HDR", "HLG", "SDR"]
}
```

### 1.2 Backend Types

**File: `internal/library/quality/attributes.go`** (new file)

```go
// Req 2.1.5: Attribute modes
type AttributeMode string
const (
    AttributeModeAny       AttributeMode = "any"       // No filtering
    AttributeModePreferred AttributeMode = "preferred" // Scoring bonus
    AttributeModeRequired  AttributeMode = "required"  // Hard filter
)

type AttributeSettings struct {
    Mode   AttributeMode `json:"mode"`
    Values []string      `json:"values"`
}

// Req 2.2: All supported attribute values
var (
    HDRFormats = []string{"DV", "HDR10+", "HDR10", "HDR", "HLG", "SDR"}
    VideoCodecs = []string{"x264", "x265", "AV1", "VP9", "XviD", "DivX", "MPEG2"}
    AudioCodecs = []string{"TrueHD", "DTS-HD MA", "DTS-HD", "DTS", "DDP", "DD", "AAC", "FLAC", "LPCM", "Opus", "MP3"}
    AudioChannels = []string{"7.1", "5.1", "2.0", "1.0"}
)
```

**File: `internal/library/quality/profile.go`** (extend Profile struct)

```go
// Req 2.1.1-2.1.4: Profile-level attribute settings
type Profile struct {
    // ... existing fields
    HDRSettings          AttributeSettings `json:"hdrSettings"`
    VideoCodecSettings   AttributeSettings `json:"videoCodecSettings"`
    AudioCodecSettings   AttributeSettings `json:"audioCodecSettings"`
    AudioChannelSettings AttributeSettings `json:"audioChannelSettings"`
}
```

### 1.3 Attribute Matching Logic

**File: `internal/library/quality/matcher.go`** (new file)

```go
// Req 2.3.1-2.3.3: Combo format handling
// ParseHDRFormats splits "DV HDR10" into ["DV", "HDR10"]
func ParseHDRFormats(input string) []string

// Req 2.3.2: Match if ANY HDR layer satisfies requirement
func MatchHDRRequirement(releaseFormats []string, settings AttributeSettings) (matches bool, score float64)

// Req 2.4.1: Multi-track audio handling
// Match if ANY track satisfies profile requirements
func MatchAudioRequirement(releaseTracks []string, settings AttributeSettings) (matches bool, score float64)

// Req 2.5.1: Unknown attributes fail "required" checks
// Req 2.5.2: Unknown attributes pass "preferred" checks (no bonus, no penalty)
func MatchAttribute(releaseValue string, settings AttributeSettings) (matches bool, score float64) {
    if releaseValue == "" || releaseValue == "unknown" {
        if settings.Mode == AttributeModeRequired {
            return false, 0 // Req 2.5.1
        }
        return true, 0 // Req 2.5.2: pass but no bonus
    }
    // ... normal matching logic
}
```

### 1.4 Frontend Changes

**File: `web/src/types/qualityProfile.ts`**

```typescript
// Req 2.1.5: Attribute modes
type AttributeMode = 'any' | 'preferred' | 'required';

interface AttributeSettings {
  mode: AttributeMode;
  values: string[];
}

// Req 2.1.1-2.1.4: Extended profile
interface QualityProfile {
  // ... existing fields
  hdrSettings: AttributeSettings;
  videoCodecSettings: AttributeSettings;
  audioCodecSettings: AttributeSettings;
  audioChannelSettings: AttributeSettings;
}
```

**File: `web/src/routes/settings/profiles.tsx`**
- Add attribute configuration sections for each attribute type (Req 2.1.1-2.1.4)
- Mode selector dropdown (Any/Preferred/Required) (Req 2.1.5)
- Multi-select checkboxes for attribute values (Req 2.2)

### 1.5 Testing

| Test | Requirements |
|------|--------------|
| Combo HDR parsing ("DV HDR10" → [DV, HDR10]) | 2.3.1 |
| DV+HDR10 matches both DV-required and HDR10-required profiles | 2.3.2, 2.3.3 |
| Multi-track audio matching (any track satisfies) | 2.4.1 |
| Unknown attribute fails required check | 2.5.1 |
| Unknown attribute passes preferred check (no bonus) | 2.5.2 |

---

## Phase 2: Mutual Exclusivity

**Spec Group:** B (Mutual Exclusivity)
**Requirements:** 3.1.1-3.1.4

### 2.1 Backend Implementation

**File: `internal/library/quality/exclusivity.go`** (new file)

```go
// Req 3.1.1: Check if profiles are mutually exclusive
func CheckMutualExclusivity(profileA, profileB *Profile) bool {
    // Profiles are exclusive if required attributes conflict
    return hasConflictingRequiredAttributes(profileA, profileB)
}

// Req 3.1.2: Detect conflicts (A requires X, B disallows X)
func hasConflictingRequiredAttributes(a, b *Profile) bool {
    // Check HDR: if A requires DV and B requires SDR, they conflict
    if a.HDRSettings.Mode == AttributeModeRequired &&
       b.HDRSettings.Mode == AttributeModeRequired {
        if !hasOverlap(a.HDRSettings.Values, b.HDRSettings.Values) {
            return true // Conflict found
        }
    }
    // ... repeat for video codec, audio codec, audio channels
    return false
}

// Req 3.1.3: Preferred attributes do not affect exclusivity
// (Only check Required mode in hasConflictingRequiredAttributes)

// Req 3.1.4: Validation function for slot configuration
func ValidateSlotExclusivity(slots []SlotConfig) error {
    for i, slotA := range slots {
        for j, slotB := range slots {
            if i >= j || !slotA.Enabled || !slotB.Enabled {
                continue
            }
            if !CheckMutualExclusivity(slotA.Profile, slotB.Profile) {
                return fmt.Errorf("profiles %s and %s are not mutually exclusive",
                    slotA.Profile.Name, slotB.Profile.Name)
            }
        }
    }
    return nil
}
```

### 2.2 Service Integration

**File: `internal/library/slots/service.go`**
- Call `ValidateSlotExclusivity()` before saving slot configuration (Req 3.1.4)
- Return validation error if profiles overlap

### 2.3 Frontend Validation

**File: `web/src/routes/settings/slots.tsx`**
- Show warning when selecting non-exclusive profiles
- Disable save button if exclusivity validation fails (Req 3.1.4)

### 2.4 Testing

| Test | Requirements |
|------|--------------|
| HDR-required vs SDR-required = exclusive | 3.1.1, 3.1.2 |
| HDR-preferred vs SDR-preferred = NOT exclusive | 3.1.3 |
| Block save when profiles overlap | 3.1.4 |

---

## Phase 3: Slot Infrastructure

**Spec Group:** C (Slot Infrastructure)
**Requirements:** 1.1.1-1.1.5, 1.2.1-1.2.2, 4.1.1-4.1.5

### 3.1 Database Changes

**Migration: `XXX_version_slots.sql`**

```sql
-- Req 1.1.1: Maximum of 3 version slots
-- Req 1.1.2: Globally defined and shared between Movies and TV
CREATE TABLE version_slots (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    slot_number INTEGER NOT NULL CHECK (slot_number >= 1 AND slot_number <= 3),
    -- Req 1.1.3: User-defined custom name/label
    name TEXT NOT NULL,
    -- Req 1.1.4: Enable/disable toggle independent of profile
    enabled INTEGER NOT NULL DEFAULT 0,
    -- Req 1.1.5: Each slot assigned exactly one quality profile
    quality_profile_id INTEGER REFERENCES quality_profiles(id),
    display_order INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(slot_number)
);

-- Req 1.2.1: Global master toggle
CREATE TABLE multi_version_settings (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    enabled INTEGER NOT NULL DEFAULT 0,
    dry_run_completed INTEGER NOT NULL DEFAULT 0,
    last_migration_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Insert default row (disabled)
INSERT INTO multi_version_settings (id, enabled, dry_run_completed) VALUES (1, 0, 0);

-- Insert 3 default slots (Req 1.1.1)
INSERT INTO version_slots (slot_number, name, enabled, display_order) VALUES
    (1, 'Primary', 0, 1),
    (2, 'Secondary', 0, 2),
    (3, 'Tertiary', 0, 3);

-- Add slot_id to file tables
ALTER TABLE movie_files ADD COLUMN slot_id INTEGER REFERENCES version_slots(id);
ALTER TABLE episode_files ADD COLUMN slot_id INTEGER REFERENCES version_slots(id);
```

### 3.2 Backend Implementation

**File: `internal/library/slots/` (new package)**

```
slots/
├── slot.go           # Slot type definitions
├── service.go        # Slot CRUD operations
├── handlers.go       # HTTP handlers
├── validation.go     # Slot configuration validation
└── naming.go         # Filename differentiator validation
```

**Types (Req 1.1.1-1.1.5):**
```go
type Slot struct {
    ID               int64            // Req 1.1.1: ID 1-3
    SlotNumber       int              // Req 1.1.1: 1, 2, or 3
    Name             string           // Req 1.1.3: Custom name
    Enabled          bool             // Req 1.1.4: Enable/disable toggle
    QualityProfileID *int64           // Req 1.1.5: Assigned profile
    QualityProfile   *quality.Profile // Resolved profile
    DisplayOrder     int
}

type MultiVersionSettings struct {
    Enabled         bool       // Req 1.2.1: Master toggle
    DryRunCompleted bool       // For Req 1.2.3 (enforced in Phase 9)
    LastMigrationAt *time.Time
}
```

**Req 1.2.2: Legacy behavior when disabled:**
```go
func (s *Service) IsMultiVersionEnabled(ctx context.Context) bool {
    settings, _ := s.GetSettings(ctx)
    return settings.Enabled
}

// When disabled, all slot-aware functions treat system as single-slot (Slot 1 only)
func (s *Service) GetEffectiveSlots(ctx context.Context) ([]Slot, error) {
    if !s.IsMultiVersionEnabled(ctx) {
        // Return only Slot 1 for legacy behavior
        return s.getSlot1Only(ctx)
    }
    return s.GetEnabledSlots(ctx)
}
```

### 3.3 File Naming Validation

**File: `internal/library/slots/naming.go`**

```go
// Req 4.1.1: Filename format must include tokens for differing attributes
// Req 4.1.2: Only conflicting differentiators are required
func ValidateFilenameFormat(slots []Slot, movieFormat, episodeFormat string) error {
    conflicts := GetConflictingAttributes(slots)

    // Req 4.1.3: If slots differ only on HDR, HDR token must be in format
    for _, conflict := range conflicts {
        token := getTokenForAttribute(conflict)
        if !strings.Contains(movieFormat, token) {
            return fmt.Errorf("movie filename format missing required token %s for %s differentiation",
                token, conflict)
        }
        if !strings.Contains(episodeFormat, token) {
            return fmt.Errorf("episode filename format missing required token %s for %s differentiation",
                token, conflict)
        }
    }
    return nil
}

// Req 4.1.4: Validation occurs when saving slot configuration
func (s *Service) UpdateSlot(ctx context.Context, id int64, input UpdateSlotInput) (*Slot, error) {
    // ... update logic
    if err := ValidateFilenameFormat(allSlots, importSettings.MovieFormat, importSettings.EpisodeFormat); err != nil {
        return nil, fmt.Errorf("filename validation failed: %w", err)
    }
    // ... save
}
```

### 3.4 Frontend Implementation

**File: `web/src/routes/settings/media-management.tsx`**

Req 19.1.1: Version slot configuration in Media Management settings page

- Master toggle switch (Req 1.2.1)
- Warning when disabled: "System operates in single-version mode" (Req 1.2.2)
- Slot configuration cards:
  - Name input field (Req 1.1.3)
  - Enable/disable toggle (Req 1.1.4)
  - Quality profile selector dropdown (Req 1.1.5)
- Filename format validation warnings (Req 4.1.5)

### 3.5 Testing

| Test | Requirements |
|------|--------------|
| Cannot create more than 3 slots | 1.1.1 |
| Slots shared between Movies and TV | 1.1.2 |
| Custom slot names saved correctly | 1.1.3 |
| Enable/disable toggle works independently | 1.1.4 |
| Profile assignment persists | 1.1.5 |
| Master toggle enables/disables feature | 1.2.1 |
| Disabled mode uses Slot 1 only | 1.2.2 |
| Missing HDR token in filename blocked | 4.1.1, 4.1.3, 4.1.4 |
| User prompted to update filename format | 4.1.5 |

---

## Phase 4: Assignment Logic

**Spec Group:** D (Assignment Logic)
**Requirements:** 5.1.1-5.1.5, 5.2.1-5.2.3, 5.3.1-5.3.3

### 4.1 Backend Implementation

**File: `internal/library/slots/assignment.go`** (new file)

```go
type SlotAssignment struct {
    SlotID       int64
    SlotNumber   int
    SlotName     string
    MatchScore   float64
    IsUpgrade    bool
    CurrentFile  *MediaFile // nil if slot empty
    Confidence   float64    // How confident in the assignment
}

// Req 5.1.1: Evaluate release against all enabled slot profiles
// Req 5.1.2: Calculate match score for each slot's profile
func (s *Service) EvaluateRelease(ctx context.Context, parsed *scanner.ParsedMedia, mediaType string, mediaID int64) ([]SlotAssignment, error) {
    slots, _ := s.GetEnabledSlots(ctx)
    assignments := make([]SlotAssignment, 0, len(slots))

    for _, slot := range slots {
        score := s.calculateMatchScore(parsed, slot.QualityProfile)
        currentFile := s.getCurrentSlotFile(ctx, mediaType, mediaID, slot.ID)

        assignments = append(assignments, SlotAssignment{
            SlotID:      slot.ID,
            SlotNumber:  slot.SlotNumber,
            SlotName:    slot.Name,
            MatchScore:  score,
            IsUpgrade:   currentFile != nil && score > currentFile.QualityScore,
            CurrentFile: currentFile,
        })
    }
    return assignments, nil
}

// Req 5.1.3: Assign to slot with best quality match
// Req 5.1.4: If scores equal, assign to empty slot (or first if both empty)
// Req 5.1.5: Mutual exclusivity should prevent true ties
func (s *Service) DetermineTargetSlot(ctx context.Context, parsed *scanner.ParsedMedia, mediaType string, mediaID int64) (*SlotAssignment, error) {
    assignments, _ := s.EvaluateRelease(ctx, parsed, mediaType, mediaID)

    // Sort by score descending, then by empty slots first
    sort.Slice(assignments, func(i, j int) bool {
        if assignments[i].MatchScore != assignments[j].MatchScore {
            return assignments[i].MatchScore > assignments[j].MatchScore
        }
        // Prefer empty slots on tie
        iEmpty := assignments[i].CurrentFile == nil
        jEmpty := assignments[j].CurrentFile == nil
        if iEmpty != jEmpty {
            return iEmpty
        }
        // Final tiebreaker: lower slot number
        return assignments[i].SlotNumber < assignments[j].SlotNumber
    })

    return &assignments[0], nil
}
```

### 4.2 Manual Import Handling

**File: `internal/import/service.go`**

```go
// Req 5.2.1: If file could fit multiple slots, prompt user to choose
// Req 5.2.2: Show recommendation based on best match score
// Req 5.2.3: Allow user to override auto-detected target slot
func (s *Service) ProcessImport(ctx context.Context, input ImportInput) (*ImportResult, error) {
    assignments, _ := s.slotService.EvaluateRelease(ctx, parsed, input.MediaType, input.MediaID)

    matchingSlots := filterMatchingSlots(assignments)
    if len(matchingSlots) > 1 && !input.HasTargetSlot() {
        // Req 5.2.1: Prompt user
        return &ImportResult{
            RequiresSlotSelection: true,
            PossibleSlots:         matchingSlots,
            RecommendedSlot:       matchingSlots[0], // Req 5.2.2
        }, nil
    }

    // Req 5.2.3: Use override if provided
    targetSlot := input.TargetSlotID
    if targetSlot == 0 {
        targetSlot = matchingSlots[0].SlotID
    }

    // ... proceed with import
}
```

### 4.3 Edge Case Handling

**File: `internal/library/slots/assignment.go`**

```go
// Req 5.3.1-5.3.3: Handle files below all profile requirements
func (s *Service) HandleBelowProfileImport(ctx context.Context, assignments []SlotAssignment) (*SlotAssignment, error) {
    emptySlots := filterEmptySlots(assignments)
    filledSlots := filterFilledSlots(assignments)

    // Req 5.3.1: All slots empty - accept to closest-matching slot
    if len(filledSlots) == 0 {
        // Fallback to Slot 1 if all scores are 0
        if assignments[0].MatchScore == 0 {
            for i := range assignments {
                if assignments[i].SlotNumber == 1 {
                    assignments[i].NeedsUpgrade = true // Show "upgrade needed"
                    return &assignments[i], nil
                }
            }
        }
        assignments[0].NeedsUpgrade = true
        return &assignments[0], nil
    }

    // Req 5.3.2: Some slots filled, some empty - reject
    if len(emptySlots) > 0 && len(filledSlots) > 0 {
        return nil, ErrRejectImport{Reason: "file below profile requirements with mixed slot states"}
    }

    // Req 5.3.3: All slots filled - reject
    return nil, ErrRejectImport{Reason: "file below profile requirements and all slots filled"}
}
```

### 4.4 Database Changes

**Migration: `XXX_slot_assignments.sql`**

```sql
-- Track per-movie slot assignments (for Req 1.1.6, but schema here)
CREATE TABLE movie_slot_assignments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    movie_id INTEGER NOT NULL REFERENCES movies(id) ON DELETE CASCADE,
    slot_id INTEGER NOT NULL REFERENCES version_slots(id),
    file_id INTEGER REFERENCES movie_files(id) ON DELETE SET NULL,
    monitored INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(movie_id, slot_id)
);

-- Track per-episode slot assignments
CREATE TABLE episode_slot_assignments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    episode_id INTEGER NOT NULL REFERENCES episodes(id) ON DELETE CASCADE,
    slot_id INTEGER NOT NULL REFERENCES version_slots(id),
    file_id INTEGER REFERENCES episode_files(id) ON DELETE SET NULL,
    monitored INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(episode_id, slot_id)
);
```

### 4.5 Testing

| Test | Requirements |
|------|--------------|
| T1: Import matching Slot 1, all empty → Slot 1 | 5.1.3 |
| T2: Import matching Slot 2, all empty → Slot 2 | 5.1.3 |
| T3: Import matching both equally → first empty | 5.1.4, 5.1.5 |
| T5: Import below profiles, all empty → closest slot | 5.3.1 |
| T6: Import below profiles, some filled → reject | 5.3.2, 5.3.3 |
| Manual import prompts for slot selection | 5.2.1 |
| Recommendation shown based on score | 5.2.2 |
| User can override target slot | 5.2.3 |

---

## Phase 5: Status & Monitoring

**Spec Group:** E (Status & Monitoring)
**Requirements:** 1.1.6, 6.1.1-6.1.2, 6.2.1-6.2.2, 8.1.1-8.1.2

### 5.1 Status Determination

**File: `internal/library/movies/status.go`** (new or extend service.go)

```go
// Req 6.1.1: Movie is "missing" if ANY monitored slot is empty
// Req 6.1.2: Unmonitored empty slots do not affect missing status
func (s *Service) CalculateMissingStatus(ctx context.Context, movieID int64) (bool, error) {
    assignments, _ := s.slotService.GetMovieSlotAssignments(ctx, movieID)

    for _, assignment := range assignments {
        // Req 6.1.2: Skip unmonitored slots
        if !assignment.Monitored {
            continue
        }
        // Req 6.1.1: If monitored slot is empty, movie is missing
        if assignment.FileID == nil {
            return true, nil
        }
    }
    return false, nil
}

// Req 6.2.1: Each slot independently tracks upgrade eligibility
// Req 6.2.2: Slot is "upgrade needed" if file below profile cutoff
func (s *Service) CalculateSlotUpgradeStatus(ctx context.Context, movieID int64) ([]SlotUpgradeStatus, error) {
    assignments, _ := s.slotService.GetMovieSlotAssignments(ctx, movieID)
    results := make([]SlotUpgradeStatus, 0, len(assignments))

    for _, assignment := range assignments {
        if assignment.FileID == nil {
            continue // Empty slot, not an upgrade candidate
        }

        file, _ := s.GetFile(ctx, *assignment.FileID)
        profile := assignment.Slot.QualityProfile

        // Req 6.2.2: Check if below cutoff
        needsUpgrade := file.QualityID < profile.Cutoff

        results = append(results, SlotUpgradeStatus{
            SlotID:       assignment.SlotID,
            SlotName:     assignment.Slot.Name,
            NeedsUpgrade: needsUpgrade,
            CurrentQuality: file.Quality,
            CutoffQuality:  profile.GetCutoffQuality().Name,
        })
    }
    return results, nil
}
```

### 5.2 Per-Slot Monitoring

**File: `internal/library/slots/monitoring.go`**

```go
// Req 1.1.6: Each slot has its own independent monitored status per movie/episode
// Req 8.1.1: Each slot has its own monitored toggle per movie/episode
// Req 8.1.2: A slot can be monitored independently
func (s *Service) SetSlotMonitored(ctx context.Context, mediaType string, mediaID int64, slotID int64, monitored bool) error {
    switch mediaType {
    case "movie":
        return s.queries.UpdateMovieSlotMonitored(ctx, sqlc.UpdateMovieSlotMonitoredParams{
            MovieID:   mediaID,
            SlotID:    slotID,
            Monitored: monitored,
        })
    case "episode":
        return s.queries.UpdateEpisodeSlotMonitored(ctx, sqlc.UpdateEpisodeSlotMonitoredParams{
            EpisodeID: mediaID,
            SlotID:    slotID,
            Monitored: monitored,
        })
    }
    return nil
}
```

### 5.3 Database Queries

**File: `internal/database/queries/slots.sql`**

```sql
-- Req 6.1.1: List movies missing in any monitored slot
-- name: ListMoviesMissingInSlots :many
SELECT DISTINCT m.*
FROM movies m
CROSS JOIN version_slots vs
LEFT JOIN movie_slot_assignments msa ON m.id = msa.movie_id AND vs.id = msa.slot_id
WHERE vs.enabled = 1
  AND m.monitored = 1
  AND m.released = 1
  AND (msa.monitored = 1 OR msa.monitored IS NULL)
  AND msa.file_id IS NULL;

-- Req 6.2.1, 6.2.2: List upgrade candidates per slot
-- name: ListMovieUpgradeCandidatesBySlot :many
SELECT m.*, msa.slot_id, mf.quality_id as current_quality_id, qp.cutoff
FROM movies m
JOIN movie_slot_assignments msa ON m.id = msa.movie_id
JOIN movie_files mf ON msa.file_id = mf.id
JOIN version_slots vs ON msa.slot_id = vs.id
JOIN quality_profiles qp ON vs.quality_profile_id = qp.id
WHERE m.monitored = 1
  AND m.released = 1
  AND vs.enabled = 1
  AND msa.monitored = 1
  AND mf.quality_id IS NOT NULL
  AND mf.quality_id < qp.cutoff;
```

### 5.4 Frontend Updates

**File: `web/src/routes/movies/$id.tsx`**
- Per-slot file status display
- Per-slot monitoring toggles (Req 8.1.1, 8.1.2)
- Slot-specific upgrade indicators (Req 6.2.2)

### 5.5 Testing

| Test | Requirements |
|------|--------------|
| T17: Missing with 1 monitored empty, 1 filled → missing | 6.1.1 |
| T18: Missing with 1 unmonitored empty → available | 6.1.2 |
| Upgrade needed shown when below cutoff | 6.2.1, 6.2.2 |
| Per-slot monitoring toggle works | 1.1.6, 8.1.1, 8.1.2 |

---

## Phase 6: Search Integration

**Spec Group:** F (Search Integration)
**Requirements:** 7.1.1-7.1.3, 7.2.1-7.2.2, 8.1.3, 11.1.1-11.1.3

### 6.1 Auto-Search Implementation

**File: `internal/autosearch/service.go`**

```go
// Req 7.1.1: Search in parallel for multiple empty monitored slots
// Req 7.1.2: May grab multiple releases simultaneously
// Req 7.1.3: Each slot's search is independent
// Req 8.1.3: Auto-search only runs for monitored slots
func (s *Service) SearchMovie(ctx context.Context, movieID int64, source SearchSource) error {
    movie, _ := s.movieService.Get(ctx, movieID)

    // Get slots needing search (empty OR upgrade candidates)
    slotsToSearch, _ := s.slotService.GetSlotsNeedingSearch(ctx, "movie", movieID)

    // Req 8.1.3: Filter to monitored only
    monitoredSlots := filterMonitored(slotsToSearch)

    if len(monitoredSlots) == 0 {
        return nil // Nothing to search
    }

    // Req 7.1.1, 7.1.3: Search in parallel, independently
    var wg sync.WaitGroup
    results := make(chan SlotSearchResult, len(monitoredSlots))

    for _, slot := range monitoredSlots {
        wg.Add(1)
        go func(slot SlotInfo) {
            defer wg.Done()
            result := s.searchForSlot(ctx, movie, slot)
            results <- result
        }(slot)
    }

    wg.Wait()
    close(results)

    // Req 7.1.2: Grab for each slot
    for result := range results {
        if result.BestRelease != nil {
            s.grabService.GrabForSlot(ctx, result.BestRelease, result.SlotID)
        }
    }

    return nil
}

// Req 7.2.1: Upgrade replaces and deletes old file
// Req 7.2.2: Standard upgrade per slot, no cross-slot movement
func (s *Service) handleUpgrade(ctx context.Context, slot SlotInfo, newFile *MediaFile) error {
    oldFile := slot.CurrentFile
    if oldFile != nil {
        // Delete old file from disk and database
        s.fileService.Delete(ctx, oldFile.ID)
    }
    // Assign new file to slot
    return s.slotService.AssignFileToSlot(ctx, slot.ID, newFile.ID)
}
```

### 6.2 Search Results Display

**File: `internal/indexer/search/handlers.go`**

```go
// Req 11.1.1: Search results indicate which slot each release would fill
// Req 11.1.2: Show whether grab would be upgrade vs new fill
type EnrichedSearchResult struct {
    *types.TorrentInfo
    TargetSlot     *SlotInfo `json:"targetSlot"`
    IsUpgrade      bool      `json:"isUpgrade"`
    IsNewFill      bool      `json:"isNewFill"`
    MatchScore     float64   `json:"matchScore"`
}

func (h *Handlers) SearchTorrentsWithSlots(c echo.Context) error {
    // ... existing search logic

    // Enrich results with slot info
    for i, result := range results {
        assignment, _ := h.slotService.DetermineTargetSlot(ctx, result.ParsedMedia, mediaType, mediaID)
        results[i].TargetSlot = assignment.Slot
        results[i].IsUpgrade = assignment.IsUpgrade
        results[i].IsNewFill = assignment.CurrentFile == nil
        results[i].MatchScore = assignment.MatchScore
    }

    return c.JSON(http.StatusOK, results)
}
```

### 6.3 Grab Request Enhancement

**File: `internal/indexer/grab/handlers.go`**

```go
// Req 11.1.3: Allow user to override auto-detected slot when grabbing
type GrabRequest struct {
    // ... existing fields
    TargetSlotID *int64 `json:"targetSlotId,omitempty"` // Optional override
}

func (h *Handlers) Grab(c echo.Context) error {
    var req GrabRequest
    c.Bind(&req)

    // If no slot specified, auto-detect
    targetSlot := req.TargetSlotID
    if targetSlot == nil {
        assignment, _ := h.slotService.DetermineTargetSlot(ctx, parsed, req.MediaType, req.MediaID)
        targetSlot = &assignment.SlotID
    }

    // ... proceed with grab using targetSlot
}
```

### 6.4 Frontend Updates

**File: `web/src/components/search/SearchModal.tsx`**

- Add "Target Slot" column to results table (Req 11.1.1)
- Show upgrade/fill badge per result (Req 11.1.2)
- Add slot selector dropdown in grab confirmation dialog (Req 11.1.3)

### 6.5 Testing

| Test | Requirements |
|------|--------------|
| T8: Auto-search with 2 empty monitored → parallel search | 7.1.1, 7.1.2 |
| T9: Auto-search with 1 monitored, 1 unmonitored → only monitored | 8.1.3 |
| T7: Upgrade within slot → replace, delete old | 7.2.1, 7.2.2 |
| Search results show slot assignment | 11.1.1 |
| Upgrade vs fill indicator shown | 11.1.2 |
| Manual slot override works | 11.1.3 |

---

## Phase 7: File Operations

**Spec Group:** G (File Operations)
**Requirements:** 9.1.1-9.1.3, 12.1.1-12.1.2, 12.2.1-12.2.2, 13.1.1-13.1.3

### 7.1 File Organization

**File: `internal/library/organizer/organizer.go`**

```go
// Req 9.1.1: Multiple versions stored in same directory
// Req 9.1.2: Files differentiated by quality suffix in filename
// Req 9.1.3: No slot identifiers added to paths
func (o *Organizer) OrganizeMovie(ctx context.Context, sourcePath string, movie *Movie, slot *Slot) (string, error) {
    // Directory: same for all slots (Req 9.1.1, 9.1.3)
    dir := o.GenerateMoviePath(movie.RootPath, movie.Tokens)

    // Filename: includes quality suffix for differentiation (Req 9.1.2)
    // Slot-specific attributes come from the file, not a slot identifier
    filename := o.GenerateMovieFilename(movie.Tokens) // Uses {Quality}, {HDR}, etc.

    return filepath.Join(dir, filename), nil
}
```

### 7.2 Deletion Behavior

**File: `internal/library/movies/service.go`**

```go
// Req 12.1.1: Deleting file does NOT trigger automatic search
// Req 12.1.2: Slot becomes empty; waits for next scheduled search
func (s *Service) DeleteFile(ctx context.Context, fileID int64) error {
    file, _ := s.GetFile(ctx, fileID)

    // Remove from disk
    os.Remove(file.Path)

    // Update slot assignment - file_id becomes NULL
    s.slotService.UnassignFile(ctx, fileID)

    // Delete database record
    s.queries.DeleteMovieFile(ctx, fileID)

    // Req 12.1.1: Do NOT trigger search - just leave slot empty
    // Next scheduled search will find it

    return nil
}
```

### 7.3 Slot Disable Handling

**File: `internal/library/slots/service.go`**

```go
// Req 12.2.1: When user disables slot with files, prompt for action
// Req 12.2.2: Options: delete files, keep unassigned, or cancel
type DisableSlotRequest struct {
    SlotID int64                  `json:"slotId"`
    Action DisableSlotAction      `json:"action"` // "delete", "keep", "cancel"
}

type DisableSlotAction string
const (
    DisableActionDelete DisableSlotAction = "delete"  // Delete all files in slot
    DisableActionKeep   DisableSlotAction = "keep"    // Keep files but unassign
    DisableActionCancel DisableSlotAction = "cancel"  // Abort disable
)

func (s *Service) DisableSlot(ctx context.Context, req DisableSlotRequest) error {
    // Check if slot has files
    assignments, _ := s.GetAllAssignmentsForSlot(ctx, req.SlotID)
    hasFiles := len(filterWithFiles(assignments)) > 0

    if hasFiles && req.Action == "" {
        // Req 12.2.1: Prompt needed
        return ErrPromptRequired{
            Message: "Slot has files assigned",
            Options: []DisableSlotAction{DisableActionDelete, DisableActionKeep, DisableActionCancel},
        }
    }

    switch req.Action {
    case DisableActionDelete:
        // Delete all files assigned to this slot
        s.deleteFilesInSlot(ctx, req.SlotID)
    case DisableActionKeep:
        // Just unassign files (set slot_id to NULL)
        s.unassignFilesFromSlot(ctx, req.SlotID)
    case DisableActionCancel:
        return nil // Do nothing
    }

    // Disable the slot
    return s.queries.UpdateSlotEnabled(ctx, sqlc.UpdateSlotEnabledParams{
        ID:      req.SlotID,
        Enabled: false,
    })
}
```

### 7.4 Library Scanning

**File: `internal/library/scanner/scanner.go`**

```go
// Req 13.1.1: Scanner discovers all files in directories
// Req 13.1.2: Auto-assign each file to best-matching slot
// Req 13.1.3: Extra files (more than slot count) queued for review
func (s *Scanner) ScanMovieFolder(ctx context.Context, moviePath string, movie *Movie) (*ScanResult, error) {
    // Req 13.1.1: Discover all files
    files, _ := s.discoverVideoFiles(moviePath)

    enabledSlots, _ := s.slotService.GetEnabledSlots(ctx)
    maxSlots := len(enabledSlots)

    assigned := make([]AssignedFile, 0, maxSlots)
    reviewQueue := make([]UnassignedFile, 0)

    // Sort files by quality (best first) for assignment priority
    sortByQuality(files)

    for _, file := range files {
        // Req 13.1.2: Auto-assign to best-matching slot
        assignment, _ := s.slotService.DetermineTargetSlot(ctx, file.Parsed, "movie", movie.ID)

        if assignment == nil || isSlotAlreadyAssigned(assigned, assignment.SlotID) {
            // Req 13.1.3: Queue for review if no slot or slot taken
            reviewQueue = append(reviewQueue, UnassignedFile{
                Path:    file.Path,
                Quality: file.Parsed.Quality,
                Reason:  "No matching slot available",
            })
            continue
        }

        assigned = append(assigned, AssignedFile{
            File:   file,
            SlotID: assignment.SlotID,
        })
    }

    return &ScanResult{
        Assigned:    assigned,
        ReviewQueue: reviewQueue,
    }, nil
}
```

### 7.5 Testing

| Test | Requirements |
|------|--------------|
| Multiple files in same directory | 9.1.1 |
| Files differentiated by quality in filename | 9.1.2 |
| No slot ID in file path | 9.1.3 |
| Delete file does not trigger search | 12.1.1 |
| Deleted slot becomes empty | 12.1.2 |
| T10: Disable slot with files → prompt | 12.2.1, 12.2.2 |
| T12: Scan with more files than slots → queue extras | 13.1.3 |

---

## Phase 8: Queue & History

**Spec Group:** H (Queue & History)
**Requirements:** 10.1.1-10.1.2, 10.2.1-10.2.2, 17.1.1-17.1.2

### 8.1 Queue Display

**File: `internal/downloader/queue.go`**

```go
// Req 10.1.1: Queue shows raw downloads with mapped media
// Req 10.1.2: Target slot info shown inline
type QueueItem struct {
    // ... existing fields
    TargetSlotID   *int64  `json:"targetSlotId"`
    TargetSlotName string  `json:"targetSlotName"`
}

func (s *Service) enrichQueueItemsWithSlots(items []QueueItem) []QueueItem {
    for i, item := range items {
        if item.DownloadMapping != nil && item.DownloadMapping.TargetSlotID != nil {
            slot, _ := s.slotService.GetSlot(ctx, *item.DownloadMapping.TargetSlotID)
            items[i].TargetSlotID = &slot.ID
            items[i].TargetSlotName = slot.Name
        }
    }
    return items
}
```

**Migration: `XXX_download_mapping_slot.sql`**

```sql
-- Req 10.1.2: Track target slot in download mappings
ALTER TABLE download_mappings ADD COLUMN target_slot_id INTEGER REFERENCES version_slots(id);
```

### 8.2 Failed Download Handling

**File: `internal/downloader/completion.go`**

```go
// Req 10.2.1: If download fails or rejected, slot reverts to "empty" immediately
// Req 10.2.2: No pending/retry state; waits for next search
func (s *Service) HandleDownloadFailed(ctx context.Context, mapping *DownloadMapping) error {
    // Req 10.2.1: Clear the slot assignment immediately
    if mapping.TargetSlotID != nil {
        s.slotService.ClearPendingAssignment(ctx, mapping.MediaType, mapping.MediaID, *mapping.TargetSlotID)
    }

    // Req 10.2.2: No retry - just leave as empty
    // Next scheduled search will pick it up

    return nil
}
```

### 8.3 History Logging

**File: `internal/history/types.go`**

```go
// Req 17.1.2: History entries include slot information
type SlotEventData struct {
    SlotID   int64  `json:"slotId"`
    SlotName string `json:"slotName"`
}

// Extended event data types
type ImportEventData struct {
    // ... existing fields
    SlotEventData // Embed slot info
}

type GrabEventData struct {
    // ... existing fields
    SlotEventData // Embed slot info
}

// Req 17.1.1: New event types for slot-related events
const (
    EventSlotAssigned   EventType = "slot_assigned"
    EventSlotReassigned EventType = "slot_reassigned"
    EventSlotUnassigned EventType = "slot_unassigned"
)
```

**File: `internal/history/service.go`**

```go
// Req 17.1.1: Log all slot-related events
func (s *Service) LogSlotAssignment(ctx context.Context, mediaType string, mediaID int64, slotID int64, fileID int64) error {
    slot, _ := s.slotService.GetSlot(ctx, slotID)

    return s.CreateEntry(ctx, CreateEntryInput{
        EventType: EventSlotAssigned,
        MediaType: mediaType,
        MediaID:   mediaID,
        Data: SlotAssignmentData{
            SlotID:   slotID,
            SlotName: slot.Name,
            FileID:   fileID,
        },
    })
}
```

### 8.4 Frontend Updates

**File: `web/src/components/queue/QueueTable.tsx`**
- Add "Target Slot" column (Req 10.1.2)
- Color-code by slot

**File: `web/src/routes/history.tsx`**
- Show slot info in history entries (Req 17.1.2)
- Filter by slot option

### 8.5 Testing

| Test | Requirements |
|------|--------------|
| Queue shows mapped media | 10.1.1 |
| Queue shows target slot | 10.1.2 |
| Failed download clears slot immediately | 10.2.1 |
| No retry state after failure | 10.2.2 |
| Slot assignment logged | 17.1.1 |
| History shows slot info | 17.1.2 |

---

## Phase 9: Migration System

**Spec Group:** I (Migration)
**Requirements:** 1.2.3, 14.1.1-14.1.5, 14.2.1-14.2.3, 14.3.1-14.3.4, 15.1.1-15.1.2

### 9.1 Dry Run Preview

**File: `internal/library/slots/migration.go`** (new file)

```go
// Req 14.1.1: Dry run preview is required before enabling
// Req 1.2.3: Enabling requires completing dry-run first
func (s *Service) EnableMultiVersion(ctx context.Context) error {
    settings, _ := s.GetSettings(ctx)

    // Req 1.2.3: Check dry-run completed
    if !settings.DryRunCompleted {
        return ErrDryRunRequired
    }

    return s.queries.UpdateMultiVersionEnabled(ctx, true)
}

// Req 14.1.2: Preview organized by type, then per-item
// Req 14.1.3: TV shows per-series, per-season breakdown
// Req 14.1.4: Show proposed slot assignment for each file
// Req 14.1.5: Show conflicts and files that can't be matched
type MigrationPreview struct {
    Movies   []MovieMigrationPreview   `json:"movies"`
    TVShows  []TVShowMigrationPreview  `json:"tvShows"`
    Summary  MigrationSummary          `json:"summary"`
}

type MovieMigrationPreview struct {
    MovieID       int64                    `json:"movieId"`
    Title         string                   `json:"title"`
    Files         []FileMigrationPreview   `json:"files"`
    HasConflict   bool                     `json:"hasConflict"`
}

type TVShowMigrationPreview struct {
    SeriesID int64                      `json:"seriesId"`
    Title    string                     `json:"title"`
    Seasons  []SeasonMigrationPreview   `json:"seasons"` // Req 14.1.3
}

type FileMigrationPreview struct {
    FileID          int64   `json:"fileId"`
    Path            string  `json:"path"`
    Quality         string  `json:"quality"`
    ProposedSlotID  *int64  `json:"proposedSlotId"`  // Req 14.1.4
    ProposedSlotName string `json:"proposedSlotName"`
    MatchScore      float64 `json:"matchScore"`
    Conflict        string  `json:"conflict,omitempty"` // Req 14.1.5
    NeedsReview     bool    `json:"needsReview"`
}

func (s *Service) GenerateMigrationPreview(ctx context.Context) (*MigrationPreview, error) {
    // ... implementation
}
```

### 9.2 Assignment Logic

**File: `internal/library/slots/migration.go`**

```go
// Req 14.2.1: Intelligently assign based on quality profile matching
// Req 14.2.2: Files that can't match go to review queue
// Req 14.2.3: Profile must be assigned before saving
func (s *Service) ExecuteMigration(ctx context.Context) error {
    // Req 14.2.3: Validate all slots have profiles
    slots, _ := s.GetEnabledSlots(ctx)
    for _, slot := range slots {
        if slot.QualityProfileID == nil {
            return fmt.Errorf("slot %s has no profile assigned", slot.Name)
        }
    }

    // Process all existing files
    movies, _ := s.movieService.ListWithFiles(ctx)
    for _, movie := range movies {
        for _, file := range movie.Files {
            assignment, err := s.DetermineTargetSlot(ctx, file.Parsed, "movie", movie.ID)

            if err != nil || assignment == nil {
                // Req 14.2.2: Queue for review
                s.AddToReviewQueue(ctx, "movie", movie.ID, file.ID, "no matching slot")
                continue
            }

            // Req 14.2.1: Assign to matched slot
            s.AssignFileToSlot(ctx, assignment.SlotID, file.ID)
        }
    }

    // Mark dry-run as completed
    s.queries.UpdateDryRunCompleted(ctx, true)

    return nil
}
```

### 9.3 Review Queue

**File: `internal/library/slots/review.go`** (new file)

```go
// Req 14.3.1: Dedicated review page
// Req 14.3.2: Show file details, quality, slot options
// Req 14.3.3: Allow assignment to slot or deletion
// Req 14.3.4: Movies with more files than slots → extras to review
type ReviewQueueItem struct {
    ID          int64           `json:"id"`
    MediaType   string          `json:"mediaType"`
    MediaID     int64           `json:"mediaId"`
    MediaTitle  string          `json:"mediaTitle"`
    FileID      int64           `json:"fileId"`
    FilePath    string          `json:"filePath"`
    Quality     string          `json:"quality"`
    Reason      string          `json:"reason"`
    SlotOptions []SlotOption    `json:"slotOptions"` // Req 14.3.2
}

func (s *Service) GetReviewQueue(ctx context.Context) ([]ReviewQueueItem, error) {
    // ... implementation
}

// Req 14.3.3: Allow assignment or deletion
func (s *Service) ResolveReviewItem(ctx context.Context, itemID int64, action ReviewAction, targetSlotID *int64) error {
    switch action {
    case ReviewActionAssign:
        return s.AssignFileToSlot(ctx, *targetSlotID, item.FileID)
    case ReviewActionDelete:
        return s.fileService.DeleteFile(ctx, item.FileID)
    }
    return nil
}
```

### 9.4 Profile Change Handling

**File: `internal/library/slots/service.go`**

```go
// Req 15.1.1: When profile changes with files assigned, prompt for action
// Req 15.1.2: Options: keep, re-evaluate, or cancel
type ProfileChangeRequest struct {
    SlotID           int64              `json:"slotId"`
    NewProfileID     int64              `json:"newProfileId"`
    Action           ProfileChangeAction `json:"action"`
}

type ProfileChangeAction string
const (
    ProfileChangeKeep       ProfileChangeAction = "keep"       // Keep current assignments
    ProfileChangeReevaluate ProfileChangeAction = "reevaluate" // Re-evaluate, queue non-matches
    ProfileChangeCancel     ProfileChangeAction = "cancel"
)

func (s *Service) ChangeSlotProfile(ctx context.Context, req ProfileChangeRequest) error {
    // Check if slot has files
    assignments, _ := s.GetAssignmentsForSlot(ctx, req.SlotID)
    hasFiles := len(filterWithFiles(assignments)) > 0

    if hasFiles && req.Action == "" {
        // Req 15.1.1: Prompt required
        return ErrPromptRequired{
            Message: "Slot has files assigned. Changing profile may affect matching.",
            Options: []ProfileChangeAction{ProfileChangeKeep, ProfileChangeReevaluate, ProfileChangeCancel},
        }
    }

    switch req.Action {
    case ProfileChangeKeep:
        // Just update profile, keep assignments
    case ProfileChangeReevaluate:
        // Re-evaluate all files, queue non-matches to review
        s.reevaluateSlotAssignments(ctx, req.SlotID, req.NewProfileID)
    case ProfileChangeCancel:
        return nil
    }

    return s.queries.UpdateSlotProfile(ctx, sqlc.UpdateSlotProfileParams{
        ID:               req.SlotID,
        QualityProfileID: req.NewProfileID,
    })
}
```

### 9.5 Frontend Pages

**File: `web/src/routes/settings/migration-preview.tsx`** (new)
- Display preview organized by type (Req 14.1.2)
- Collapsible TV show/season sections (Req 14.1.3)
- Show slot assignments per file (Req 14.1.4)
- Highlight conflicts (Req 14.1.5)

**File: `web/src/routes/settings/review-queue.tsx`** (new)
- List files needing review (Req 14.3.1)
- Show file details and quality (Req 14.3.2)
- Slot assignment dropdown and delete button (Req 14.3.3)

### 9.6 Testing

| Test | Requirements |
|------|--------------|
| Cannot enable without dry-run | 1.2.3, 14.1.1 |
| Preview shows movies and TV separately | 14.1.2 |
| TV preview has season breakdown | 14.1.3 |
| Preview shows proposed slot per file | 14.1.4 |
| Preview shows conflicts | 14.1.5 |
| Files auto-assigned during migration | 14.2.1 |
| Unmatched files go to review | 14.2.2 |
| Cannot save slot without profile | 14.2.3 |
| Review page shows file details | 14.3.1, 14.3.2 |
| Can assign or delete from review | 14.3.3 |
| Extra files go to review | 14.3.4 |
| T11: Change profile with files → prompt | 15.1.1, 15.1.2 |

---

## Phase 10: TV-Specific Behavior

**Spec Group:** J (TV-Specific)
**Requirements:** 16.1.1-16.1.2, 16.2.1-16.2.3

### 10.1 Episode Independence

**File: `internal/library/tv/service.go`**

```go
// Req 16.1.1: Each episode independently tracks which slots are filled
// Req 16.1.2: Different episodes may have different slot fills
func (s *Service) GetEpisodeSlotStatus(ctx context.Context, episodeID int64) ([]EpisodeSlotStatus, error) {
    assignments, _ := s.slotService.GetEpisodeSlotAssignments(ctx, episodeID)

    results := make([]EpisodeSlotStatus, len(assignments))
    for i, a := range assignments {
        results[i] = EpisodeSlotStatus{
            SlotID:    a.SlotID,
            SlotName:  a.Slot.Name,
            HasFile:   a.FileID != nil,
            Monitored: a.Monitored,
        }
    }
    return results, nil
}

// Example: S01 may have 4K (Slot 1), S02 may have only 1080p (Slot 2)
// Each episode tracks independently per Req 16.1.1, 16.1.2
```

### 10.2 Season Pack Handling

**File: `internal/import/season_pack.go`**

```go
// Req 16.2.1: Season pack assigned to best matching slot
// Req 16.2.2: May result in mixed slots across seasons
// Req 16.2.3: Each episode from pack individually assessed
func (s *Service) ImportSeasonPack(ctx context.Context, pack *SeasonPack, seriesID int64) error {
    // Req 16.2.1: Determine slot for the pack based on quality
    packSlot, _ := s.slotService.DetermineTargetSlot(ctx, pack.Parsed, "episode", 0)

    // Req 16.2.3: Process each episode individually
    for _, episodeFile := range pack.Episodes {
        episode, _ := s.tvService.GetOrCreateEpisode(ctx, seriesID, episodeFile.Season, episodeFile.Episode)

        // Individual assessment for each episode
        assignment, _ := s.slotService.DetermineTargetSlot(ctx, episodeFile.Parsed, "episode", episode.ID)

        // Use pack slot as default if individual assessment unclear
        targetSlot := assignment.SlotID
        if targetSlot == 0 {
            targetSlot = packSlot.SlotID
        }

        s.importEpisodeFile(ctx, episodeFile, episode.ID, targetSlot)
    }

    // Req 16.2.2: Mixed slots across seasons is acceptable
    // No enforcement of consistency

    return nil
}
```

### 10.3 Testing

| Test | Requirements |
|------|--------------|
| Episodes track slots independently | 16.1.1 |
| S01 in Slot 1, S02 in Slot 2 works | 16.1.2 |
| T16: Season pack imports to matching slot per episode | 16.2.1, 16.2.3 |
| Mixed slots across seasons allowed | 16.2.2 |

---

## Phase 11: API Endpoints

**Spec Group:** K (API)
**Requirements:** 18.1.1-18.1.2, 18.2.1-18.2.2

### 11.1 Slot Management API

**File: `internal/library/slots/handlers.go`**

```go
// Req 18.1.1: Full CRUD endpoints for slot assignments
// Req 18.1.2: Endpoints to view, assign, reassign, and unassign
func (h *Handlers) RegisterRoutes(g *echo.Group) {
    // Global slot configuration
    g.GET("/slots", h.ListSlots)
    g.GET("/slots/:id", h.GetSlot)
    g.PUT("/slots/:id", h.UpdateSlot)

    // Multi-version settings
    g.GET("/multiversion", h.GetSettings)
    g.PUT("/multiversion", h.UpdateSettings)
    g.POST("/multiversion/preview", h.GeneratePreview)
    g.POST("/multiversion/migrate", h.ExecuteMigration)
    g.GET("/multiversion/review", h.GetReviewQueue)
    g.POST("/multiversion/review/:id", h.ResolveReviewItem)

    // Movie slot assignments (Req 18.1.2)
    g.GET("/movies/:id/slots", h.GetMovieSlotAssignments)
    g.PUT("/movies/:id/slots/:slotId", h.UpdateMovieSlotAssignment)
    g.POST("/movies/:id/slots/:slotId/assign", h.AssignMovieFileToSlot)
    g.POST("/movies/:id/slots/:slotId/unassign", h.UnassignMovieSlot)

    // Episode slot assignments (Req 18.1.2)
    g.GET("/series/:seriesId/episodes/:episodeId/slots", h.GetEpisodeSlotAssignments)
    g.PUT("/series/:seriesId/episodes/:episodeId/slots/:slotId", h.UpdateEpisodeSlotAssignment)
    g.POST("/series/:seriesId/episodes/:episodeId/slots/:slotId/assign", h.AssignEpisodeFileToSlot)
    g.POST("/series/:seriesId/episodes/:episodeId/slots/:slotId/unassign", h.UnassignEpisodeSlot)
}
```

### 11.2 Grab Request Enhancement

**File: `internal/indexer/grab/handlers.go`**

```go
// Req 18.2.1: API grab requests accept optional target_slot parameter
// Req 18.2.2: If omitted, auto-detect best slot
type GrabRequest struct {
    IndexerID    int64   `json:"indexerId"`
    ReleaseID    string  `json:"releaseId"`
    MediaType    string  `json:"mediaType"`
    MediaID      int64   `json:"mediaId"`
    ClientID     *int64  `json:"clientId,omitempty"`
    TargetSlotID *int64  `json:"targetSlotId,omitempty"` // Req 18.2.1
}

func (h *Handlers) Grab(c echo.Context) error {
    var req GrabRequest
    if err := c.Bind(&req); err != nil {
        return c.JSON(http.StatusBadRequest, err)
    }

    targetSlot := req.TargetSlotID

    // Req 18.2.2: Auto-detect if not provided
    if targetSlot == nil {
        assignment, _ := h.slotService.DetermineTargetSlot(ctx, parsed, req.MediaType, req.MediaID)
        targetSlot = &assignment.SlotID
    }

    result, err := h.grabService.Grab(ctx, GrabInput{
        // ... other fields
        TargetSlotID: targetSlot,
    })

    return c.JSON(http.StatusOK, result)
}
```

### 11.3 Testing

| Test | Requirements |
|------|--------------|
| GET /slots returns all slots | 18.1.1 |
| GET /movies/:id/slots returns assignments | 18.1.2 |
| PUT /movies/:id/slots/:slotId updates assignment | 18.1.2 |
| POST grab with targetSlotId works | 18.2.1 |
| POST grab without targetSlotId auto-detects | 18.2.2 |

---

## Phase 12: UI Components

**Spec Group:** L (UI)
**Requirements:** 19.1.1-19.1.2

### 12.1 Settings Page

**File: `web/src/routes/settings/media-management.tsx`**

Req 19.1.1: Version slot configuration in Media Management settings page

Components:
- Multi-version master toggle with dry-run requirement
- Slot configuration cards (name, profile, enable/disable)
- Exclusivity validation display
- Filename format validation warnings
- Migration preview and review queue links

### 12.2 Detail Pages

**File: `web/src/routes/movies/$id.tsx`**
**File: `web/src/routes/series/$id.tsx`**

Req 19.1.2: Movie/episode detail page shows files with slot assignments

Components:
- File table with slot column
- Per-slot status badges (filled, empty, upgrade needed)
- Per-slot monitoring toggles
- Per-slot search buttons
- Slot assignment dropdown for manual file assignment

### 12.3 Testing

| Test | Requirements |
|------|--------------|
| Slot config in Media Management | 19.1.1 |
| Files shown in table with slots | 19.1.2 |

---

## Phase 13: Debug & Testing Features

**Spec Group:** M (Testing)
**Requirements:** 20.1, 20.2.1-20.2.5

### 13.1 Backend Test Matrix (Req 20.1)

Implement comprehensive tests covering T1-T18 scenarios from spec:

| ID | Scenario | Test File |
|----|----------|-----------|
| T1 | Import file matching Slot 1, all empty | `slots/assignment_test.go` |
| T2 | Import file matching Slot 2, all empty | `slots/assignment_test.go` |
| T3 | Import file matching both equally | `slots/assignment_test.go` |
| T4 | Import to filled slot, empty slot available | `import/service_test.go` |
| T5 | Import below profiles, all empty | `slots/assignment_test.go` |
| T6 | Import below profiles, some filled | `slots/assignment_test.go` |
| T7 | Upgrade within slot | `autosearch/service_test.go` |
| T8 | Auto-search with 2 empty monitored | `autosearch/service_test.go` |
| T9 | Auto-search with mixed monitoring | `autosearch/service_test.go` |
| T10 | Disable slot with files | `slots/service_test.go` |
| T11 | Change profile with files | `slots/service_test.go` |
| T12 | Scan with more files than slots | `scanner/scanner_test.go` |
| T13 | DV+HDR10 file, conflicting slot profiles | `quality/matcher_test.go` |
| T14 | Multi-audio Remux | `quality/matcher_test.go` |
| T15 | Unknown codec, required profile | `quality/matcher_test.go` |
| T16 | Season pack mixed slots | `import/season_pack_test.go` |
| T17 | Missing status, 1 monitored empty | `movies/status_test.go` |
| T18 | Missing status, 1 unmonitored empty | `movies/status_test.go` |

### 13.2 Developer Mode Debug Features

**File: `web/src/routes/settings/debug/slots.tsx`** (new, gated)

Req 20.2.5: All debug features gated behind `developerMode`

```tsx
export function SlotDebugPanel() {
  const developerMode = useDeveloperMode();
  if (!developerMode) return null;

  return (
    <div>
      {/* Req 20.2.1: Slot State Viewer */}
      <SlotStateViewer />

      {/* Req 20.2.2: Mock File Import */}
      <MockFileImporter />

      {/* Req 20.2.3: Profile Matching Tester */}
      <ProfileMatchingTester />

      {/* Req 20.2.4: Migration Simulator */}
      <MigrationSimulator />
    </div>
  );
}
```

**Req 20.2.1: Slot State Viewer**
- Panel showing raw slot assignments for any movie/episode
- Display match scores for each slot

**Req 20.2.2: Mock File Import**
- Form to input custom quality attributes
- Test matching logic without real files

**Req 20.2.3: Profile Matching Tester**
- Input release attributes (resolution, HDR, codec, etc.)
- See which slots would match and their scores

**Req 20.2.4: Migration Simulator**
- Preview migration results without enabling
- Dry-run without committing changes

### 13.3 Testing

| Test | Requirements |
|------|--------------|
| All T1-T18 scenarios covered | 20.1 |
| Debug panel hidden when not developerMode | 20.2.5 |
| Slot state viewer shows raw data | 20.2.1 |
| Mock import tests matching | 20.2.2 |
| Profile tester shows scores | 20.2.3 |
| Migration simulator works without enabling | 20.2.4 |

---

## Database Migration Summary

| Order | Migration File | Requirements |
|-------|---------------|--------------|
| 1 | `XXX_quality_profile_attributes.sql` | 2.1.1-2.1.5 |
| 2 | `XXX_version_slots.sql` | 1.1.1-1.1.5, 1.2.1-1.2.2 |
| 3 | `XXX_slot_assignments.sql` | 1.1.6 (schema for per-media monitoring) |
| 4 | `XXX_download_mapping_slot.sql` | 10.1.2 |

---

## Implementation Order

1. **Phase 1** - Quality Profile Extensions (2.1.x, 2.2, 2.3.x, 2.4.x, 2.5.x)
2. **Phase 2** - Mutual Exclusivity (3.1.x)
3. **Phase 3** - Slot Infrastructure (1.1.1-1.1.5, 1.2.1-1.2.2, 4.1.x)
4. **Phase 4** - Assignment Logic (5.1.x, 5.2.x, 5.3.x)
5. **Phase 5** - Status & Monitoring (1.1.6, 6.1.x, 6.2.x, 8.1.1-8.1.2)
6. **Phase 6** - Search Integration (7.1.x, 7.2.x, 8.1.3, 11.1.x)
7. **Phase 7** - File Operations (9.1.x, 12.1.x, 12.2.x, 13.1.x)
8. **Phase 8** - Queue & History (10.1.x, 10.2.x, 17.1.x)
9. **Phase 10** - TV-Specific (16.1.x, 16.2.x) - can run parallel with Phase 8
10. **Phase 11** - API Endpoints (18.1.x, 18.2.x)
11. **Phase 9** - Migration System (1.2.3, 14.x, 15.x)
12. **Phase 12** - UI Components (19.1.x)
13. **Phase 13** - Debug & Testing (20.x) - parallel with all phases

---

## Verification Checklist

All 89 requirements mapped:
- Section 1: 9 requirements ✓
- Section 2: 12 requirements ✓
- Section 3: 4 requirements ✓
- Section 4: 5 requirements ✓
- Section 5: 11 requirements ✓
- Section 6: 4 requirements ✓
- Section 7: 5 requirements ✓
- Section 8: 3 requirements ✓
- Section 9: 3 requirements ✓
- Section 10: 4 requirements ✓
- Section 11: 3 requirements ✓
- Section 12: 4 requirements ✓
- Section 13: 3 requirements ✓
- Section 14: 12 requirements ✓
- Section 15: 2 requirements ✓
- Section 16: 5 requirements ✓
- Section 17: 2 requirements ✓
- Section 18: 4 requirements ✓
- Section 19: 2 requirements ✓
- Section 20: 6 requirements ✓

**Total: 89 requirements mapped to 13 phases**
