# Search System

## Overview

The search system aggregates searches across multiple indexers, handling:
- Query normalization and distribution
- Parallel execution across indexers
- Result aggregation and deduplication
- Category filtering and mapping
- Rate limit enforcement

## Search Flow

```
┌──────────────────────────────────────────────────────────────────┐
│                      Search Request                               │
│   Query: "The Matrix 1999", Type: movie, Categories: [2000]      │
└─────────────────────────────┬────────────────────────────────────┘
                              │
                              ▼
┌──────────────────────────────────────────────────────────────────┐
│                   Query Parameter Extraction                      │
│   Parse: title="The Matrix", year=1999, imdbid=...               │
└─────────────────────────────┬────────────────────────────────────┘
                              │
                              ▼
┌──────────────────────────────────────────────────────────────────┐
│                   Create Search Criteria                          │
│   MovieSearchCriteria { SearchTerm, Year, Categories, ... }      │
└─────────────────────────────┬────────────────────────────────────┘
                              │
                              ▼
┌──────────────────────────────────────────────────────────────────┐
│                     Filter Indexers                               │
│   - Enabled indexers only                                        │
│   - Support requested search type                                │
│   - Support requested categories                                 │
│   - Not currently rate-limited                                   │
└─────────────────────────────┬────────────────────────────────────┘
                              │
         ┌────────────────────┼────────────────────┐
         ▼                    ▼                    ▼
    ┌─────────┐          ┌─────────┐          ┌─────────┐
    │Indexer 1│          │Indexer 2│          │Indexer N│
    │ Search  │          │ Search  │          │ Search  │
    └────┬────┘          └────┬────┘          └────┬────┘
         │                    │                    │
         ▼                    ▼                    ▼
    ┌─────────┐          ┌─────────┐          ┌─────────┐
    │Results 1│          │Results 2│          │Results N│
    └────┬────┘          └────┬────┘          └────┬────┘
         │                    │                    │
         └────────────────────┼────────────────────┘
                              │
                              ▼
┌──────────────────────────────────────────────────────────────────┐
│                   Aggregate & Filter Results                      │
│   - Apply category filter                                        │
│   - Apply age filter (MinAge/MaxAge)                            │
│   - Apply size filter (MinSize/MaxSize)                         │
└─────────────────────────────┬────────────────────────────────────┘
                              │
                              ▼
┌──────────────────────────────────────────────────────────────────┐
│                      Deduplicate                                  │
│   Group by GUID, keep highest priority indexer                   │
└─────────────────────────────┬────────────────────────────────────┘
                              │
                              ▼
┌──────────────────────────────────────────────────────────────────┐
│                    Return Results                                 │
│   List<ReleaseInfo> with metadata                                │
└──────────────────────────────────────────────────────────────────┘
```

## Search Criteria Types

### Base Criteria

All search criteria extend `SearchCriteriaBase`:

```
SearchCriteriaBase
├── SearchTerm: string (raw query)
├── SanitizedSearchTerm: string (cleaned query)
├── Categories: int[] (Newznab category IDs)
├── Limit: int (max results)
├── Offset: int (pagination)
├── Source: string (request source)
├── Host: string (request host)
├── IndexerIds: int[] (specific indexers)
│
├── Filters
│   ├── MinAge: int (days, older than)
│   ├── MaxAge: int (days, newer than)
│   ├── MinSize: long (bytes)
│   └── MaxSize: long (bytes)
│
└── Methods
    ├── IsRssSearch: bool (empty query)
    ├── IsIdSearch: bool (has external ID)
    └── SanitizedSearchTerm: string
```

### TV Search Criteria

```
TvSearchCriteria extends SearchCriteriaBase
├── External IDs
│   ├── ImdbId: string (tt1234567)
│   ├── TvdbId: int
│   ├── TvMazeId: int
│   ├── TraktId: int
│   ├── TmdbId: int
│   ├── TvRageId: int
│   └── DoubanId: int
│
├── Episode Info
│   ├── Season: int
│   ├── Episode: string (can be range: "1-5")
│   └── Year: int
│
└── Metadata
    └── Genre: string
```

### Movie Search Criteria

```
MovieSearchCriteria extends SearchCriteriaBase
├── External IDs
│   ├── ImdbId: string
│   ├── TmdbId: int
│   ├── TraktId: int
│   └── DoubanId: int
│
└── Metadata
    ├── Year: int
    └── Genre: string
```

