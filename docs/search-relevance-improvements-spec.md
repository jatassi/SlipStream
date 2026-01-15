# Search Relevance Improvements Specification

## Pre-Implementation: Codebase Exploration

Before beginning implementation, explore these areas to understand the current search architecture:

### Priority 1: Core Search Flow (Read These First)

| File | Why | Key Functions/Structures |
|------|-----|-------------------------|
| `internal/indexer/search/service.go` | Entry point for all searches | `SearchTorrents()`, `SearchTV()`, `SearchMovies()` |
| `internal/indexer/search/aggregator.go` | Where filtering will be added | `aggregateTorrentResults()`, `deduplicateTorrents()`, `enrichTorrentsWithQuality()` |
| `internal/indexer/types/types.go` | Search criteria structure | `SearchCriteria`, `TorrentInfo`, `ReleaseInfo` |
| `internal/indexer/cardigann/client.go` | Query construction | `buildSearchQuery()`, `SearchTorrents()` |

### Priority 2: Title Parsing (Understand Before Writing Title Matching)

| File | Why | Key Functions/Structures |
|------|-----|-------------------------|
| `internal/library/scanner/parser.go` | Extracts title, season, episode, year from filenames | `ParseFilename()`, `ParsedMedia` struct, regex patterns |
| `internal/library/scanner/parser_test.go` | Shows expected parsing behavior | Test cases for various filename formats |

### Priority 3: Supporting Context

| File | Why | Key Functions/Structures |
|------|-----|-------------------------|
| `internal/indexer/search/handlers.go` | HTTP handlers that call search service | `SearchTV()`, `SearchMovies()`, `toCriteria()` |
| `internal/indexer/cardigann/search.go` | How queries are sent to indexers | `Search()`, `buildSearchURL()`, `SearchQuery` struct |
| `internal/indexer/scoring/scorer.go` | Current scoring logic (for reference) | `ScoreTorrent()`, `calculateMatchScore()` |

### Key Questions to Answer During Exploration

1. **How does SearchCriteria flow through the system?**
   - Trace from HTTP handler → service → cardigann client → aggregator

2. **What fields are available in SearchCriteria for filtering?**
   - Query, Type, Season, Episode, Year, TvdbID, ImdbID, Categories

3. **What does ParseFilename() return?**
   - ParsedMedia with Title, Season, Episode, Year, IsTV, Quality, Source, etc.

4. **Where is enrichTorrentsWithQuality() called?**
   - This already calls ParseFilename() - can reuse parsed data for filtering

5. **How are results currently sorted/returned?**
   - Deduplication → Enrichment → Sorting → Return

### Sample Search to Trace

To understand the full flow, trace a TV search request:
```
POST /api/v1/search/tv
{
  "query": "Dark",
  "season": 1,
  "episode": 1,
  "qualityProfileId": 1
}
```

Follow the request through:
1. `handlers.go:SearchTV()` → converts to SearchCriteria
2. `service.go:SearchTorrents()` → dispatches to indexers
3. `cardigann/client.go:SearchTorrents()` → builds query, calls indexer
4. `aggregator.go:aggregateTorrentResults()` → combines results ← **filtering goes here**
5. `scoring/scorer.go:ScoreTorrents()` → ranks results

---

## Problem Statement

When searching for media with short or common titles (e.g., "Dark", "It", "Her"), SlipStream returns many irrelevant results because:

1. **Query Construction**: Search queries only contain the title keyword, not season/episode (TV) or year (movies)
2. **No Filtering**: All indexer results pass through without validation against search criteria
3. **Scoring Limitations**: Scoring ranks results but doesn't exclude mismatches

### Example: TV Search for "Dark" S01E01

**Current behavior**: Returns "The Dark Knight", "Zero Dark Thirty", "Dark Phoenix", etc.

**Expected behavior**: Returns only releases matching the TV series "Dark" season 1 episode 1 (or S01 season packs)

### Example: Movie Search for "It" (2017)

**Current behavior**: Returns "It Follows", "It Chapter Two", various TV shows containing "it"

**Expected behavior**: Returns only releases matching the movie "It" from 2017 (±1 year tolerance)

---

## Implementation Tasks

### Phase 1: Title Matching Utility

- [x] **1.1** Create new file `internal/indexer/search/title_match.go`

- [x] **1.2** Implement `NormalizeTitle(title string) string`
  - Convert to lowercase
  - Remove special characters (keep alphanumeric and spaces)
  - Collapse multiple spaces to single space
  - Trim leading/trailing whitespace
  - Example: "The.Dark.Knight.2008" → "the dark knight 2008"

- [x] **1.3** Implement `TitlesMatch(parsedTitle, searchQuery string) bool`
  - Normalize both titles
  - Return true if normalized titles are equal
  - This is strict matching - "Dark" matches "Dark" but NOT "Dark Knight"

- [x] **1.4** Implement `CalculateTitleSimilarity(title1, title2 string) float64`
  - Normalize both titles
  - Split into tokens (words)
  - Calculate Jaccard similarity: matching tokens / max token count
  - Return value 0.0 to 1.0
  - Used for debugging/logging, not filtering decisions

