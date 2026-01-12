# Release Importing

## Feature Overview

Moving media files from the download directory to the appropriate root directory based on media type (movie or series) upon download completion. Includes renaming, character replacement, hardlinks/symlinks to faciliate seeding completed downloads, library status updates, integration with health system in case of error, queue management, etc.

## Import Entry Point

This section defines how the system detects when files are ready for import and initiates the import process.

### Download-to-Library Relationship

#### Queue Schema
- **Separate mapping table** links queue items to library entries
- Queue item creation flow: Send to download client → Create queue record from client response
- Junction table `queue_media` maps queue items to episodes/movies: `(queue_id, episode_id, movie_id, file_status)`

#### Multi-Episode Downloads (Season Packs)
- Junction table tracks individual episode status within a queue item
- Each episode in a pack has its own status: `pending`, `downloading`, `ready`, `imported`, `failed`
- Enables importing individual episodes as they complete (not waiting for full pack)

### Import Triggers

#### Filesystem Watcher
- **Primary trigger**: Watch download client folders for file changes
- **Activation**: Watcher is **only active when queue has items** (not always-on)
- Monitor for new files, size changes, and file completion

#### Startup Behavior
- **Full scan on startup**: Check all download folders for importable files
- Catches any downloads completed while SlipStream was offline
- Matches found files against pending queue items

### File Completion Detection

Since not all download clients use `.part` extension for incomplete files, multiple detection methods are used:

#### Detection Methods (in priority order)
1. **Download client API**: Query client for per-file completion status (preferred)
2. **File size stability**: Wait until file size unchanged for **15-30 seconds**
3. **File lock detection**: Check if file is still locked by download client process

#### Archive Detection
- If `.rar`, `.zip`, or other archive files are present, wait for extraction
- Look for extracted folders/files before attempting import
- Avoids importing archive files instead of extracted video

### Pre-Import Validation

#### Validation Levels (Configurable)
User chooses validation strictness:
- **Basic**: File exists and size > 0
- **Standard**: Size > minimum threshold, valid video extension
- **Full**: Size + extension + MediaInfo probe to verify valid video

#### Minimum File Size
- Default: **100 MB** minimum file size
- Filters out sample files and promotional clips
- Configurable per quality profile if needed

#### Sample File Detection
Detect and skip sample files using **both methods**:
- Files in folders named `sample`, `samples`
- Files with `sample` in the filename
- Combined detection prevents importing promotional samples

#### Video File Extensions
- **Configurable whitelist** of accepted video extensions
- User defines which extensions to import
- Default includes: `.mkv`, `.mp4`, `.avi`, `.m4v`, `.mov`, `.wmv`, `.ts`, `.m2ts`, `.webm`

### File-to-Library Matching

#### Queue-Based Matching (Primary)
- Queue items directly reference library entries via mapping table
- Import already knows exactly which episode/movie the file belongs to
- No filename parsing needed for queue-initiated downloads

#### Match Conflict Handling
If filename parsing produces different match than queue record:
- **Configurable behavior**: User chooses trust-queue vs trust-parse vs fail-with-warning
- Default: Trust queue record (user initiated the download)

#### Manual Import Matching
For files not from queue (manual imports):
- System **suggests best match** to missing/wanted episodes
- **User must confirm** the match before import proceeds
- Can browse any accessible path via manual import UI

### Unknown Media Handling

When download completes but doesn't match existing library:
- **Configurable via toggle** in settings:
  - **Ignore mode**: Reject files that don't match library items
  - **Auto-add mode**: Create new library entry automatically, trigger metadata refresh

### Import Processing

#### Concurrency
- **Sequential processing**: Import one file at a time
- Prevents I/O contention and simplifies error handling
- Queue items with multiple files process files one-by-one

#### Torrent Seeding
- **Import immediately with hardlinks**
- Original file in download folder continues seeding with original name
- Library hardlink gets renamed filename
- Seeding is unaffected by import

#### Import Delay
- **Configurable delay per download client**
- Allows time for file writes to complete, post-processing to finish
- Default: 0 (no delay), user can set per-client

