# Data Model

## Overview

Prowlarr uses a relational database (SQLite or PostgreSQL) with the following key characteristics:
- Dapper ORM for data access
- Repository pattern for data operations
- FluentMigrator for schema migrations
- JSON serialization for complex objects

## Database Support

| Database | Connection String | Notes |
|----------|------------------|-------|
| SQLite | `Data Source={path}` | Default, WAL mode |
| PostgreSQL | `Host=...;Database=...;User Id=...;Password=...` | External DB |

## Core Tables

### Indexers

```sql
CREATE TABLE Indexers (
    Id INTEGER PRIMARY KEY,
    Name TEXT NOT NULL UNIQUE,
    Implementation TEXT NOT NULL,
    Settings TEXT,                    -- JSON serialized settings
    ConfigContract TEXT,              -- Settings class name
    EnableRss INTEGER,
    EnableAutomaticSearch INTEGER,
    EnableInteractiveSearch INTEGER,
    Priority INTEGER DEFAULT 25,
    Added DATETIME
);

Index: IX_Indexers_Name (unique)
```

### IndexerStatus

```sql
CREATE TABLE IndexerStatus (
    Id INTEGER PRIMARY KEY,
    ProviderId INTEGER NOT NULL UNIQUE,   -- FK to Indexers
    InitialFailure DATETIME,
    MostRecentFailure DATETIME,
    EscalationLevel INTEGER DEFAULT 0,
    DisabledTill DATETIME,
    LastRssSyncReleaseInfo TEXT,          -- JSON
    Cookies TEXT,                         -- JSON
    CookiesExpirationDate DATETIME
);

Index: IX_IndexerStatus_ProviderId (unique)
FK: ProviderId -> Indexers.Id
```

### Applications

```sql
CREATE TABLE Applications (
    Id INTEGER PRIMARY KEY,
    Name TEXT NOT NULL UNIQUE,
    Implementation TEXT NOT NULL,
    Settings TEXT,                        -- JSON
    ConfigContract TEXT,
    SyncLevel INTEGER,                    -- 0=Disabled, 1=AddOnly, 2=FullSync
    Tags TEXT                             -- JSON array of tag IDs
);
```

### ApplicationIndexerMapping

```sql
CREATE TABLE ApplicationIndexerMapping (
    Id INTEGER PRIMARY KEY,
    IndexerId INTEGER NOT NULL,           -- FK to Indexers
    AppId INTEGER NOT NULL,               -- FK to Applications
    RemoteIndexerId INTEGER NOT NULL,
    RemoteIndexerName TEXT
);

Index: IX_ApplicationIndexerMapping_IndexerId_AppId (unique)
```

### History

```sql
CREATE TABLE History (
    Id INTEGER PRIMARY KEY,
    IndexerId INTEGER,                    -- FK to Indexers
    Date DATETIME NOT NULL,
    Data TEXT,                            -- JSON (query, categories, etc.)
    EventType INTEGER,                    -- Query, RSS, Grab, Auth, Info
    DownloadId TEXT,
    Successful INTEGER
);

Index: IX_History_Date
Index: IX_History_DownloadId
```

### Notifications

```sql
CREATE TABLE Notifications (
    Id INTEGER PRIMARY KEY,
    Name TEXT NOT NULL,
    Settings TEXT,                        -- JSON
    Implementation TEXT NOT NULL,
    ConfigContract TEXT,
    Tags TEXT,                            -- JSON array
    OnHealthIssue INTEGER DEFAULT 0,
    IncludeHealthWarnings INTEGER DEFAULT 0,
    OnGrab INTEGER DEFAULT 0,
    OnHealthRestored INTEGER DEFAULT 0,
    OnApplicationUpdate INTEGER DEFAULT 0
);
```

### DownloadClients

```sql
CREATE TABLE DownloadClients (
    Id INTEGER PRIMARY KEY,
    Name TEXT NOT NULL,
    Implementation TEXT NOT NULL,
    Settings TEXT,                        -- JSON
    ConfigContract TEXT,
    Enable INTEGER DEFAULT 1,
    Priority INTEGER DEFAULT 1,
    Tags TEXT                             -- JSON array
);
```