### Music Search Criteria

```
MusicSearchCriteria extends SearchCriteriaBase
├── Artist: string
├── Album: string
├── Track: string
├── Label: string
├── Year: int
└── Genre: string
```

### Book Search Criteria

```
BookSearchCriteria extends SearchCriteriaBase
├── Author: string
├── Title: string
├── Publisher: string
├── Year: int
└── Genre: string
```

### Basic Search Criteria

```
BasicSearchCriteria extends SearchCriteriaBase
(No additional fields - general purpose search)
```

## Query Parameter Extraction

For Newznab-compatible API requests, parameters are extracted from query strings:

### TV Search Parameters

```
Input: "Breaking Bad {tvdbid:81189} {season:1} {episode:2}"

Extraction:
  title = "Breaking Bad"
  tvdbid = 81189
  season = 1
  episode = 2

Patterns:
  {imdbid:tt1234567} or {imdbid:1234567}
  {tvdbid:12345}
  {season:1}
  {episode:2} or {ep:2}
  {year:2020}
  {genre:drama}
```

### Movie Search Parameters

```
Input: "The Matrix {imdbid:tt0133093} {year:1999}"

Extraction:
  title = "The Matrix"
  imdbid = tt0133093
  year = 1999

Patterns:
  {imdbid:tt1234567}
  {tmdbid:12345}
  {year:1999}
  {genre:action}
```

### Music Search Parameters

```
Input: "{artist:Pink Floyd} {album:The Wall}"

Extraction:
  artist = "Pink Floyd"
  album = "The Wall"

Patterns:
  {artist:name}
  {album:name}
  {track:name}
  {label:name}
  {year:1979}
  {genre:rock}
```

### Book Search Parameters

```
Input: "{author:Stephen King} {title:The Shining}"

Extraction:
  author = "Stephen King"
  title = "The Shining"

Patterns:
  {author:name}
  {title:name}
  {publisher:name}
  {year:1977}
```

## Indexer Filtering

### Selection Algorithm

```
FUNCTION GetSearchableIndexers(criteria):
    indexers = IndexerFactory.Enabled()

    // Filter by specific IDs if provided
    IF criteria.IndexerIds.NotEmpty:
        // Special IDs
        IF criteria.IndexerIds.Contains(-1):  // All Usenet
            indexers = indexers.Where(i => i.Protocol == Usenet)
        ELSE IF criteria.IndexerIds.Contains(-2):  // All Torrent
            indexers = indexers.Where(i => i.Protocol == Torrent)
        ELSE:
            indexers = indexers.Where(i => criteria.IndexerIds.Contains(i.Id))

    // Filter by category support
    IF criteria.Categories.NotEmpty:
        indexers = indexers.Where(i =>
            i.Capabilities.SupportedCategories(criteria.Categories).Any()
        )

    // Filter by search type support
    searchType = DetermineSearchType(criteria)
    indexers = indexers.Where(i => SupportsSearchType(i, searchType))

    // Filter out rate-limited indexers
    indexers = indexers.Where(i => !IsRateLimited(i))

    RETURN indexers
```

### Search Type Support Check

```
FUNCTION SupportsSearchType(indexer, searchType):
    caps = indexer.Capabilities

    SWITCH searchType:
        CASE tv-search:
            RETURN caps.TvSearchParams.Contains(Q) OR caps.TvSearchParams.Any()
        CASE movie:
            RETURN caps.MovieSearchParams.Contains(Q) OR caps.MovieSearchParams.Any()
        CASE music:
            RETURN caps.MusicSearchParams.Contains(Q) OR caps.MusicSearchParams.Any()
        CASE book:
            RETURN caps.BookSearchParams.Contains(Q) OR caps.BookSearchParams.Any()
        DEFAULT:
            RETURN caps.SearchParams.Contains(Q)
```

## Parallel Search Execution

### Dispatch Process

```
FUNCTION DispatchSearch(criteria):
    indexers = GetSearchableIndexers(criteria)
    tasks = []

    FOR EACH indexer IN indexers:
        task = DispatchIndexerAsync(indexer, criteria)
        tasks.Add(task)

    // Wait for all searches to complete
    results = await Task.WhenAll(tasks)

    // Flatten and aggregate
    allReleases = results.SelectMany(r => r.Releases)

    RETURN allReleases

FUNCTION DispatchIndexerAsync(indexer, criteria):
    TRY:
        // Check rate limits
        IF IndexerLimitService.AtQueryLimit(indexer):
            RETURN EmptyResult()

        // Execute search
        result = await indexer.Fetch(criteria)

        // Record query in history
        PublishEvent(IndexerQueryEvent(indexer, criteria, result.Count))

        RETURN result

    CATCH exception:
        LogWarning("Search failed for {indexer}: {exception}")
        RecordFailure(indexer, exception)
        RETURN EmptyResult()
```

