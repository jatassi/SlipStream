# System Architecture

## Layer Architecture

Prowlarr follows a layered architecture pattern with clear separation of concerns:

```
┌─────────────────────────────────────────────────────────────────┐
│                    Presentation Layer                            │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │  REST API Controllers    │  SignalR Hubs  │  Newznab API    ││
│  └─────────────────────────────────────────────────────────────┘│
├─────────────────────────────────────────────────────────────────┤
│                    Business Logic Layer                          │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │  Services  │  Factories  │  Event Handlers  │  Providers    ││
│  └─────────────────────────────────────────────────────────────┘│
├─────────────────────────────────────────────────────────────────┤
│                    Data Access Layer                             │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │  Repositories  │  Database Context  │  Type Converters      ││
│  └─────────────────────────────────────────────────────────────┘│
├─────────────────────────────────────────────────────────────────┤
│                    Infrastructure Layer                          │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │  HTTP Client  │  File System  │  Caching  │  Messaging      ││
│  └─────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────┘
```

## Core Components

### 1. Indexer System

The indexer system is the heart of Prowlarr, responsible for:
- Managing indexer configurations
- Executing searches against indexers
- Parsing search results
- Handling authentication with indexers
- Rate limiting requests

**Key Components:**
- `IndexerFactory`: Creates and manages indexer instances
- `IndexerBase<TSettings>`: Abstract base for all indexers
- `HttpIndexerBase<TSettings>`: HTTP-specific functionality
- `IndexerStatusService`: Tracks indexer health/failures

### 2. Search System

Aggregates search requests across multiple indexers:
- Dispatches parallel searches
- Filters results by categories
- Deduplicates results
- Caches results for download requests

**Key Components:**
- `ReleaseSearchService`: Orchestrates searches
- `SearchCriteriaBase`: Defines search parameters
- `ReleaseInfo/TorrentInfo`: Result models

### 3. Application Sync System

Manages integration with external applications:
- Pushes indexer configurations
- Maps categories appropriately
- Handles two-way sync
- Manages authentication credentials

**Key Components:**
- `ApplicationService`: Event-driven sync orchestration
- `ApplicationBase<TSettings>`: Base for all applications
- `AppIndexerMapService`: Tracks indexer-to-app mappings

### 4. Download System

Handles release acquisition:
- Routes to appropriate download client
- Validates downloads
- Tracks grab history

**Key Components:**
- `DownloadService`: Main orchestrator
- `DownloadClientProvider`: Client selection/load balancing
- `TorrentClientBase`/`UsenetClientBase`: Protocol-specific bases

### 5. Notification System

Event-driven notification dispatch:
- Health check alerts
- Grab notifications
- Application update notices

**Key Components:**
- `NotificationService`: Event handler
- `NotificationFactory`: Creates notification instances
- `NotificationBase<TSettings>`: Provider base class

## Provider Pattern

Prowlarr uses a "Provider" pattern for extensibility:

```
                    IProvider
                        │
              ┌─────────┴─────────┐
              ▼                   ▼
    ProviderDefinition      ProviderBase<T>
              │                   │
    ┌─────────┴─────────┐        │
    ▼                   ▼        ▼
IndexerDefinition  ApplicationDefinition  Concrete Providers
    │                   │
    └───────┬───────────┘
            ▼
      ProviderFactory
    (Creates instances)
```

**Provider Types:**
1. **Indexers**: Connect to indexer services
2. **Applications**: Connect to media managers
3. **Download Clients**: Send downloads
4. **Notifications**: Send alerts
5. **Indexer Proxies**: HTTP proxies (SOCKS, FlareSolverr)

### Provider Lifecycle

1. **Definition Storage**: Provider configurations stored in database
2. **Factory Resolution**: Factory creates provider instance from definition
3. **Settings Binding**: Settings deserialized and bound to instance
4. **Instance Use**: Provider methods called for operations
5. **Status Tracking**: Success/failure recorded for health monitoring

## Event System

Prowlarr uses an event aggregator pattern for loose coupling:

```
┌─────────────┐     PublishEvent     ┌───────────────┐
│   Service   │──────────────────────▶│EventAggregator│
└─────────────┘                      └───────┬───────┘
                                             │
                         ┌───────────────────┼───────────────────┐
                         ▼                   ▼                   ▼
                  ┌────────────┐      ┌────────────┐      ┌────────────┐
                  │IHandle<T>  │      │IHandle<T>  │      │IHandleAsync│
                  │ Sync       │      │ Sync       │      │   Async    │
                  └────────────┘      └────────────┘      └────────────┘
```