### DownloadClientStatus

```sql
CREATE TABLE DownloadClientStatus (
    Id INTEGER PRIMARY KEY,
    ProviderId INTEGER NOT NULL UNIQUE,
    InitialFailure DATETIME,
    MostRecentFailure DATETIME,
    EscalationLevel INTEGER DEFAULT 0,
    DisabledTill DATETIME
);
```

### Tags

```sql
CREATE TABLE Tags (
    Id INTEGER PRIMARY KEY,
    Label TEXT NOT NULL UNIQUE
);
```

### Users

```sql
CREATE TABLE Users (
    Id INTEGER PRIMARY KEY,
    Identifier TEXT NOT NULL UNIQUE,      -- GUID
    Username TEXT NOT NULL UNIQUE,
    Password TEXT NOT NULL,               -- PBKDF2 hash
    Salt TEXT NOT NULL                    -- Base64 salt
);
```

### Config

```sql
CREATE TABLE Config (
    Id INTEGER PRIMARY KEY,
    Key TEXT NOT NULL UNIQUE,             -- Lowercase key
    Value TEXT NOT NULL
);
```

### ScheduledTasks

```sql
CREATE TABLE ScheduledTasks (
    Id INTEGER PRIMARY KEY,
    TypeName TEXT NOT NULL UNIQUE,
    Interval INTEGER NOT NULL,
    LastExecution DATETIME,
    LastStartTime DATETIME
);
```

### Commands

```sql
CREATE TABLE Commands (
    Id INTEGER PRIMARY KEY,
    Name TEXT NOT NULL,
    Body TEXT,                            -- JSON command parameters
    Priority INTEGER,
    Status INTEGER,                       -- Queued, Started, Completed, Failed
    QueuedAt DATETIME NOT NULL,
    StartedAt DATETIME,
    EndedAt DATETIME,
    Duration TEXT,                        -- TimeSpan
    Exception TEXT,
    Trigger INTEGER                       -- Manual, Scheduled, Unspecified
);
```

### CustomFilters

```sql
CREATE TABLE CustomFilters (
    Id INTEGER PRIMARY KEY,
    Type TEXT NOT NULL,
    Label TEXT NOT NULL,
    Filters TEXT                          -- JSON filter definitions
);
```

### AppSyncProfiles

```sql
CREATE TABLE AppSyncProfiles (
    Id INTEGER PRIMARY KEY,
    Name TEXT NOT NULL,
    EnableRss INTEGER DEFAULT 1,
    EnableInteractiveSearch INTEGER DEFAULT 1,
    EnableAutomaticSearch INTEGER DEFAULT 1,
    MinimumSeeders INTEGER DEFAULT 1
);
```

### IndexerProxies

```sql
CREATE TABLE IndexerProxies (
    Id INTEGER PRIMARY KEY,
    Name TEXT NOT NULL,
    Implementation TEXT NOT NULL,
    Settings TEXT,                        -- JSON
    ConfigContract TEXT,
    Tags TEXT                             -- JSON array
);
```

## Entity Models

### ModelBase

```
ModelBase (abstract base for all entities)
├── Id: int (primary key)
```

### ProviderDefinition

```
ProviderDefinition extends ModelBase
├── Name: string
├── ImplementationName: string (computed)
├── Implementation: string (class name)
├── ConfigContract: string (settings class)
├── Enable: bool
├── Message: ProviderMessage
├── Tags: HashSet<int>
└── Settings: IProviderConfig (JSON)
```

### IndexerDefinition

```
IndexerDefinition extends ProviderDefinition
├── IndexerUrls: string[]
├── LegacyUrls: string[]
├── Description: string
├── Encoding: Encoding
├── Language: string
├── Protocol: DownloadProtocol
├── Privacy: IndexerPrivacy
├── SupportsRss: bool
├── SupportsSearch: bool
├── SupportsRedirect: bool
├── SupportsPagination: bool
├── Capabilities: IndexerCapabilities
├── Priority: int (default: 25)
├── Redirect: bool
├── DownloadClientId: int?
├── Added: DateTime
├── AppProfileId: int
├── AppProfile: LazyLoaded<AppSyncProfile>
└── ExtraFields: List<SettingsField>
```