- [x] **1.5** Write unit tests in `internal/indexer/search/title_match_test.go`
  - Test cases for NormalizeTitle:
    - "The.Dark.Knight.2008" → "the dark knight 2008"
    - "Dark.S01E01.1080p.WEB-DL" → "dark s01e01 1080p web dl"
    - "It (2017)" → "it 2017"
    - "  Multiple   Spaces  " → "multiple spaces"
  - Test cases for TitlesMatch:
    - ("Dark", "Dark") → true
    - ("Dark", "dark") → true
    - ("Dark", "The Dark Knight") → false
    - ("Dark", "Dark Matter") → false
    - ("It", "It") → true
    - ("It", "It Follows") → false
  - Test cases for CalculateTitleSimilarity:
    - ("Dark", "Dark") → 1.0
    - ("Dark", "The Dark Knight") → 0.33
    - ("The Dark Knight", "The Dark Knight Rises") → 0.75

---

### Phase 2: Post-Search Filtering

- [x] **2.1** Add `FilterByCriteria` function to `internal/indexer/search/aggregator.go`

- [x] **2.2** Implement TV search filtering logic:
  ```
  For each result:
    1. Parse title using scanner.ParseFilename()
    2. Check IsTV == true (reject movies)
    3. Check title matches search query (using TitlesMatch)
    4. If criteria.Season > 0: check parsed.Season == criteria.Season
    5. If criteria.Episode > 0:
       - Accept if parsed.Episode == criteria.Episode (exact match)
       - Accept if parsed.Episode == 0 (season pack)
       - Reject otherwise (wrong episode)
  ```

- [x] **2.3** Implement movie search filtering logic:
  ```
  For each result:
    1. Parse title using scanner.ParseFilename()
    2. Check IsTV == false (reject TV content)
    3. Check title matches search query (using TitlesMatch)
    4. If criteria.Year > 0 and parsed.Year > 0:
       - Accept if abs(parsed.Year - criteria.Year) <= 1
       - Reject otherwise (wrong year)
    5. If parsed.Year == 0: accept (year not in release name)
  ```

- [x] **2.4** Integrate filtering into `aggregateTorrentResults`:
  ```go
  // After deduplication and enrichment, before sorting:
  if criteria != nil {
      deduplicated = FilterByCriteria(deduplicated, *criteria)
  }
  ```

- [x] **2.5** Update `aggregateTorrentResults` signature to accept criteria:
  - Current: `func (s *Service) aggregateTorrentResults(results <-chan searchTaskResult) *TorrentSearchResult`
  - New: `func (s *Service) aggregateTorrentResults(results <-chan searchTaskResult, criteria *types.SearchCriteria) *TorrentSearchResult`

- [x] **2.6** Update all callers of `aggregateTorrentResults` to pass criteria

- [x] **2.7** Write unit tests in `internal/indexer/search/aggregator_test.go`
  - Test TV filtering:
    - "Dark.S01E01.1080p" with criteria (Dark, S01E01) → PASS
    - "Dark.S01.Complete" with criteria (Dark, S01E01) → PASS (season pack)
    - "Dark.S01E02.1080p" with criteria (Dark, S01E01) → REJECT (wrong episode)
    - "Dark.S02E01.1080p" with criteria (Dark, S01E01) → REJECT (wrong season)
    - "The.Dark.Knight.2008" with criteria (Dark, S01E01) → REJECT (movie + wrong title)
    - "Dark.Matter.S01E01" with criteria (Dark, S01E01) → REJECT (wrong title)
  - Test movie filtering:
    - "It.2017.1080p.BluRay" with criteria (It, 2017) → PASS
    - "It.2018.1080p.BluRay" with criteria (It, 2017) → PASS (±1 year)
    - "It.2019.1080p.BluRay" with criteria (It, 2017) → REJECT (>1 year diff)
    - "It.Follows.2014" with criteria (It, 2017) → REJECT (wrong title)
    - "It.S01E01" with criteria (It, 2017) → REJECT (TV content)

---

### Phase 3: Query Construction Enhancement

- [x] **3.1** Modify `buildSearchQuery` in `internal/indexer/cardigann/client.go`

- [x] **3.2** Enhance TV search query keywords:
  ```go
  if criteria.Type == "tvsearch" && criteria.Season > 0 {
      if criteria.Episode > 0 {
          // Specific episode: "Dark S01E01"
          query.Query = fmt.Sprintf("%s S%02dE%02d", criteria.Query, criteria.Season, criteria.Episode)
      } else {
          // Season search: "Dark S01"
          query.Query = fmt.Sprintf("%s S%02d", criteria.Query, criteria.Season)
      }
  }
  ```

- [x] **3.3** Enhance movie search query keywords:
  ```go
  if criteria.Type == "movie" && criteria.Year > 0 {
      // Movie with year: "It 2017"
      query.Query = fmt.Sprintf("%s %d", criteria.Query, criteria.Year)
  }
  ```