**Event Categories:**
- **Model Events**: Created, Updated, Deleted
- **Provider Events**: Added, Updated, Deleted
- **Download Events**: Grab, Release unavailable
- **Health Events**: Check failed, restored
- **Command Events**: Execution status

## Command System

Long-running tasks use a command pattern:

```
Command Request
      │
      ▼
┌─────────────┐
│CommandQueue │
└─────┬───────┘
      │
      ▼
┌─────────────┐
│CommandExecutor│──▶ Executes IExecute<TCommand>
└─────────────┘
      │
      ▼
CommandCompletedEvent
```

**Command Types:**
- `ApplicationIndexerSyncCommand`: Sync indexers to apps
- `CheckHealthCommand`: Run health checks
- `BackupCommand`: Create backup
- `ResetApiKeyCommand`: Regenerate API key

## HTTP Pipeline

### Incoming Requests

```
HTTP Request
     │
     ▼
┌──────────────────┐
│ Authentication   │──▶ API Key / Form Auth
│ Middleware       │
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│ CORS Middleware  │
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│ Route to         │
│ Controller       │
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│ Validation       │──▶ FluentValidation
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│ Execute Action   │
└──────────────────┘
```

### Outgoing HTTP Requests

```
                                    ┌─────────────────┐
Service Request ──────────────────▶│IIndexerHttpClient│
                                    └────────┬────────┘
                                             │
        ┌────────────────────────────────────┼────────────────────┐
        ▼                                    ▼                    ▼
┌───────────────┐                   ┌───────────────┐    ┌───────────────┐
│ Rate Limiter  │                   │ Proxy Handler │    │ Retry Handler │
└───────┬───────┘                   └───────┬───────┘    └───────┬───────┘
        │                                   │                    │
        └───────────────────────────────────┼────────────────────┘
                                            ▼
                                    ┌───────────────┐
                                    │  HTTP Client  │
                                    └───────────────┘
```

## Caching Strategy

### Cache Types

1. **In-Memory Cache**: Short-lived operational data
   - Search results (30 minutes)
   - Schema definitions (7 days)
   - Capabilities data

2. **Database Cache**: Persistent configuration
   - Indexer definitions
   - Provider settings
   - User preferences

3. **File Cache**: External resources
   - Cardigann YAML definitions
   - Log files

### Cache Invalidation

- **Time-Based**: Automatic expiration
- **Event-Based**: Invalidate on model changes
- **Manual**: API commands to clear cache

## Error Handling Strategy

### Escalation Policy

```
Initial Failure
      │
      ▼
Record Failure ──────────────────────────────────────┐
      │                                              │
      ▼                                              ▼
EscalationLevel++                           Wait (backoff period)
      │                                              │
      ▼                                              │
DisabledTill = CalculateBackoff(level)              │
      │                                              │
      └──────────────────────────────────────────────┘
                      │
                      ▼
              On Success: Reset
```

### Backoff Periods

| Level | Duration |
|-------|----------|
| 1 | 5 minutes |
| 2 | 15 minutes |
| 3 | 30 minutes |
| 4 | 1 hour |
| 5+ | 3 hours |

### Exception Hierarchy

```
Exception
├── DownloadClientException
│   ├── DownloadClientAuthenticationException
│   └── DownloadClientUnavailableException
├── IndexerException
│   └── IndexerAuthException
├── ApplicationException
│   ├── ApplicationValidationException
│   └── ApplicationUnavailableException
└── ReleaseDownloadException
    └── ReleaseUnavailableException
```

## Threading Model

### Thread Pools

1. **Request Threads**: ASP.NET Core thread pool for HTTP requests
2. **Background Workers**: Scheduled task execution
3. **Event Handlers**: Async event processing

### Concurrency Controls

- **Search Dispatch**: Parallel searches with `Task.WhenAll()`
- **Rate Limiting**: Per-indexer request throttling
- **Database Access**: Connection pooling, retry on lock

## Configuration Management

### Configuration Layers

1. **Config File (XML)**: Core application settings
   - Port, bind address, API key
   - Database connection
   - Authentication mode

2. **Database**: Dynamic configuration
   - Indexer definitions
   - Provider settings
   - User preferences

3. **Environment Variables**: Override capability
   - Connection strings
   - Feature flags

### Settings Hierarchy

```
ConfigFileProvider (XML)
         │
         ▼
┌─────────────────┐
│ Config Service  │◀──── Database Config Table
└─────────────────┘
         │
         ▼
   Application
```