### History

```
History extends ModelBase
├── IndexerId: int
├── Date: DateTime
├── Successful: bool
├── EventType: HistoryEventType
│   ├── ReleaseGrabbed (1)
│   ├── IndexerQuery (2)
│   ├── IndexerRss (3)
│   ├── IndexerAuth (4)
│   └── IndexerInfo (5)
├── Data: Dictionary<string, string>
└── DownloadId: string
```

### ReleaseInfo

```
ReleaseInfo
├── Guid: string (unique)
├── Title: string
├── Description: string
├── PublishDate: DateTime
├── Size: long
├── Files: int
├── Grabs: int
├── DownloadUrl: string
├── InfoUrl: string
├── PosterUrl: string
├── Categories: List<IndexerCategory>
├── IndexerFlags: List<IndexerFlag>
├── Languages: List<string>
├── Subs: List<string>
├── Genres: List<string>
├── Year: int
├── ImdbId: int
├── TmdbId: int
├── TvdbId: int
├── TvMazeId: int
├── TvRageId: int
├── TraktId: int
├── DoubanId: int
├── Author: string
├── BookTitle: string
├── Artist: string
├── Album: string
├── Label: string
├── Track: string
├── Publisher: string
├── IndexerId: int
├── Indexer: string
├── IndexerPriority: int
├── IndexerPrivacy: IndexerPrivacy
└── DownloadProtocol: DownloadProtocol
```

### TorrentInfo

```
TorrentInfo extends ReleaseInfo
├── MagnetUrl: string
├── InfoHash: string
├── Seeders: int
├── Peers: int
├── MinimumRatio: double
├── MinimumSeedTime: long (seconds)
├── DownloadVolumeFactor: double (0=free, 1=normal)
├── UploadVolumeFactor: double (1=normal, 2=double)
└── Scene: bool
```

## Type Converters

### Embedded Document Converter

Serializes complex objects to JSON:

```
Types using EmbeddedDocumentConverter:
├── Dictionary<string, string>
├── HashSet<int>
├── List<int>
├── List<IndexerCategory>
├── List<SettingsField>
├── ReleaseInfo
└── Various collection types
```

### Provider Settings Converter

Handles IProviderConfig settings deserialization:

```
FUNCTION Deserialize(json, configContract):
    // Find settings type by contract name
    settingsType = ResolveType(configContract)

    // Deserialize to concrete type
    settings = JSON.Deserialize(json, settingsType)

    RETURN settings
```

### UTC DateTime Converter

Ensures all dates are stored/retrieved as UTC:

```
FUNCTION Convert(value):
    IF value.Kind == Unspecified:
        RETURN DateTime.SpecifyKind(value, UTC)
    RETURN value.ToUniversalTime()
```

## Repository Pattern

### IBasicRepository<T>

```
interface IBasicRepository<T>
├── All(): List<T>
├── Find(id): T
├── Get(id): T (throws if not found)
├── Insert(model): T
├── Update(model): T
├── Upsert(model): T
├── Delete(id): void
├── Delete(model): void
├── InsertMany(models): void
├── UpdateMany(models): void
├── DeleteMany(ids): void
├── Purge(): void
├── HasItems(): bool
└── GetPaged(pagingSpec): PagingSpec<T>
```

### BasicRepository<T>

```
class BasicRepository<T> implements IBasicRepository<T>
├── Database connection
├── Query building (SqlBuilder)
├── Event publishing (ModelEvent)
├── Retry logic for SQLite locks
└── Batch operations
```

### Query Building

```
FUNCTION BuildQuery(criteria):
    builder = new SqlBuilder()

    builder.Select("*")
    builder.From(TableName)

    IF criteria.Filter EXISTS:
        builder.Where(BuildWhereClause(criteria.Filter))

    IF criteria.SortKey EXISTS:
        builder.OrderBy(criteria.SortKey, criteria.SortDirection)

    IF criteria.Limit EXISTS:
        builder.Limit(criteria.Limit)
        builder.Offset(criteria.Offset)

    RETURN builder.Build()
```