### Error Handling

#### Retry Logic
- **Immediate retry once**, then mark as failed
- Total: 2 attempts per file
- No exponential backoff or scheduled retries

#### Partial Failure (Multi-File Downloads)
- **Continue importing other files** if one fails
- After all files attempted, **retry failed files once** each
- Each file's status tracked independently in junction table

#### Failure Notification
- Failed imports reported via **health system warning only**
- Appears in health check page/API
- No push notifications for import failures

### Stalled Downloads
- **Ignore paused/stalled downloads** until resumed or completed
- Do not attempt to import partial files
- Wait for download client to report completion

### Source File Cleanup

After successful import:
- **Configurable per download client**
- Options: Leave for seeding, delete after import, delete after seed ratio met
- Different clients can have different cleanup behaviors

### Duplicate Prevention

- **Debounce by file path**: Ignore repeated filesystem events for same path within short window
- Database tracks imported files to prevent re-importing
- Schema supports multiple file versions per episode/movie

### Manual Import

#### Supported Sources
- Can import from **any accessible path** via manual import UI
- Not limited to download client folders
- Useful for external drives, network shares, existing media

#### Manual Import Flow
1. User browses to folder/file
2. System parses filenames and suggests matches
3. User confirms or corrects matches
4. Import proceeds with confirmed matches

## TV Item Renaming

### Overview

TV item renaming provides configurable file and folder naming for imported episodes. Users can define naming patterns using tokens that get replaced with metadata values during import.

### Settings

#### Enable Renaming
- **Rename Episodes** (boolean, default: true)
  - When enabled, files are renamed according to the configured format patterns
  - When disabled, original filenames are preserved

#### Character Replacement

##### Illegal Characters
- **Replace Illegal Characters** (boolean, default: true)
  - When enabled: illegal filesystem characters are replaced with valid alternatives
  - When disabled: illegal characters are simply removed
  - Illegal characters: `\ / : * ? " < > |`

##### Colon Replacement
How to handle colons (`:`) in titles, which are illegal on Windows filesystems:
- **Delete** - Remove colons entirely (`Title: Subtitle` → `Title Subtitle`)
- **Replace with Dash** - Replace with dash (`Title: Subtitle` → `Title- Subtitle`)
- **Replace with Space Dash** - Replace with space-dash (`Title: Subtitle` → `Title - Subtitle`)
- **Replace with Space Dash Space** - Replace with space-dash-space (`Title: Subtitle` → `Title - Subtitle`)
- **Smart Replace** (default) - Contextually choose dash or space-dash based on surrounding characters
- **Custom** - User-defined replacement character (e.g., Unicode colon alternative)

### Naming Format Patterns

#### Episode Format Types

Three distinct format patterns for different show types:

1. **Standard Episode Format**
   - For regular episodic series
   - Default: `{Series Title} - S{season:00}E{episode:00} - {Quality Title} {MediaInfo VideoDynamicRangeType}`
   - Example: `The Series Title - S01E01 - WEBDL-1080p`

2. **Daily Episode Format**
   - For daily/date-based shows (talk shows, news programs)
   - Default: `{Series Title} - {Air-Date} - {Episode Title} {Quality Full}`
   - Example: `The Daily Show - 2024-03-20 - Episode Title WEBDL-1080p Proper`

3. **Anime Episode Format**
   - For anime series (supports absolute episode numbering)
   - Default: `{Series Title} - S{season:00}E{episode:00} - {Episode Title} {Quality Full}`
   - Example: `Anime Series - S01E01 - Episode Title (1) WEBDL-1080p v2`

#### Folder Format Types

1. **Series Folder Format**
   - Root folder name for the series
   - Default: `{Series Title}`
   - Example: `The Series Title`

2. **Season Folder Format**
   - Subfolder for each season
   - Default: `Season {season}`
   - Example: `Season 1`

3. **Specials Folder Format** (advanced)
   - Subfolder for specials (Season 0)
   - Default: `Specials`

#### Multi-Episode Style

How to format filenames containing multiple episodes:

