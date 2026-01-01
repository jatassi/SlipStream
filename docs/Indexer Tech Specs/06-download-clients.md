# Download Client Integration

## Overview

Prowlarr integrates with download clients to send grabbed releases for download. It supports both Usenet (NZB) and Torrent protocols through a unified interface.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                      Download Request                            │
│   Release: { Title, DownloadUrl, Protocol, Size, ... }          │
└─────────────────────────────┬────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                      DownloadService                             │
│   - Select appropriate client                                   │
│   - Handle redirect vs. download                                │
│   - Record history                                               │
└─────────────────────────────┬────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                  DownloadClientProvider                          │
│   - Filter by protocol (Usenet/Torrent)                         │
│   - Apply priority ordering                                      │
│   - Load balance across clients                                 │
│   - Skip blocked clients                                        │
└─────────────────────────────┬────────────────────────────────────┘
                              │
         ┌────────────────────┼────────────────────┐
         ▼                    ▼                    ▼
┌──────────────────┐ ┌──────────────────┐ ┌──────────────────┐
│ UsenetClientBase │ │TorrentClientBase │ │ BlackholeClient  │
│  - SABnzbd       │ │  - qBittorrent   │ │  - Write to disk │
│  - NzbGet        │ │  - Transmission  │ │                  │
│  - Usenet Blackhole│ │  - Deluge      │ │                  │
└────────┬─────────┘ └────────┬─────────┘ └────────┬─────────┘
         │                    │                    │
         ▼                    ▼                    ▼
┌─────────────────────────────────────────────────────────────────┐
│              Download Client APIs / Disk I/O                     │
└─────────────────────────────────────────────────────────────────┘
```

## Download Client Interface

```
IDownloadClient
├── Properties
│   ├── Name: string
│   ├── Protocol: DownloadProtocol (Usenet | Torrent)
│   └── SupportsCategories: bool
│
└── Methods
    └── Download(release, redirect, indexer): Task<string>
        Returns: Download ID (or null for blackhole)
```

## Protocol Types

### DownloadProtocol

```
DownloadProtocol
├── Unknown = 0
├── Usenet = 1   // NZB files
└── Torrent = 2  // Torrent files / Magnet links
```

## Supported Download Clients

### Torrent Clients

| Client | API Type | Auth | Categories | Seed Config |
|--------|----------|------|------------|-------------|
| qBittorrent | REST | User/Pass | Yes (labels) | Yes (v2.8.1+) |
| Transmission | JSON-RPC | User/Pass | No | No |
| Deluge | JSON-RPC | Password | Yes (labels) | Yes |
| rTorrent | XMLRPC | User/Pass | No | No |
| uTorrent | REST | User/Pass | No | No |
| Flood | REST | User/Pass | No | No |
| Aria2 | JSON-RPC | Token | No | No |
| DownloadStation | REST | User/Pass | Yes | Limited |
| Torrent Blackhole | Filesystem | N/A | No | N/A |

### Usenet Clients

| Client | API Type | Auth | Categories |
|--------|----------|------|------------|
| SABnzbd | REST | API Key | Yes |
| NzbGet | JSON-RPC | User/Pass | Yes |
| NzbVortex | REST | API Key | Yes |
| DownloadStation | REST | User/Pass | Yes |
| Usenet Blackhole | Filesystem | N/A | No |

## Download Flow

### Torrent Download

```
FUNCTION DownloadTorrent(release, redirect, indexer):
    // Determine download method
    IF release.MagnetUrl EXISTS:
        downloadUrl = release.MagnetUrl
        isMagnet = true
    ELSE:
        downloadUrl = release.DownloadUrl
        isMagnet = false

    // Try magnet first (preferred - no file download needed)
    IF isMagnet:
        TRY:
            downloadId = AddFromMagnetLink(downloadUrl, release)
            RETURN downloadId
        CATCH:
            // Fall back to torrent file if available
            IF release.DownloadUrl EXISTS:
                downloadUrl = release.DownloadUrl
                isMagnet = false
            ELSE:
                THROW

    // Download torrent file
    IF NOT isMagnet:
        // Get torrent bytes from indexer
        torrentBytes = indexer.Download(downloadUrl)

        // Validate torrent
        ValidateTorrent(torrentBytes)

        // Extract info hash
        infoHash = ExtractInfoHash(torrentBytes)

        // Add to client
        downloadId = AddFromTorrentFile(torrentBytes, release)

        // Configure seed settings
        IF Client.SupportsSeedConfig:
            ConfigureSeedLimits(downloadId, release)

        RETURN downloadId
