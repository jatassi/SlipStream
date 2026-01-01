# Indexer System

## Overview

The indexer system manages connections to usenet and torrent indexers, handling:
- Authentication and session management
- Search request generation
- Response parsing
- Rate limiting and failure tracking

## Indexer Hierarchy

```
IIndexer (Interface)
    │
    ▼
IndexerBase<TSettings>
    │
    ▼
HttpIndexerBase<TSettings>
    │
    ├─────────────────────────┐
    ▼                         ▼
TorrentIndexerBase<T>    UsenetIndexerBase<T>
    │                         │
    ▼                         ▼
[Concrete Indexers]      [Concrete Indexers]
```

## Core Interface

```
IIndexer
├── Properties
│   ├── Name: string
│   ├── Protocol: DownloadProtocol (Usenet | Torrent)
│   ├── Privacy: IndexerPrivacy (Public | SemiPrivate | Private)
│   ├── IndexerUrls: string[]
│   ├── LegacyUrls: string[]
│   ├── Capabilities: IndexerCapabilities
│   ├── SupportsRss: bool
│   ├── SupportsSearch: bool
│   ├── SupportsRedirect: bool
│   └── SupportsPagination: bool
│
└── Methods
    ├── Fetch(criteria): IndexerPageableQueryResult
    │   ├── MovieSearchCriteria
    │   ├── MusicSearchCriteria
    │   ├── TvSearchCriteria
    │   ├── BookSearchCriteria
    │   └── BasicSearchCriteria
    ├── Download(link): byte[] | redirect
    └── GetCapabilities(): IndexerCapabilities
```

## Indexer Definition

Each indexer configuration is stored as an `IndexerDefinition`:

```
IndexerDefinition
├── Id: int (primary key)
├── Name: string
├── Implementation: string (class name)
├── Settings: JSON (serialized settings object)
├── ConfigContract: string (settings class name)
├── Enable: bool
├── Tags: int[]
│
├── Protocol: DownloadProtocol
├── Privacy: IndexerPrivacy
├── IndexerUrls: string[]
├── LegacyUrls: string[]
├── Language: string
├── Encoding: string
├── Description: string
│
├── SupportsRss: bool
├── SupportsSearch: bool
├── SupportsRedirect: bool
├── SupportsPagination: bool
│
├── Priority: int (default: 25, lower = higher priority)
├── DownloadClientId: int (optional, links to specific client)
├── AppProfileId: int (for sync filtering)
├── Added: DateTime
└── Capabilities: IndexerCapabilities
```

## Indexer Types

### 1. Native HTTP Indexers

Indexers implemented directly in code, each with custom parsing logic.

**Components per indexer:**
- Main class extending `TorrentIndexerBase<T>` or `UsenetIndexerBase<T>`
- Settings class implementing `IIndexerSettings`
- Request generator implementing `IIndexerRequestGenerator`
- Response parser implementing `IParseIndexerResponse`
- Validator class (optional)

**Example Structure:**
```
MyIndexer/
├── MyIndexer.cs           // Main indexer class
├── MyIndexerSettings.cs   // Configuration settings
├── MyIndexerRequestGenerator.cs  // Builds search URLs
├── MyIndexerParser.cs     // Parses HTML/JSON responses
└── MyIndexerValidator.cs  // Validates settings
```

### 2. Newznab/Torznab Indexers

Generic indexers using standardized Newznab/Torznab APIs.

**Characteristics:**
- Capabilities fetched dynamically from indexer
- Standardized request format
- XML/RSS response parsing
- Multiple preset definitions (DOGnzb, DrunkenSlug, etc.)

**Request Format:**
```
GET {baseUrl}{apiPath}?t={type}&q={query}&apikey={key}&cat={categories}

Parameters:
- t: search | tvsearch | movie | music | book | caps
- q: search query
- cat: comma-separated category IDs
- limit: max results
- offset: pagination offset

Type-specific parameters:
- tvsearch: season, ep, tvdbid, imdbid
- movie: imdbid, tmdbid
- music: artist, album, track
- book: author, title
```