| Style | Format | Example |
|-------|--------|---------|
| **Extend** (default) | Sequential episode range | `S01E01-02-03` |
| **Duplicate** | Repeat full identifiers | `S01E01.S01E02` |
| **Repeat** | Concatenate episode numbers | `S01E01E02E03` |
| **Scene** | Scene naming convention | `S01E01-E02-E03` |
| **Range** | Compressed range | `S01E01-03` |
| **Prefixed Range** | Range with E prefix | `S01E01-E03` |

### Format Tokens

Tokens are placeholders wrapped in curly braces that get replaced with actual values during renaming.

#### Series Tokens

| Token | Description | Example Output |
|-------|-------------|----------------|
| `{Series Title}` | Full series title | `The Series Title's!` |
| `{Series TitleYear}` | Title with year | `The Series Title's! (2024)` |
| `{Series CleanTitle}` | Title without special chars | `The Series Titles` |
| `{Series CleanTitleYear}` | Clean title with year | `The Series Titles 2024` |

#### Season Tokens

| Token | Description | Example Output |
|-------|-------------|----------------|
| `{season:0}` | Season number (no padding) | `1` |
| `{season:00}` | Season number (2-digit pad) | `01` |

#### Episode Tokens

| Token | Description | Example Output |
|-------|-------------|----------------|
| `{episode:0}` | Episode number (no padding) | `1` |
| `{episode:00}` | Episode number (2-digit pad) | `01` |

#### Air Date Tokens

| Token | Description | Example Output |
|-------|-------------|----------------|
| `{Air-Date}` | Air date with dashes | `2024-03-20` |
| `{Air Date}` | Air date with spaces | `2024 03 20` |

#### Episode Title Tokens

| Token | Description | Example Output |
|-------|-------------|----------------|
| `{Episode Title}` | Full episode title | `Episode's Title` |
| `{Episode CleanTitle}` | Title without special chars | `Episodes Title` |

#### Quality Tokens

| Token | Description | Example Output |
|-------|-------------|----------------|
| `{Quality Full}` | Quality with revision info | `WEBDL-1080p Proper` |
| `{Quality Title}` | Quality without revision | `WEBDL-1080p` |

#### Media Info Tokens

| Token | Description | Example Output |
|-------|-------------|----------------|
| `{MediaInfo Simple}` | Basic video/audio codec | `x264 DTS` |
| `{MediaInfo Full}` | Full codec with languages | `x264 DTS [EN+DE]` |
| `{MediaInfo VideoCodec}` | Video codec | `x264`, `x265`, `AV1` |
| `{MediaInfo VideoBitDepth}` | Video bit depth | `8`, `10` |
| `{MediaInfo VideoDynamicRange}` | HDR format | `HDR` |
| `{MediaInfo VideoDynamicRangeType}` | Specific HDR type | `DV HDR10`, `HDR10+` |
| `{MediaInfo AudioCodec}` | Audio codec | `DTS`, `AAC`, `TrueHD Atmos` |
| `{MediaInfo AudioChannels}` | Audio channel layout | `5.1`, `7.1` |
| `{MediaInfo AudioLanguages}` | Audio languages | `[EN+DE]` |
| `{MediaInfo SubtitleLanguages}` | Subtitle languages | `[EN]` |

#### Other Tokens

| Token | Description | Example Output |
|-------|-------------|----------------|
| `{Release Group}` | Release group name | `SPARKS`, `NTb` |
| `{Custom Formats}` | Applied custom format tags | `iNTERNAL` |
| `{Custom Format:Name}` | Specific custom format | `AMZN` |
| `{Original Title}` | Original release title | `The.Series.S01E01.1080p.WEB-DL` |
| `{Original Filename}` | Original filename | `the.series.s01e01.1080p.web-dl` |

### Token Modifiers

#### Separator Control
Tokens support separator prefixes that control word separation:
- `{Series Title}` - Space separator (default): `The Series Title`
- `{Series.Title}` - Period separator: `The.Series.Title`
- `{Series-Title}` - Dash separator: `The-Series-Title`
- `{Series_Title}` - Underscore separator: `The_Series_Title`