- [x] **3.4** Preserve original query for filtering:
  - Store original title in SearchQuery struct for use in filtering
  - Add `OriginalQuery string` field to SearchQuery if needed
  - Note: Original query is preserved in SearchCriteria which is passed to filtering

- [x] **3.5** Write unit tests for query construction:
  - TV S01E01: ("Dark", tvsearch, S01, E01) → "Dark S01E01"
  - TV S01: ("Dark", tvsearch, S01, 0) → "Dark S01"
  - TV no season: ("Dark", tvsearch, 0, 0) → "Dark"
  - Movie with year: ("It", movie, 2017) → "It 2017"
  - Movie no year: ("It", movie, 0) → "It"
  - General search: ("Dark Knight", search) → "Dark Knight"

---

### Phase 4: Integration and Testing

- [x] **4.1** Update SearchCriteria type if needed (`internal/indexer/types/types.go`)
  - Ensure all required fields are present for filtering
  - Note: SearchCriteria already has all required fields (Query, Type, Season, Episode, Year)

- [ ] **4.2** End-to-end integration testing:
  - Test with mock indexer returning mixed results
  - Verify filtering removes irrelevant results
  - Verify scoring ranks remaining results correctly

- [ ] **4.3** Manual testing scenarios:
  - [ ] TV: Search "Dark" S01E01 - verify no movies, no wrong episodes
  - [ ] TV: Search "Dark" S01 - verify season packs and individual episodes pass
  - [ ] TV: Search "Game of Thrones" S01E01 - verify multi-word title matching
  - [ ] Movie: Search "It" 2017 - verify no TV, no wrong years
  - [ ] Movie: Search "The Dark Knight" 2008 - verify multi-word title matching
  - [ ] Movie: Search "Dune" 2021 - verify 2020/2021/2022 releases pass (±1 year)

- [ ] **4.4** Edge case testing:
  - [ ] Empty search results after filtering (should return empty, not error)
  - [ ] Title with special characters: "Spider-Man"
  - [ ] Title with numbers: "2001: A Space Odyssey"
  - [ ] Title with colon: "Star Wars: The Force Awakens"
  - [ ] Very short title: "It", "Us", "Her"
  - [ ] Release with no year in filename
  - [ ] Release with no season/episode in filename

---

## File Summary

| File | Action | Description |
|------|--------|-------------|
| `internal/indexer/search/title_match.go` | CREATE | Title normalization and matching utilities |
| `internal/indexer/search/title_match_test.go` | CREATE | Unit tests for title matching |
| `internal/indexer/search/aggregator.go` | MODIFY | Add FilterByCriteria function |
| `internal/indexer/search/aggregator_test.go` | MODIFY | Add filtering unit tests |
| `internal/indexer/cardigann/client.go` | MODIFY | Enhance query keyword construction |
| `internal/indexer/cardigann/client_test.go` | MODIFY | Add query construction tests |

---

## Design Decisions

### 1. Strict Title Matching
**Decision**: Titles must match exactly after normalization.

**Rationale**: "Dark" should only match "Dark", not "Dark Knight" or "Dark Matter". Users searching for a specific title don't want similar titles.

**Trade-off**: May miss some valid results with slight title variations. Can be relaxed later if needed.

### 2. Season Packs Pass Through
**Decision**: When searching for a specific episode (S01E01), season packs (S01) are accepted.

**Rationale**: Season packs contain the requested episode and are often preferred by users. Filtering them out would be overly restrictive.

### 3. Year Tolerance ±1
**Decision**: Movie year matching allows off-by-one difference.

**Rationale**: Release dates vary by region. A movie listed as "2017" in TMDB might have releases tagged "2016" or "2018" in some regions.

### 4. Content Type Enforcement
**Decision**: TV searches filter out movies, movie searches filter out TV.

**Rationale**: This is fundamental - users searching for a TV episode never want movie results and vice versa.

### 5. Filtering vs Scoring Separation
**Decision**: Filtering removes invalid results; scoring only ranks valid ones.

**Rationale**: Keeps concerns separate. Scoring penalties for filtered-out content would be redundant and never execute.

---

## Future Considerations

### Potential Enhancements (Not in Scope)
- [ ] Fuzzy title matching with configurable threshold
- [ ] TVDB/IMDB ID matching when available (more reliable than title)
- [ ] User preference for strict vs relaxed filtering
- [ ] "Show filtered results" toggle in UI
- [ ] Alternative title matching (e.g., "Doctor Who" vs "Dr. Who")

---

## Acceptance Criteria

1. Searching for "Dark" S01E01 returns ONLY:
   - Releases with title "Dark" (not "Dark Knight", "Dark Matter", etc.)
   - TV content (not movies)
   - Season 1 content (episodes or season packs)
   - Episode 1 or season packs (not other episodes)

2. Searching for "It" (2017) returns ONLY:
   - Releases with title "It" (not "It Follows", "It Chapter Two", etc.)
   - Movie content (not TV)
   - Year 2016, 2017, or 2018 (±1 tolerance)

3. Query keywords sent to indexers include season/episode or year for better server-side filtering

4. All existing tests continue to pass

5. No performance regression in search response times