### Timeout Handling

```
FUNCTION FetchWithTimeout(indexer, criteria, timeout = 30000):
    cancellationToken = new CancellationToken(timeout)

    TRY:
        RETURN await indexer.Fetch(criteria).WithCancellation(cancellationToken)
    CATCH OperationCanceledException:
        LogWarning("Search timeout for {indexer}")
        RETURN EmptyResult()
```

## Result Processing

### Category Filtering

```
FUNCTION FilterByCategories(releases, requestedCategories):
    IF requestedCategories.IsEmpty:
        RETURN releases

    // Expand parent categories to include children
    expandedCategories = ExpandCategories(requestedCategories)

    RETURN releases.Where(r =>
        r.Categories.Any(c => expandedCategories.Contains(c.Id))
    )

FUNCTION ExpandCategories(categories):
    expanded = new List<int>()

    FOR EACH categoryId IN categories:
        expanded.Add(categoryId)

        // Add child categories
        IF IsParentCategory(categoryId):
            children = GetChildCategories(categoryId)
            expanded.AddRange(children)

    RETURN expanded.Distinct()
```

### Age Filtering

```
FUNCTION FilterByAge(releases, minAge, maxAge):
    now = DateTime.UtcNow

    RETURN releases.Where(r => {
        age = (now - r.PublishDate).Days

        IF minAge.HasValue AND age < minAge:
            RETURN false

        IF maxAge.HasValue AND age > maxAge:
            RETURN false

        RETURN true
    })
```

### Size Filtering

```
FUNCTION FilterBySize(releases, minSize, maxSize):
    RETURN releases.Where(r => {
        IF minSize.HasValue AND r.Size < minSize:
            RETURN false

        IF maxSize.HasValue AND r.Size > maxSize:
            RETURN false

        RETURN true
    })
```

## Deduplication

### Algorithm

```
FUNCTION DeDupeReleases(releases):
    // Group by unique identifier (GUID)
    groups = releases.GroupBy(r => r.Guid)

    // For each group, select release from highest priority indexer
    deduplicated = groups.Select(g =>
        g.OrderBy(r => r.IndexerPriority).First()
    )

    RETURN deduplicated.ToList()
```

### Priority Determination

Lower `IndexerPriority` value = higher priority

```
IndexerPriority (default: 25)
├── 1-10: High priority indexers
├── 11-25: Normal priority
└── 26-50: Low priority

During deduplication:
- Release from indexer with Priority=1 wins over Priority=25
- Ties broken by order in database (first indexer wins)
```

## Result Caching

### Cache Structure

```
Cache Key: "{IndexerId}_{Guid}"
Cache Duration: 30 minutes
Cache Purpose: Support download requests after search

FUNCTION CacheResults(releases):
    FOR EACH release IN releases:
        key = "{release.IndexerId}_{release.Guid}"
        Cache.Set(key, release, TimeSpan.FromMinutes(30))

FUNCTION GetCachedRelease(indexerId, guid):
    key = "{indexerId}_{guid}"
    RETURN Cache.Get<ReleaseInfo>(key)
```

## Newznab/Torznab API Response

### XML Response Format