#### Case Control
Global case transformation options:
- **Default Case** - Preserve original casing
- **Uppercase** - Convert to uppercase
- **Lowercase** - Convert to lowercase
- **Title Case** - Capitalize first letter of each word

#### Truncation
Limit token output length to prevent filesystem path length issues:
- `{Episode Title:30}` - Truncate to 30 characters from the end (with ellipsis)
- `{Episode Title:-30}` - Truncate to 30 characters from the beginning
- Episode titles are automatically truncated if they exceed filesystem limitations

#### Language Filtering (MediaInfo tokens)
Filter language tokens to include/exclude specific languages:
- `{MediaInfo Full:EN+DE}` - Include only English and German
- `{MediaInfo AudioLanguages:-DE}` - Exclude German
- `{MediaInfo SubtitleLanguages:EN+}` - Require English, show others contextually

#### Multi-Track Language Format
When files contain multiple audio or subtitle tracks, language tokens format is user-configurable:
- Separator character (default: `+`)
- Bracket style (default: `[EN+JA]`)
- Example configurations: `[EN|JA|DE]`, `EN-JA-DE`, `(English, Japanese)`

### Anime-Specific Tokens

For series detected as anime type, additional tokens are available:

| Token | Description | Example Output |
|-------|-------------|----------------|
| `{absolute:0}` | Absolute episode number (no padding) | `365` |
| `{absolute:00}` | Absolute episode number (2-digit pad) | `365` |
| `{absolute:000}` | Absolute episode number (3-digit pad) | `365` |
| `{version}` | Release version number | `v2` |

### Revision Token

Separate from quality, revision information has its own token:

| Token | Description | Example Output |
|-------|-------------|----------------|
| `{Revision}` | Proper/Repack/version indicator | `Proper`, `Repack`, `v2` |

This allows patterns like `{Quality Title} {Revision}` for explicit control.

---

## Behavioral Requirements

### Settings Scope
- Renaming settings are **global only** - one pattern applies to all series
- No per-series overrides for naming patterns

### Episode Format Type Selection
- **Auto-detection** from TVDB/TMDB metadata is the default behavior
- Users can **manually override** the format type per series (Standard/Daily/Anime)
- Metadata-based detection takes precedence when available

### Empty Token Handling
When a token has no available data (e.g., no release group detected, no episode title):
- **Smart cleanup**: Remove the token AND its surrounding separators
- Example: `{Series Title} - {Release Group}` becomes `Series Title` (not `Series Title -`)
- Prevents orphaned separators and double-spaces in filenames

### Multi-Episode Title Handling
For files containing multiple episodes:
- **Concatenate all episode titles** with a separator
- Example: `Episode 1 Title + Episode 2 Title + Episode 3 Title`
- If concatenated title exceeds path limits, import fails (no auto-truncation)

### Daily Show Episode Title Fallback
When a daily show episode has no title in metadata:
- Use the **formatted air date** as the episode title
- Example: `March 20, 2024` or `2024-03-20` depending on format

---

## Import Pipeline

### Rename Timing
Renaming occurs in this sequence:
1. **MediaInfo extraction** (blocking) - Extract codec, resolution, HDR info from file
2. **Compute final filename** - Resolve all tokens with extracted data
3. **Copy/Hardlink with final name** - File arrives in library with correct name

### MediaInfo Extraction Failure
If MediaInfo extraction fails (corrupted file, unsupported format):
- **Fall back to parsed release info** from the filename
- Use resolution, codec info parsed from release name instead
- Import proceeds with best-available information

### Hardlink/Symlink Behavior
When using hardlinks for seeding torrent support:
- **Original file** in download folder: Keeps original release name (continues seeding)
- **Library copy** (hardlink): Gets the renamed filename
- Only the library copy is renamed; download folder remains untouched

### File Lock Handling
If rename fails due to file lock (common with seeding files):
- This scenario is avoided by the hardlink approach above
- The library hardlink is created with the new name; original is never renamed

---

## Validation and Errors