**Response Format (RSS/XML):**
```xml
<rss version="2.0">
  <channel>
    <item>
      <title>Release Title</title>
      <guid>unique-id</guid>
      <link>download-url</link>
      <pubDate>RFC 822 date</pubDate>
      <enclosure url="..." length="..." type="..."/>
      <newznab:attr name="category" value="5000"/>
      <newznab:attr name="size" value="1234567"/>
      <torznab:attr name="seeders" value="100"/>
      <torznab:attr name="peers" value="150"/>
    </item>
  </channel>
</rss>
```

### 3. Cardigann (YAML) Indexers

Dynamically defined indexers using YAML configuration.
See [03-cardigann-definitions.md](03-cardigann-definitions.md) for details.

### 4. RSS Feed Indexers

Simple RSS/Atom feed parsing without search capability.

## Request Generation

### IIndexerRequestGenerator

Responsible for building HTTP requests for searches:

```
IIndexerRequestGenerator
├── GetSearchRequests(MovieSearchCriteria): IndexerPageableRequestChain
├── GetSearchRequests(MusicSearchCriteria): IndexerPageableRequestChain
├── GetSearchRequests(TvSearchCriteria): IndexerPageableRequestChain
├── GetSearchRequests(BookSearchCriteria): IndexerPageableRequestChain
├── GetSearchRequests(BasicSearchCriteria): IndexerPageableRequestChain
│
├── GetCookies: Func<IDictionary<string, string>>
└── CookiesUpdater: Action<IDictionary<string, string>, DateTime?>
```

### IndexerPageableRequestChain

Supports pagination across multiple request tiers:

```
IndexerPageableRequestChain
├── Tiers: List<List<IndexerRequest>>
├── GetAllTiers(): IEnumerable<IEnumerable<IndexerRequest>>
└── Add(requests): void

Usage:
- Tier 0: Primary requests (most specific)
- Tier 1: Fallback requests (broader search)
- Tier N: Additional fallbacks
```

### Request Building Pseudocode

```
FUNCTION BuildSearchRequest(criteria):
    chain = new IndexerPageableRequestChain()

    // Build query string
    query = SanitizeSearchTerm(criteria.SearchTerm)

    // Map categories
    categories = MapCategories(criteria.Categories)

    // Build URL with parameters
    url = BuildUrl(baseUrl, query, categories, criteria)

    // Add authentication (API key, cookies, etc.)
    request = new IndexerRequest(url)
    request.Headers.Add(authHeaders)
    request.Cookies.Add(sessionCookies)

    chain.Add([request])

    // Add pagination if supported
    IF SupportsPagination:
        FOR offset = pageSize TO maxResults STEP pageSize:
            chain.Add([BuildPaginatedRequest(url, offset)])

    RETURN chain
```

## Response Parsing

### IParseIndexerResponse

Transforms HTTP responses into `ReleaseInfo` objects:

```
IParseIndexerResponse
├── ParseResponse(IndexerResponse): IList<ReleaseInfo>
└── CookiesUpdater: Action<IDictionary<string, string>, DateTime?>
```

### Release Information Model

```
ReleaseInfo
├── Identity
│   ├── Guid: string (unique identifier)
│   ├── Title: string
│   └── Description: string
│
├── Metadata
│   ├── PublishDate: DateTime
│   ├── Size: long (bytes)
│   ├── Files: int
│   ├── Grabs: int
│   ├── Year: int
│   ├── Genres: string[]
│   ├── Languages: string[]
│   └── Subs: string[]
│
├── External IDs
│   ├── ImdbId: int
│   ├── TmdbId: int
│   ├── TvdbId: int
│   ├── TvMazeId: int
│   ├── TvRageId: int
│   ├── TraktId: int
│   └── DoubanId: int
│
├── Media-Specific
│   ├── Artist: string (music)
│   ├── Album: string (music)
│   ├── Track: string (music)
│   ├── Label: string (music)
│   ├── Author: string (books)
│   ├── BookTitle: string (books)
│   └── Publisher: string (books)
│
├── URLs
│   ├── DownloadUrl: string
│   ├── InfoUrl: string
│   └── PosterUrl: string
│
├── Categories
│   └── Categories: IndexerCategory[]
│
├── Indexer Info
│   ├── IndexerId: int
│   ├── Indexer: string (name)
│   ├── IndexerPriority: int
│   ├── IndexerPrivacy: IndexerPrivacy
│   └── DownloadProtocol: DownloadProtocol
│
└── IndexerFlags: IndexerFlag[]
    (FreeLeech, HalfLeech, DoubleUpload, etc.)
```