```xml
<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0"
     xmlns:atom="http://www.w3.org/2005/Atom"
     xmlns:newznab="http://www.newznab.com/DTD/2010/feeds/attributes/"
     xmlns:torznab="http://torznab.com/schemas/2015/feed">
  <channel>
    <title>Prowlarr</title>
    <description>Prowlarr Search Results</description>

    <newznab:response offset="0" total="100"/>

    <item>
      <title>Release Title S01E01 1080p WEB-DL</title>
      <guid>unique-guid-12345</guid>
      <link>http://prowlarr/1/download?link=...</link>
      <comments>http://indexer.com/details/12345</comments>
      <pubDate>Mon, 01 Jan 2024 12:00:00 +0000</pubDate>
      <size>1500000000</size>
      <description>Release description</description>

      <enclosure url="http://prowlarr/1/download?link=..."
                 length="1500000000"
                 type="application/x-bittorrent"/>

      <!-- Standard attributes -->
      <newznab:attr name="category" value="5030"/>
      <newznab:attr name="size" value="1500000000"/>
      <newznab:attr name="files" value="5"/>
      <newznab:attr name="grabs" value="100"/>
      <newznab:attr name="poster" value="http://..."/>

      <!-- External IDs -->
      <newznab:attr name="imdb" value="1234567"/>
      <newznab:attr name="tvdbid" value="123456"/>
      <newznab:attr name="tmdbid" value="12345"/>

      <!-- Torrent-specific -->
      <torznab:attr name="seeders" value="50"/>
      <torznab:attr name="peers" value="75"/>
      <torznab:attr name="infohash" value="abc123..."/>
      <torznab:attr name="magneturl" value="magnet:?xt=..."/>
      <torznab:attr name="minimumratio" value="1.0"/>
      <torznab:attr name="minimumseedtime" value="86400"/>
      <torznab:attr name="downloadvolumefactor" value="0"/>
      <torznab:attr name="uploadvolumefactor" value="1"/>
    </item>

    <!-- More items... -->
  </channel>
</rss>
```

### Capabilities Response

```xml
<?xml version="1.0" encoding="UTF-8"?>
<caps>
  <server title="Prowlarr" />

  <limits max="100" default="50"/>

  <searching>
    <search available="yes" supportedParams="q"/>
    <tv-search available="yes"
               supportedParams="q,season,ep,imdbid,tvdbid"/>
    <movie-search available="yes"
                  supportedParams="q,imdbid,tmdbid"/>
    <music-search available="yes"
                  supportedParams="q,artist,album"/>
    <book-search available="yes"
                 supportedParams="q,author,title"/>
  </searching>

  <categories>
    <category id="2000" name="Movies">
      <subcat id="2030" name="Movies/SD"/>
      <subcat id="2040" name="Movies/HD"/>
      <subcat id="2045" name="Movies/UHD"/>
    </category>
    <category id="5000" name="TV">
      <subcat id="5030" name="TV/SD"/>
      <subcat id="5040" name="TV/HD"/>
    </category>
  </categories>
</caps>
```

## Search History

### History Record

```
History
├── Id: int
├── IndexerId: int
├── Date: DateTime
├── EventType: HistoryEventType
│   ├── IndexerQuery
│   ├── IndexerRss
│   ├── IndexerAuth
│   └── ReleaseGrabbed
├── Successful: bool
├── DownloadId: string (if grabbed)
└── Data: JSON
    ├── Query
    ├── Categories
    ├── ElapsedTime
    ├── Results
    └── Source
```

### Recording Search History

```
FUNCTION RecordSearchHistory(indexer, criteria, resultCount, elapsed):
    history = new History
    {
        IndexerId = indexer.Id,
        Date = DateTime.UtcNow,
        EventType = criteria.IsRssSearch ? IndexerRss : IndexerQuery,
        Successful = true,
        Data = {
            Query = criteria.SearchTerm,
            Categories = criteria.Categories,
            Results = resultCount,
            ElapsedTime = elapsed.TotalMilliseconds,
            Source = criteria.Source,
            Host = criteria.Host
        }
    }

    HistoryRepository.Insert(history)
```

## Error Handling

### Search Errors

```
TRY:
    results = indexer.Fetch(criteria)
CATCH IndexerAuthException:
    RecordAuthFailure(indexer)
    PublishEvent(IndexerAuthEvent(indexer))
CATCH RateLimitedException:
    RecordRateLimitHit(indexer)
    // Don't record as failure - temporary
CATCH WebException:
    RecordConnectionFailure(indexer)
CATCH Exception:
    RecordGeneralFailure(indexer, exception)
```

### Graceful Degradation

```
FUNCTION DispatchWithFallback(criteria):
    results = []
    failedIndexers = []

    FOR EACH indexer IN enabledIndexers:
        TRY:
            result = await DispatchIndexer(indexer, criteria)
            results.AddRange(result)
        CATCH:
            failedIndexers.Add(indexer)

    // Return partial results even if some indexers failed
    IF results.Any():
        RETURN results

    // If all failed, throw aggregate exception
    IF failedIndexers.Count == enabledIndexers.Count:
        THROW AggregateSearchException(failedIndexers)

    RETURN results
```