### Format Pattern Validation
- **Strict validation** when saving format patterns
- Invalid tokens are rejected immediately
- Syntax errors prevent saving
- Live preview in UI shows validation errors in real-time

### Path Length Limits
- Windows MAX_PATH (260 characters) is enforced
- If computed path exceeds limit: **Fail the import with error**
- No automatic truncation of tokens
- Health system reports path length failures

### File Collisions
When destination file already exists:
- **Always overwrite** the existing file
- Upgrade decisions are handled upstream (quality comparison, etc.)
- Every completed download is treated as an intentional upgrade

### Clean Title Behavior
The `CleanTitle` token variants remove only **filesystem-illegal characters**:
- Characters removed: `\ / : * ? " < > |`
- All other characters (including Unicode, accents, etc.) are preserved

---

## File Operations

### File Extension Handling
- **Preserve original extension** exactly as-is
- No normalization (`.m4v` stays `.m4v`, not converted to `.mp4`)

### Extra Files (Subtitles, NFO, etc.)
- **Preserve original names** for extra files
- Only the main video file is renamed
- Subtitles, NFO files, images keep their original filenames

### Upgraded File Handling
When a new file replaces an existing episode (upgrade):
- **Delete old file immediately** upon successful import
- No recycling bin, no `.old` suffix
- Old file is removed as soon as new file is in place

### Folder Creation Permissions
When creating new series/season folders:
- **Inherit permissions from parent folder**
- No explicit chmod configuration needed

### Folder Renaming on Pattern Change
When series folder format changes or series is renamed:
- **Automatically move entire series structure** to new location
- All season folders and episode files are relocated
- No orphaned files left behind

---

## Season Pack Handling

When importing a season pack (full season in one release):
- **Split into individual episode files**
- Each episode is extracted and renamed according to format pattern
- Individual episodes placed in appropriate season folder

---

## Specials Detection

Episodes are identified as Specials (Season 0) using:
1. **Metadata from TVDB/TMDB** (primary source)
2. **Filename pattern parsing** (fallback) - detects `Special`, `SP01`, `S00E01`
- Metadata takes precedence when available

---

## Mass Rename Feature

When naming pattern settings change:
- **Prompt user** with confirmation dialog
- Ask: "Apply new naming pattern to existing library files?"
- User explicitly chooses to rename existing files or not
- Only affects files already in library, not download folder

---

## UI Requirements

### Live Preview
- Settings page shows **real-time preview** of resulting filename
- Preview updates as user types/modifies pattern
- Uses sample data to demonstrate token resolution
- Shows validation errors inline

### Debug/Test Mode
- Provide **detailed token breakdown** for debugging
- Show each token's resolved value
- Display the step-by-step name computation
- Useful for troubleshooting complex patterns

---

## History and Logging

### Rename History
- Rename operations are logged as **part of import history**
- No separate rename-specific log
- Import history shows before/after filenames

### Original Name Preservation
- Original release name is **stored in database only**
- No sidecar files or filesystem artifacts
- Queryable for reference but not exposed in filename

---

## Release Group Detection

- Parse release group **from filename only**
- No indexer data or MediaInfo tag fallback
- Standard scene naming patterns used for extraction

---

## Default Patterns

Default naming patterns **match Sonarr defaults** for easier migration:
- Standard: `{Series Title} - S{season:00}E{episode:00} - {Quality Title} {MediaInfo VideoDynamicRangeType}`
- Daily: `{Series Title} - {Air-Date} - {Episode Title} {Quality Full}`
- Anime: `{Series Title} - S{season:00}E{episode:00} - {Episode Title} {Quality Full}`
- Series Folder: `{Series Title}`
- Season Folder: `Season {season}`

---

## Scope Limitations

### Media Types
- **Video files only** are renamed
- Audio files, images, and other media types are not processed by renaming system

### Conditional Tokens
- **No conditional logic** in format patterns
- Keep patterns simple and predictable
- Empty token cleanup handles most conditional needs

### Token Length Limits
- **No per-token length limits**
- Only total path length is enforced
- Individual tokens can be any length