## Migrations

### Migration Framework

```
Migration Base Class:
├── NzbDroneMigrationBase
│   ├── MainDbUpgrade() - Main database
│   └── LogDbUpgrade() - Log database

Migration Naming:
├── 001_initial_setup.cs
├── 002_add_column.cs
├── ...
├── 043_latest_change.cs
```

### Migration Operations

```
Available Operations:
├── Create.Table(name)
├── Alter.Table(name)
├── Delete.Table(name)
├── Create.Index(name)
├── Create.Column(name)
├── Alter.Column(name)
├── Delete.Column(name)
├── Execute.Sql(sql)
└── Rename.Table(from).To(to)
```

### Initial Schema (Migration 001)

```
Tables Created:
├── Config
├── History
├── Notifications
├── ScheduledTasks
├── Indexers
├── ApplicationIndexerMapping
├── Applications
├── Tags
├── Users
├── Commands
├── IndexerStatus
└── CustomFilters
```

## Database Configuration

### SQLite Settings

```
Connection String Options:
├── Cache = Shared
├── Journal Mode = WAL (Windows/Linux) or Truncate (macOS)
├── Cache Size = 20MB
├── DateTimeKind = UTC
└── BusyTimeout = 100ms
```

### PostgreSQL Settings

```
Connection String Options:
├── Pooling = true
├── MaxPoolSize = 100
├── Timeout = 30
├── Timezone = UTC
└── ApplicationName = Prowlarr
```

## Caching Strategy

### Entity Caching

```
Cache Types:
├── In-memory cache for frequently accessed data
├── Database as persistent cache
└── File cache for external resources (YAML definitions)

Cache Invalidation:
├── On model update/delete
├── Via ModelEvent publishing
├── Time-based expiration
```

### Provider Status Caching

```
IndexerStatus cached with:
├── Cookies (with expiration)
├── Failure state
├── Last successful sync info
```

## Relationships

### Entity Relationships

```
Indexers (1) ─────── (N) History
    │
    └── (1) ─────── (1) IndexerStatus
    │
    └── (N) ─────── (N) Applications
              via ApplicationIndexerMapping

Applications (1) ── (N) ApplicationIndexerMapping
    │
    └── (1) ─────── (1) ApplicationStatus

DownloadClients (1) ── (1) DownloadClientStatus

Tags (N) ─────────── (N) [Multiple entities via JSON arrays]
```

### Lazy Loading

```
LazyLoaded<T> pattern:
├── Deferred loading of related entities
├── Loaded on first access
├── Reduces initial query overhead

Example:
IndexerDefinition.AppProfile: LazyLoaded<AppSyncProfile>
```

## Data Validation

### FluentValidation

```
Validator Pattern:
├── SharedValidator: Common rules (always run)
├── PostValidator: Rules for creation
├── PutValidator: Rules for updates
├── PropertyValidator: Custom validation logic
```

### Common Validations

```
IndexerSettingsValidator:
├── BaseUrl: Required, valid URL
├── ApiKey: Required (if applicable)
├── QueryLimit: >= 0
├── GrabLimit: >= 0

ApplicationSettingsValidator:
├── BaseUrl: Required, valid URL
├── ApiKey: Required
├── ProwlarrUrl: Required, valid URL
```

## Event System Integration

### Model Events

```
Events Published on Database Changes:
├── ModelEvent<T>.Created
├── ModelEvent<T>.Updated
├── ModelEvent<T>.Deleted

Handlers:
├── Cache invalidation
├── SignalR broadcast
├── Related entity updates
```

### Event Flow

```
Repository.Insert(model)
    │
    ▼
Database Insert
    │
    ▼
EventAggregator.PublishEvent(ModelEvent.Created)
    │
    ├──▶ SignalR Broadcast
    ├──▶ Cache Invalidation
    └──▶ Related Updates
```