```

### Usenet Download

```
FUNCTION DownloadUsenet(release, redirect, indexer):
    IF redirect:
        // Send URL directly to client
        downloadId = AddFromLink(release.DownloadUrl)
    ELSE:
        // Download NZB file
        nzbBytes = indexer.Download(release.DownloadUrl)

        // Validate NZB
        ValidateNzb(nzbBytes)

        // Add to client
        downloadId = AddFromNzbFile(nzbBytes, release)

    RETURN downloadId
```

## Client Selection

### DownloadClientProvider Algorithm

```
FUNCTION GetDownloadClient(indexerId, protocol):
    // Get all enabled clients
    allClients = DownloadClientFactory.Enabled()

    // Filter by protocol
    clients = allClients.Where(c => c.Protocol == protocol)

    // Check if indexer has specific client assigned
    indexer = IndexerFactory.Get(indexerId)
    IF indexer.DownloadClientId.HasValue:
        specificClient = clients.Find(indexer.DownloadClientId)
        IF specificClient EXISTS AND NOT IsBlocked(specificClient):
            RETURN specificClient
        // If blocked, fall through to others

    // Get blocked clients
    blockedIds = DownloadClientStatusService.GetBlockedProviders()

    // Prefer non-blocked clients
    availableClients = clients.Where(c => NOT blockedIds.Contains(c.Id))

    IF availableClients.IsEmpty:
        // Fall back to blocked clients
        availableClients = clients

    // Sort by priority
    sortedClients = availableClients.OrderBy(c => c.Priority)

    // Round-robin within same priority
    selectedClient = GetNextInRotation(sortedClients)

    RETURN selectedClient

FUNCTION GetNextInRotation(clients):
    // Group by priority
    groups = clients.GroupBy(c => c.Priority)

    FOR EACH group IN groups:
        lastUsedId = Cache.Get("lastClient_{group.Priority}")

        // Find next client after last used
        FOR EACH client IN group:
            IF client.Id > lastUsedId:
                Cache.Set("lastClient_{group.Priority}", client.Id)
                RETURN client

        // Wrap around
        firstClient = group.First()
        Cache.Set("lastClient_{group.Priority}", firstClient.Id)
        RETURN firstClient
```

## Client Implementation: qBittorrent

### Settings

```
QBittorrentSettings
├── Host: string
├── Port: int (default: 8080)
├── UseSsl: bool
├── UrlBase: string (optional)
├── Username: string
├── Password: string
├── Category: string (default category)
├── Priority: DownloadClientPriority
├── InitialState: QBittorrentState
│   ├── ForceStart
│   ├── Start
│   └── Pause
├── SequentialOrder: bool
└── FirstAndLastFirst: bool
```

### API Endpoints

```
Authentication:
POST /api/v2/auth/login
  Form: username, password
  Response: Sets cookie "SID"

Add Torrent:
POST /api/v2/torrents/add
  Form: urls (magnet) OR torrents (file)
  Form: category, savepath, rename, paused, etc.

Get Torrent Properties:
GET /api/v2/torrents/properties?hash={hash}

Set Seed Limits (v2.8.1+):
POST /api/v2/torrents/setShareLimits
  Form: hashes, ratioLimit, seedingTimeLimit