### TorrentInfo (extends ReleaseInfo)

```
TorrentInfo (extends ReleaseInfo)
├── MagnetUrl: string
├── InfoHash: string
├── Seeders: int
├── Peers: int (total, including seeders)
├── MinimumRatio: double
├── MinimumSeedTime: long (seconds)
├── DownloadVolumeFactor: double (0=freeleech, 1=normal)
├── UploadVolumeFactor: double (1=normal, 2=double upload)
└── Scene: bool
```

### Parsing Pseudocode

```
FUNCTION ParseResponse(response):
    releases = []

    IF response.ContentType == "application/json":
        data = JSON.Parse(response.Content)
        rows = SelectJsonPath(data, rowsSelector)
    ELSE:
        document = HTML.Parse(response.Content)
        rows = SelectCss(document, rowsSelector)

    FOR each row IN rows:
        release = new ReleaseInfo()

        // Extract fields using selectors
        release.Title = ExtractField(row, titleSelector)
        release.DownloadUrl = ExtractField(row, downloadSelector)
        release.Size = ParseSize(ExtractField(row, sizeSelector))
        release.PublishDate = ParseDate(ExtractField(row, dateSelector))

        // Torrent-specific fields
        IF protocol == Torrent:
            release = new TorrentInfo(release)
            release.Seeders = ParseInt(ExtractField(row, seedersSelector))
            release.Peers = ParseInt(ExtractField(row, peersSelector))
            release.InfoHash = ExtractField(row, hashSelector)

        // Map categories
        release.Categories = MapCategories(ExtractCategories(row))

        releases.Add(release)

    RETURN releases
```

## Authentication Patterns

### 1. API Key Authentication

Most common for Newznab/Torznab indexers.

```
Request:
GET /api?t=search&apikey={key}&q={query}

OR

Headers:
X-Api-Key: {key}
```

### 2. Cookie Authentication

For indexers requiring browser-like sessions.

```
Flow:
1. Store user's session cookie in settings
2. Include cookie in all requests
3. Monitor for session expiry
4. Prompt user to refresh cookie when needed
```

### 3. Username/Password Authentication

For indexers with login forms.

```
Flow:
1. POST credentials to login endpoint
2. Extract session cookie from response
3. Store cookie with expiration
4. Use cookie for subsequent requests
5. Re-authenticate when cookie expires
```

### 4. Form-Based Login (Cardigann)

Complex login flows defined in YAML.

```
Flow:
1. GET login page
2. Extract form fields (including CSRF tokens)
3. Fill credentials and dynamic fields
4. POST to login endpoint
5. Follow redirects
6. Verify login success via test selector
7. Store cookies
```

### 5. Passkey Authentication

URL-embedded authentication tokens.

```
Request:
GET /download/{passkey}/{torrent_id}

Settings store passkey, appended to all download URLs.
```

## Category Mapping

### Standard Newznab Categories

```
1000 - Console
2000 - Movies
3000 - Audio
4000 - PC
5000 - TV
6000 - XXX
7000 - Books
8000 - Other

Subcategories (examples):
2010 - Movies/Foreign
2020 - Movies/Other
2030 - Movies/SD
2040 - Movies/HD
2045 - Movies/UHD
2050 - Movies/BluRay
2060 - Movies/3D
2070 - Movies/DVD
```

