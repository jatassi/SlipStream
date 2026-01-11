# System Health Monitoring

System health monitoring allows SlipStream to notify users of system status issues and avoid triggering scheduled tasks when dependent features are unhealthy.

## Core Design Principles

- **State Model**: Current state only (no historical event tracking)
- **Severity Levels**: Three-tier (OK / Warning / Error)
- **Notifications**: UI-only display, no push notifications
- **Storage**: In-memory only (resets on application restart)
- **Recovery**: Next successful operation clears error/warning state
- **Timeouts**: Reuse existing configured timeouts for each client type

## Health Categories

### Download Clients

| Aspect | Detail |
|--------|--------|
| States | Binary (OK / Error) |
| Passive Monitoring | Error handling on release fetching, download queue polling |
| Active Monitoring | Scheduled connection test task |
| Special Behavior | Skip connection test if downloads in progress; show OK while downloads are active |

### Indexers

| Aspect | Detail |
|--------|--------|
| States | Three-tier (OK / Warning / Error) |
| Warning Condition | Rate limiting (too many requests) |
| Error Condition | Connection failure |
| Passive Monitoring | Error handling on release searching (manual and automatic) |
| Active Monitoring | Scheduled connection test task |

### Root Folders

| Aspect | Detail |
|--------|--------|
| States | Binary (OK / Error) |
| Error Conditions | Inaccessible OR read-only |
| Passive Monitoring | Error handling on library scan (manual and scheduled) |
| Note | Writability test must work cross-platform (Windows and Unix/Linux/macOS) |

### Metadata Providers

| Aspect | Detail |
|--------|--------|
| States | Three-tier (OK / Warning / Error) |
| Items | TMDB and TVDB tracked as separate items under "Metadata" category |
| Passive Monitoring | Error handling during metadata refresh (manual and scheduled) |

### Storage

| Aspect | Detail |
|--------|--------|
| States | Three-tier (OK / Warning / Error) |
| Scope | Per logical storage location (per drive/volume) |
| Warning Threshold | Less than 20% remaining |
| Error Threshold | Less than 5% remaining |

## Scheduled Connection Tests

- **Scope**: Download clients and indexers
- **Configuration**: Configurable interval per-type (not per-item)
- **Default Interval**: Every 6 hours
- **Skip Behavior**: Skip test for download clients if downloads are in progress

## Dependent Scheduled Tasks

When a feature is unhealthy, scheduled tasks that depend on it:
- **Skip** the scheduled run
- **Log** that the task was skipped due to health issues

## UI: Dashboard Widget

| Aspect | Detail |
|--------|--------|
| Visibility | Always visible (not collapsible/dismissible) |
| Display Format | Category breakdown (e.g., "Download Clients: 2 OK, 1 Error") |
| Actions | Test buttons per category, links to settings |

## UI: System Health Page

| Aspect | Detail |
|--------|--------|
| Navigation | First item under "System" section in left nav |
| Layout | Grouped by category |
| Item Display | Status indicator + last error message |
| Timestamps | Shown only for items with errors/warnings |
| Actions | Test button per item, "Test All" per category, links to settings |

## UI: Movie/Series Add Flow

When a root folder is unhealthy:
- **Block** the add operation with an error message
- Do not allow user to proceed with an unhealthy root folder

## Real-time Updates

- Health status changes pushed to frontend via WebSocket
- Frontend receives updates immediately when health state changes

## Manual Testing

- Test buttons available on both dashboard widget and health page
- Manual test results **update** the health status (not just one-time display)
- Test All buttons available per category