Get Categories:
GET /api/v2/torrents/categories
```

### Implementation Pseudocode

```
FUNCTION AddTorrent(torrentBytes, release):
    // Authenticate
    EnsureAuthenticated()

    // Extract hash before adding
    hash = CalculateInfoHash(torrentBytes)

    // Build request
    request = new FormRequest("/api/v2/torrents/add")
    request.AddFile("torrents", torrentBytes, "release.torrent")
    request.AddField("category", Settings.Category)
    request.AddField("rename", release.Title)

    IF Settings.InitialState == Pause:
        request.AddField("paused", "true")

    // Send request
    response = HttpClient.Post(request)

    IF response.StatusCode != 200:
        THROW DownloadClientException("Add failed")

    // Wait for torrent to appear (polling)
    FOR i = 0 TO 5:
        Sleep(100ms)
        IF GetTorrentProperties(hash) EXISTS:
            BREAK

    // Apply seed configuration
    IF Settings.SeedRatio OR Settings.SeedTime:
        SetSeedLimits(hash, Settings.SeedRatio, Settings.SeedTime)

    // Start if needed
    IF Settings.InitialState == ForceStart:
        ForceStartTorrent(hash)

    RETURN hash
```

## Client Implementation: SABnzbd

### Settings

```
SabnzbdSettings
├── Host: string
├── Port: int (default: 8080)
├── UseSsl: bool
├── UrlBase: string (optional)
├── ApiKey: string
├── Username: string (optional, for basic auth fallback)
├── Password: string (optional)
├── Category: string
└── Priority: SabnzbdPriority
    ├── Default = -100
    ├── Paused = -2
    ├── Low = -1
    ├── Normal = 0
    ├── High = 1
    └── Force = 2
```

### API Endpoints

```
Base URL: {host}:{port}/{urlBase}/api

Add by URL:
GET ?mode=addurl&name={url}&cat={category}&priority={priority}&apikey={key}

Add by File:
POST ?mode=addfile&cat={category}&priority={priority}&apikey={key}
  Multipart: file

Get Categories:
GET ?mode=get_cats&output=json&apikey={key}

Get Config:
GET ?mode=get_config&section=misc&apikey={key}
```

### Implementation Pseudocode

```
FUNCTION AddNzb(nzbBytes, release):
    // Build request
    url = BuildUrl("addfile")
    url.AddQuery("cat", Settings.Category)
    url.AddQuery("priority", Settings.Priority)
    url.AddQuery("nzbname", SanitizeFilename(release.Title))

    request = new MultipartRequest(url)
    request.AddFile("nzbfile", nzbBytes, "release.nzb")

    // Add auth
    IF Settings.ApiKey:
        request.AddQuery("apikey", Settings.ApiKey)
    ELSE:
        request.AddBasicAuth(Settings.Username, Settings.Password)

    // Send request
    response = HttpClient.Post(request)

    // Parse response
    result = JSON.Parse(response.Content)

    IF result.status == false:
        THROW DownloadClientException(result.error)

    // Return job ID
    RETURN result.nzo_ids.First()
```

## Category Mapping

### Category Configuration

```
DownloadClientCategory
├── ClientCategory: string (name in download client)
└── Categories: int[] (Newznab category IDs)
```

### Mapping Algorithm

```
FUNCTION GetCategoryForRelease(release, clientCategories):
    releaseCategories = release.Categories.Select(c => c.Id)

    // Find matching category
    FOR EACH mapping IN clientCategories:
        IF releaseCategories.Intersect(mapping.Categories).Any():
            RETURN mapping.ClientCategory

        // Check parent categories
        FOR EACH releaseCat IN releaseCategories:
            parentId = GetParentCategory(releaseCat)
            IF mapping.Categories.Contains(parentId):
                RETURN mapping.ClientCategory

    // Return default category
    RETURN Settings.DefaultCategory
```

## Seed Configuration (Torrents)

### TorrentSeedConfiguration

```
TorrentSeedConfiguration
├── Ratio: double? (e.g., 2.0 for 200%)
└── SeedTime: TimeSpan? (e.g., 24 hours)
```

### Applying Seed Limits

```
FUNCTION ConfigureSeedLimits(downloadId, release):
    // Get limits from release or default settings
    IF release.MinimumRatio.HasValue:
        ratio = release.MinimumRatio
    ELSE:
        ratio = Settings.SeedRatioLimit

    IF release.MinimumSeedTime.HasValue:
        seedTime = release.MinimumSeedTime
    ELSE:
        seedTime = Settings.SeedTimeLimit

    // Apply to client
    Client.SetSeedLimits(downloadId, ratio, seedTime)