### Category Mapping Process

```
FUNCTION MapCategories(indexerCategories):
    standardCategories = []

    FOR each category IN indexerCategories:
        // Check explicit mapping
        IF categoryMappings.Contains(category):
            standardCategories.Add(categoryMappings[category])

        // Check parent category
        ELSE IF isSubcategory(category):
            parent = GetParentCategory(category)
            standardCategories.Add(parent)

        // Use custom category ID (>= 100000)
        ELSE:
            customId = GenerateCustomId(category)
            standardCategories.Add(customId)

    RETURN Distinct(standardCategories)
```

### IndexerCapabilitiesCategories

```
IndexerCapabilitiesCategories
├── AddCategoryMapping(trackerCat, newznabCat, desc)
├── MapTorznabCapsToTrackers(categories): string[]
├── SupportedCategories(categories): IndexerCategory[]
├── ExpandTorznabQueryCategories(categories): IndexerCategory[]
└── GetTrackerCategories(): List<IndexerCategory>
```

## Rate Limiting

### Query Limits

```
IndexerBaseSettings
├── QueryLimit: int (max queries per period)
├── GrabLimit: int (max downloads per period)
└── LimitUnit: IndexerLimitUnit (Day | Hour)
```

### Limit Checking

```
FUNCTION CheckRateLimits(indexer, isGrab):
    IF isGrab:
        count = GetGrabCountSince(indexer, limitPeriod)
        limit = indexer.Settings.GrabLimit
    ELSE:
        count = GetQueryCountSince(indexer, limitPeriod)
        limit = indexer.Settings.QueryLimit

    IF count >= limit:
        RETURN (limited: true, retryAfter: CalculateResetTime())

    RETURN (limited: false)
```

### Request Throttling

Per-indexer request delay to prevent hammering:

```
FUNCTION ThrottleRequest(indexer):
    lastRequest = GetLastRequestTime(indexer)
    minDelay = indexer.RequestDelay OR defaultDelay

    elapsed = Now - lastRequest
    IF elapsed < minDelay:
        Sleep(minDelay - elapsed)

    RecordRequestTime(indexer, Now)
```

## Indexer Status Tracking

### Status Model

```
IndexerStatus
├── ProviderId: int
├── InitialFailure: DateTime?
├── MostRecentFailure: DateTime?
├── EscalationLevel: int
├── DisabledTill: DateTime?
├── Cookies: string (JSON)
├── CookiesExpirationDate: DateTime?
└── LastRssSyncReleaseInfo: string (JSON)
```

### Failure Handling

```
FUNCTION RecordFailure(indexerId, exception):
    status = GetStatus(indexerId)

    IF status.InitialFailure == null:
        status.InitialFailure = Now

    status.MostRecentFailure = Now
    status.EscalationLevel++

    backoff = CalculateBackoff(status.EscalationLevel)
    status.DisabledTill = Now + backoff

    Save(status)

FUNCTION RecordSuccess(indexerId):
    status = GetStatus(indexerId)
    status.InitialFailure = null
    status.MostRecentFailure = null
    status.EscalationLevel = 0
    status.DisabledTill = null
    Save(status)
```

## Download Handling

### Direct Download

```
FUNCTION Download(release, indexer):
    // Check grab limits
    IF AtGrabLimit(indexer):
        THROW RateLimitedException

    // Build download request
    request = BuildDownloadRequest(release.DownloadUrl)
    AddAuthentication(request, indexer)

    // Execute download
    response = HttpClient.Execute(request)

    // Validate response
    IF response.StatusCode != 200:
        THROW DownloadException

    // Check content type
    IF IsTorrentFile(response):
        ValidateTorrent(response.Content)
    ELSE IF IsNzbFile(response):
        ValidateNzb(response.Content)
    ELSE IF IsMagnetLink(response):
        RETURN ExtractMagnetUrl(response)

    // Record grab in history
    RecordGrab(indexer, release)

    RETURN response.Content
```

### Proxy Download (via Prowlarr)

