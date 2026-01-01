# Application Sync System

## Overview

Prowlarr synchronizes indexer configurations to external media management applications (Sonarr, Radarr, Lidarr, Readarr, etc.). This enables centralized indexer management while allowing applications to search through Prowlarr's Newznab/Torznab endpoints.

## Supported Applications

| Application | Protocol | API Version | Purpose |
|-------------|----------|-------------|---------|
| Sonarr | HTTP REST | V3 | TV show management |
| Radarr | HTTP REST | V3 | Movie management |
| Lidarr | HTTP REST | V1 | Music management |
| Readarr | HTTP REST | V1 | Book management |
| Whisparr | HTTP REST | V3 | Adult content management |
| Mylar | HTTP REST | V3 (proxy) | Comics management |
| LazyLibrarian | HTTP REST | V1 (proxy) | Books/audio management |

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         Prowlarr                                 │
├─────────────────────────────────────────────────────────────────┤
│  ┌─────────────────┐  ┌─────────────────┐  ┌────────────────┐   │
│  │ IndexerFactory  │  │ApplicationService│  │AppIndexerMap   │   │
│  │ (Indexers)      │  │ (Event Handler) │  │Service         │   │
│  └────────┬────────┘  └────────┬────────┘  └────────┬───────┘   │
│           │                    │                     │           │
│           │       ┌────────────┴────────────┐       │           │
│           │       │  Sync Orchestration     │       │           │
│           │       │  - Add/Update/Remove    │       │           │
│           │       │  - Category Mapping     │       │           │
│           │       │  - URL Generation       │       │           │
│           │       └────────────┬────────────┘       │           │
│           │                    │                     │           │
│  ┌────────▼────────────────────▼─────────────────────▼────────┐ │
│  │                    Application Proxies                      │ │
│  │  SonarrV3Proxy  RadarrV3Proxy  LidarrV1Proxy  ReadarrV1... │ │
│  └───────────────────────────┬─────────────────────────────────┘ │
└──────────────────────────────┼───────────────────────────────────┘
                               │
              HTTP REST API (JSON)
                               │
        ┌──────────────────────┼───────────────────────┐
        ▼                      ▼                       ▼
   ┌─────────┐            ┌─────────┐            ┌─────────┐
   │ Sonarr  │            │ Radarr  │            │ Lidarr  │
   │ /api/v3 │            │ /api/v3 │            │ /api/v1 │
   └─────────┘            └─────────┘            └─────────┘
```

## Application Definition

```
ApplicationDefinition
├── Id: int
├── Name: string
├── Implementation: string (e.g., "Sonarr")
├── Settings: JSON (ApplicationSettings)
├── ConfigContract: string
├── Enable: bool
├── Tags: int[] (for filtering)
└── SyncLevel: ApplicationSyncLevel
    ├── Disabled (no sync)
    ├── AddOnly (only add new indexers)
    └── FullSync (add + update + remove)