```

## Validation

### NZB Validation

```
FUNCTION ValidateNzb(content):
    // Check for indexer error response
    IF StartsWithXmlTag(content) AND ContainsError(content):
        error = ExtractErrorMessage(content)
        THROW InvalidNzbException(error)

    // Parse as XML
    TRY:
        doc = XML.Parse(content)
        files = doc.SelectNodes("//file")

        IF files.Count == 0:
            THROW InvalidNzbException("No files in NZB")

    CATCH XmlException:
        THROW InvalidNzbException("Invalid NZB format")
```

### Torrent Validation

```
FUNCTION ValidateTorrent(content):
    // Check for magnet link disguised as file
    IF content.StartsWith("magn"):
        // It's actually a magnet link
        RETURN ExtractMagnetUrl(content)

    // Parse as bencoded torrent
    TRY:
        torrent = BencodedParser.Parse(content)

        IF NOT torrent.ContainsKey("info"):
            THROW InvalidTorrentException("Missing info dictionary")

        // Extract hash
        hash = CalculateSHA1(torrent["info"])

        RETURN (isValid: true, hash: hash)

    CATCH:
        THROW InvalidTorrentException("Invalid torrent format")
```

## Blackhole Clients

### Torrent Blackhole

```
FUNCTION Download(release, redirect, indexer):
    // Get torrent content
    torrentBytes = indexer.Download(release.DownloadUrl)

    // Validate
    ValidateTorrent(torrentBytes)

    // Build filename
    filename = SanitizeFilename(release.Title) + ".torrent"
    path = Path.Combine(Settings.TorrentFolder, filename)

    // Write to disk
    File.WriteAllBytes(path, torrentBytes)

    // No tracking ID available
    RETURN null
```

### Usenet Blackhole

```
FUNCTION Download(release, redirect, indexer):
    // Get NZB content
    nzbBytes = indexer.Download(release.DownloadUrl)

    // Validate
    ValidateNzb(nzbBytes)

    // Build filename
    filename = SanitizeFilename(release.Title) + ".nzb"
    path = Path.Combine(Settings.NzbFolder, filename)

    // Write to disk
    File.WriteAllBytes(path, nzbBytes)

    RETURN null
```

## Error Handling

### Download Exceptions

```
DownloadClientException (base)
├── DownloadClientAuthenticationException
│   - Invalid credentials
│   - API key expired
├── DownloadClientUnavailableException
│   - Connection refused
│   - Timeout
│   - Service unavailable
└── DownloadClientRejectedReleaseException
    - Duplicate torrent
    - Invalid file
```

### Failure Recording

```
FUNCTION HandleDownloadError(client, exception):
    IF exception IS DownloadClientAuthenticationException:
        StatusService.RecordAuthFailure(client.Id)
        DisableClient(client)

    ELSE IF exception IS DownloadClientUnavailableException:
        StatusService.RecordConnectionFailure(client.Id)
        // Client will be skipped temporarily

    ELSE IF exception IS DownloadClientRejectedReleaseException:
        // Don't record as client failure
        // Throw to caller for handling

    ELSE:
        StatusService.RecordFailure(client.Id)
```

### Status Tracking

```
DownloadClientStatus
├── ProviderId: int
├── InitialFailure: DateTime?
├── MostRecentFailure: DateTime?
├── EscalationLevel: int (0-5)
└── DisabledTill: DateTime?
```

## Connection Testing

### Test Steps

```
FUNCTION TestConnection(client):
    failures = []

    // 1. Test connectivity
    TRY:
        Ping(client.Settings.Host, client.Settings.Port)
    CATCH:
        failures.Add("Cannot connect to host")
        RETURN failures

    // 2. Test authentication
    TRY:
        Authenticate(client)
    CATCH AuthException:
        failures.Add("Authentication failed")
        RETURN failures

    // 3. Test API version
    TRY:
        version = GetVersion(client)
        IF version < MinimumVersion:
            failures.Add("Client version too old")
    CATCH:
        // Optional - don't fail

    // 4. Test category (if configured)
    IF client.Settings.Category:
        TRY:
            categories = GetCategories(client)
            IF NOT categories.Contains(client.Settings.Category):
                failures.Add("Category not found")
        CATCH:
            // Some clients don't support category listing

    RETURN failures
```