When applications use Prowlarr's Newznab endpoint:

```
FUNCTION ProxyDownload(indexerId, encryptedLink):
    // Decrypt original URL
    originalUrl = Decrypt(encryptedLink)

    // Get indexer
    indexer = GetIndexer(indexerId)

    // Check limits
    IF AtGrabLimit(indexer):
        RETURN 429 TooManyRequests

    // Decide redirect vs proxy
    IF indexer.SupportsRedirect AND settings.Redirect:
        RETURN Redirect(originalUrl)
    ELSE:
        content = indexer.Download(originalUrl)
        RETURN content
```

## Implementing a New Indexer

### Step 1: Create Settings Class

```
public class MyIndexerSettings : IIndexerSettings
{
    public string BaseUrl { get; set; }
    public IndexerBaseSettings BaseSettings { get; set; }

    // Authentication fields
    [FieldDefinition(Label = "Username")]
    public string Username { get; set; }

    [FieldDefinition(Label = "Password", Privacy = PrivacyLevel.Password)]
    public string Password { get; set; }

    // Custom fields
    [FieldDefinition(Label = "Use Freeleech")]
    public bool FreeleechOnly { get; set; }
}
```

### Step 2: Create Request Generator

```
public class MyIndexerRequestGenerator : IIndexerRequestGenerator
{
    public MyIndexerSettings Settings { get; set; }

    public IndexerPageableRequestChain GetSearchRequests(
        BasicSearchCriteria criteria)
    {
        var chain = new IndexerPageableRequestChain();

        var url = Settings.BaseUrl + "/search";
        var parameters = new Dictionary<string, string>
        {
            {"q", criteria.SanitizedSearchTerm},
            {"cat", string.Join(",", criteria.Categories)}
        };

        var request = new IndexerRequest(url, parameters);
        chain.Add(new[] { request });

        return chain;
    }

    // Implement other search types...
}
```

### Step 3: Create Parser

```
public class MyIndexerParser : IParseIndexerResponse
{
    public IList<ReleaseInfo> ParseResponse(IndexerResponse response)
    {
        var releases = new List<ReleaseInfo>();
        var document = Html.Parse(response.Content);

        var rows = document.QuerySelectorAll("table.torrents tr");

        foreach (var row in rows)
        {
            var release = new TorrentInfo
            {
                Title = row.QuerySelector(".title").TextContent,
                DownloadUrl = row.QuerySelector("a.download").GetAttribute("href"),
                Size = ParseSize(row.QuerySelector(".size").TextContent),
                Seeders = ParseInt(row.QuerySelector(".seeders").TextContent),
                PublishDate = ParseDate(row.QuerySelector(".date").TextContent)
            };

            releases.Add(release);
        }

        return releases;
    }
}
```

### Step 4: Create Main Indexer Class

```
public class MyIndexer : TorrentIndexerBase<MyIndexerSettings>
{
    public override string Name => "My Indexer";
    public override string[] IndexerUrls => new[] { "https://myindexer.com" };
    public override IndexerPrivacy Privacy => IndexerPrivacy.Private;

    public override IndexerCapabilities Capabilities => SetCapabilities();

    private IndexerCapabilities SetCapabilities()
    {
        var caps = new IndexerCapabilities
        {
            TvSearchParams = new List<TvSearchParam>
            {
                TvSearchParam.Q, TvSearchParam.Season, TvSearchParam.Ep
            },
            MovieSearchParams = new List<MovieSearchParam>
            {
                MovieSearchParam.Q, MovieSearchParam.ImdbId
            }
        };

        caps.Categories.AddCategoryMapping(1, NewznabStandardCategory.Movies);
        caps.Categories.AddCategoryMapping(2, NewznabStandardCategory.TV);

        return caps;
    }

    public override IIndexerRequestGenerator GetRequestGenerator()
        => new MyIndexerRequestGenerator { Settings = Settings };

    public override IParseIndexerResponse GetParser()
        => new MyIndexerParser();
}
```