```

## Application Settings

### Common Settings

```
ApplicationSettings (base)
├── ProwlarrUrl: string (Prowlarr's external URL)
├── BaseUrl: string (application's URL)
├── ApiKey: string (application's API key)
├── SyncCategories: int[] (categories to sync)
├── SyncRejectBlocklistedTorrentHashesWhileGrabbing: bool
└── Tags: int[] (indexer filter tags)
```

### Sonarr-Specific Settings

```
SonarrSettings extends ApplicationSettings
├── AnimeSyncCategories: int[] (anime categories)
└── SyncAnimeStandardFormatSearch: bool
```

### Category Defaults by Application

| Application | Default Categories |
|-------------|-------------------|
| Sonarr | 5000, 5010, 5020, 5030, 5040, 5045, 5050, 5090 (TV) |
| Sonarr (Anime) | 5070 (Anime) |
| Radarr | 2000, 2010, 2020, 2030, 2040, 2045, 2050, 2060, 2070 (Movies) |
| Lidarr | 3000, 3010, 3020, 3030, 3040, 3050 (Audio) |
| Readarr | 7000, 7010, 7020, 7030 (Books/Audiobooks) |

## Sync Events and Triggers

### Event-Driven Synchronization

```
Events that trigger sync:
├── ProviderAddedEvent<IApplication> → Sync all indexers to new app
├── ProviderUpdatedEvent<IApplication> → Re-sync if settings changed
├── ProviderAddedEvent<IIndexer> → Add indexer to all apps
├── ProviderUpdatedEvent<IIndexer> → Update indexer in all apps
├── ProviderDeletedEvent<IIndexer> → Remove indexer from all apps
├── ApiKeyChangedEvent → Update all indexers with new API key
└── ApplicationIndexerSyncCommand → Manual full sync
```

### Sync Process Flow

```
FUNCTION SyncIndexers(application, removeRemote = false):
    // Get local indexer mappings
    localMappings = AppIndexerMapService.GetMappingsByApp(application.Id)

    // Get remote indexer configurations
    remoteIndexers = application.GetIndexerMappings()

    // Update local mappings with remote IDs
    FOR EACH remoteIndexer IN remoteIndexers:
        mapping = FindOrCreateMapping(remoteIndexer, localMappings)
        UpdateMapping(mapping, remoteIndexer)

    // Get all local indexers
    localIndexers = IndexerFactory.Enabled()

    // Filter by tags
    filteredIndexers = FilterByTags(localIndexers, application.Tags)

    // Sync each indexer
    FOR EACH indexer IN filteredIndexers:
        mapping = localMappings.Find(indexer.Id)

        IF mapping EXISTS:
            // Update existing
            IF syncLevel == FullSync:
                application.UpdateIndexer(indexer)
        ELSE:
            // Check if indexer supports app's categories
            IF SupportsCategories(indexer, application.SyncCategories):
                application.AddIndexer(indexer)

    // Remove orphaned remote indexers (optional)
    IF removeRemote AND syncLevel == FullSync:
        FOR EACH remoteIndexer IN remoteIndexers:
            IF NOT MappedToLocalIndexer(remoteIndexer):
                application.RemoveIndexer(remoteIndexer.Id)
```

## Indexer Mapping

### AppIndexerMap Table

```
AppIndexerMap
├── Id: int (primary key)
├── IndexerId: int (Prowlarr indexer ID)
├── AppId: int (application definition ID)
├── RemoteIndexerId: int (ID in remote application)
└── RemoteIndexerName: string (fallback for apps without IDs)
```

### Mapping Process

```
FUNCTION AddIndexerToApplication(indexer, application):
    // Build indexer configuration for remote app
    config = BuildIndexerConfig(indexer, application)

    // Send to application
    remoteIndexer = application.Proxy.AddIndexer(config)

    // Store mapping
    mapping = new AppIndexerMap
    {
        IndexerId = indexer.Id,
        AppId = application.Id,
        RemoteIndexerId = remoteIndexer.Id,
        RemoteIndexerName = remoteIndexer.Name
    }
    AppIndexerMapService.Insert(mapping)

FUNCTION BuildIndexerConfig(indexer, application):
    prowlarrUrl = application.Settings.ProwlarrUrl
    apiKey = GetProwlarrApiKey()

    config = {
        name: "{indexer.Name} (Prowlarr)",
        implementation: GetImplementation(indexer.Protocol),
        configContract: GetConfigContract(indexer.Protocol),
        enableRss: indexer.SupportsRss,
        enableAutomaticSearch: indexer.SupportsSearch,
        enableInteractiveSearch: indexer.SupportsSearch,
        priority: indexer.Priority,
        fields: [
            { name: "baseUrl", value: "{prowlarrUrl}/{indexer.Id}/" },
            { name: "apiPath", value: "/api" },
            { name: "apiKey", value: apiKey },
            { name: "categories", value: FilterCategories(indexer, application) },
            // Protocol-specific fields...
        ]
    }

    RETURN config
```

## URL Structure

### Prowlarr Endpoint URLs

When synced to applications, indexers point back to Prowlarr:

```
Base URL Pattern:
{ProwlarrUrl}/{IndexerId}/

Example:
http://prowlarr:9696/5/

Newznab API Endpoint:
http://prowlarr:9696/5/api?t=search&apikey={prowlarrApiKey}&q={query}

Download Endpoint:
http://prowlarr:9696/5/download?apikey={prowlarrApiKey}&link={encryptedUrl}
```

### Application sees:

```
Indexer Configuration in Sonarr:
{
  "name": "MyIndexer (Prowlarr)",
  "implementation": "Torznab",
  "fields": {
    "baseUrl": "http://prowlarr:9696/5/",
    "apiPath": "/api",
    "apiKey": "abc123..."
  }
}
```

## Application Proxy Protocol

### Common API Operations

```
// List all indexers
GET {baseUrl}/api/v{version}/indexer
Response: IndexerResource[]

// Get single indexer
GET {baseUrl}/api/v{version}/indexer/{id}
Response: IndexerResource

// Add indexer
POST {baseUrl}/api/v{version}/indexer
Body: IndexerResource
Response: IndexerResource (with ID)

// Update indexer
PUT {baseUrl}/api/v{version}/indexer/{id}
Body: IndexerResource
Response: IndexerResource

// Delete indexer
DELETE {baseUrl}/api/v{version}/indexer/{id}
Response: 200 OK

// Get indexer schema
GET {baseUrl}/api/v{version}/indexer/schema
Response: IndexerResource[] (templates)

// Test indexer
POST {baseUrl}/api/v{version}/indexer/test
Body: IndexerResource
Response: 200 OK or errors
```

### Authentication

All requests include API key header:

```
Headers:
  X-Api-Key: {applicationApiKey}
```

### IndexerResource Structure

```
IndexerResource (JSON sent to applications)
{
  "id": 0,                          // 0 for new, existing ID for update
  "name": "Indexer Name (Prowlarr)",
  "implementation": "Newznab|Torznab",
  "configContract": "NewznabSettings|TorznabSettings",
  "enableRss": true,
  "enableAutomaticSearch": true,
  "enableInteractiveSearch": true,
  "priority": 25,
  "fields": [
    { "name": "baseUrl", "value": "http://prowlarr:9696/1/" },
    { "name": "apiPath", "value": "/api" },
    { "name": "apiKey", "value": "prowlarr-api-key" },
    { "name": "categories", "value": [5030, 5040] },
    { "name": "minimumSeeders", "value": 1 },
    { "name": "seedCriteria.seedRatio", "value": null },
    { "name": "seedCriteria.seedTime", "value": null }
  ],
  "tags": []
}
```

## Category Filtering

### Per-Indexer Category Filtering

```
FUNCTION FilterCategories(indexer, application):
    // Get categories indexer supports
    indexerCategories = indexer.Capabilities.Categories.GetTrackerCategories()

    // Get categories application wants
    appCategories = application.Settings.SyncCategories

    // Find intersection
    intersection = indexerCategories
        .Where(ic => appCategories.Contains(ic.Id) OR
                     appCategories.Contains(ic.ParentId))
        .Select(ic => ic.Id)

    RETURN intersection.ToArray()
```

### Skip Indexer if No Category Match

```
FUNCTION ShouldSyncIndexer(indexer, application):
    filteredCategories = FilterCategories(indexer, application)

    // Don't sync if no category overlap
    IF filteredCategories.IsEmpty:
        RETURN false

    // Don't sync if protocol mismatch
    IF indexer.Protocol == Usenet AND NOT application.SupportsUsenet:
        RETURN false

    RETURN true
```

## Tag-Based Filtering

### Filtering Logic

```
FUNCTION FilterByTags(indexers, applicationTags):
    // If app has no tags, sync all indexers
    IF applicationTags.IsEmpty:
        RETURN indexers

    // Only sync indexers with matching tags
    RETURN indexers.Where(i =>
        i.Tags.Intersect(applicationTags).Any()
    )
```

### Use Cases

- Sync only "Premium" tagged indexers to primary application
- Sync "Public" tagged indexers to less important applications
- Different indexer sets for different Sonarr/Radarr instances

## Field Preservation

When updating existing indexers, user customizations are preserved:

```
FUNCTION UpdateIndexerConfig(existingConfig, newConfig, schema):
    // Get field definitions from schema
    schemaFields = schema.Fields

    // Start with new config as base
    result = newConfig

    // Preserve user-customized fields
    FOR EACH field IN existingConfig.Fields:
        IF IsUserField(field, schemaFields):
            result.Fields[field.Name] = field.Value

    // Preserve user tags
    result.Tags = MergeTags(existingConfig.Tags, newConfig.Tags)

    // Preserve download client assignment
    IF existingConfig.DownloadClientId:
        result.DownloadClientId = existingConfig.DownloadClientId

    RETURN result

FUNCTION IsUserField(field, schemaFields):
    schemaField = schemaFields.Find(field.Name)

    // Fields not in Prowlarr sync are user fields
    syncFields = ["baseUrl", "apiPath", "apiKey", "categories"]
    RETURN NOT syncFields.Contains(field.Name)
```

## Error Handling

### Failure Recording

```
FUNCTION HandleSyncError(application, exception):
    IF exception IS WebException:
        // DNS/Connection failure
        ApplicationStatusService.RecordConnectionFailure(application.Id)

    ELSE IF exception IS HttpException:
        statusCode = exception.StatusCode

        IF statusCode == 401 OR statusCode == 403:
            // Auth failure
            ApplicationStatusService.RecordAuthFailure(application.Id)

        ELSE IF statusCode == 429:
            // Rate limited
            retryAfter = ParseRetryAfter(exception)
            ApplicationStatusService.RecordRateLimit(application.Id, retryAfter)

        ELSE:
            ApplicationStatusService.RecordFailure(application.Id)

    ELSE:
        ApplicationStatusService.RecordFailure(application.Id)
```

### Application Status

```
ApplicationStatus
├── ProviderId: int
├── InitialFailure: DateTime?
├── MostRecentFailure: DateTime?
├── EscalationLevel: int
└── DisabledTill: DateTime?
```

### Escalation Backoff

| Level | Backoff Duration |
|-------|------------------|
| 1 | 5 minutes |
| 2 | 15 minutes |
| 3 | 30 minutes |
| 4 | 1 hour |
| 5+ | 3 hours |

## Schema Caching

### Schema Fetch and Cache

```
FUNCTION GetSchemaForApp(application):
    cacheKey = "schema_{application.Implementation}_{application.Id}"
    cacheDuration = TimeSpan.FromDays(7)

    cached = Cache.Get(cacheKey)
    IF cached EXISTS:
        RETURN cached

    // Fetch from application
    schema = application.Proxy.GetIndexerSchema()

    // Cache result
    Cache.Set(cacheKey, schema, cacheDuration)

    RETURN schema
```

### Schema Invalidation

- On application settings change
- On explicit sync command
- After 7 days (automatic expiry)

## Sync Levels

### Disabled

```
No synchronization occurs.
Use case: Manual management of indexers in application.
```

### AddOnly

```
- New indexers are added to application
- Existing indexers are NOT updated
- Removed indexers are NOT deleted from application

Use case: Let user customize indexers after initial setup.
```

### FullSync (Default)

```
- New indexers are added
- Existing indexers are updated with current settings
- Removed/disabled indexers are deleted from application

Use case: Complete Prowlarr-managed indexer configuration.
```

## Implementation Example: Sonarr Sync

```
FUNCTION SyncToSonarr(indexer, sonarrApp):
    settings = sonarrApp.Settings as SonarrSettings
    proxy = new SonarrV3Proxy(settings)

    // Get existing mapping
    mapping = AppIndexerMapService.GetMapping(indexer.Id, sonarrApp.Id)

    // Build configuration
    config = new SonarrIndexer
    {
        Name = "{indexer.Name} (Prowlarr)",
        Implementation = indexer.Protocol == Torrent ? "Torznab" : "Newznab",
        EnableRss = indexer.SupportsRss,
        EnableAutomaticSearch = indexer.SupportsSearch,
        EnableInteractiveSearch = indexer.SupportsSearch,
        Priority = indexer.Priority,
        Fields = BuildFields(indexer, settings)
    }

    IF mapping EXISTS:
        // Update existing
        config.Id = mapping.RemoteIndexerId
        proxy.UpdateIndexer(config)
    ELSE:
        // Check category support
        categories = FilterCategories(indexer, settings.SyncCategories)
        IF categories.IsEmpty:
            RETURN // Skip - no matching categories

        // Add new
        result = proxy.AddIndexer(config)

        // Store mapping
        AppIndexerMapService.Insert(new AppIndexerMap
        {
            IndexerId = indexer.Id,
            AppId = sonarrApp.Id,
            RemoteIndexerId = result.Id
        })

FUNCTION BuildFields(indexer, settings):
    RETURN [
        { Name: "baseUrl", Value: "{settings.ProwlarrUrl}/{indexer.Id}/" },
        { Name: "apiPath", Value: "/api" },
        { Name: "apiKey", Value: GetProwlarrApiKey() },
        { Name: "categories", Value: FilterCategories(indexer, settings.SyncCategories) },
        { Name: "animeCategories", Value: FilterCategories(indexer, settings.AnimeSyncCategories) },
        { Name: "minimumSeeders", Value: 1 },
        { Name: "seedCriteria.seedRatio", Value: null },
        { Name: "seedCriteria.seedTime", Value: null }
    ]
```
