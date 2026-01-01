# Notification System

## Overview

Prowlarr uses an event-driven notification system that triggers alerts based on application events. Notifications can be sent through various channels including email, webhooks, and messaging services.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                       Event Sources                              │
│  HealthCheckService  │  DownloadService  │  UpdateService       │
└───────────────────────────────┬─────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│                      EventAggregator                             │
│   Dispatches events to registered handlers                      │
└───────────────────────────────┬─────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│                    NotificationService                           │
│   IHandle<HealthCheckFailedEvent>                               │
│   IHandle<HealthCheckRestoredEvent>                             │
│   IHandle<IndexerDownloadEvent>                                 │
│   IHandle<UpdateInstalledEvent>                                 │
└───────────────────────────────┬─────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│                    NotificationFactory                           │
│   Filter by event type enablement                               │
│   Filter blocked notifications                                   │
└───────────────────────────────┬─────────────────────────────────┘
                                │
         ┌──────────────────────┼──────────────────────┐
         ▼                      ▼                      ▼
    ┌─────────┐            ┌─────────┐           ┌─────────┐
    │  Email  │            │ Webhook │           │ Discord │
    └─────────┘            └─────────┘           └─────────┘
```

## Event Types

### HealthCheckFailedEvent

```
Triggered: When a health check fails
Properties:
├── HealthCheck: HealthCheck
│   ├── Source: string (check identifier)
│   ├── Type: HealthCheckResult (Error | Warning | Notice)
│   ├── Message: string
│   └── WikiUrl: string
└── IsInStartupGracePeriod: bool

Grace Period: First 15 minutes after startup
```

### HealthCheckRestoredEvent

```
Triggered: When a previously failed check passes
Properties:
├── HealthCheck: HealthCheck (previous failure)
├── IsInStartupGracePeriod: bool
└── PreviouslyFailed: bool
```

### IndexerDownloadEvent

```
Triggered: When a release is grabbed
Properties:
├── Release: ReleaseInfo
├── DownloadClient: string
├── DownloadClientId: int
├── DownloadId: string
├── Successful: bool
├── Redirect: bool
├── GrabTrigger: GrabTrigger (Manual | Api)
├── Source: string
├── Host: string
└── ElapsedTime: TimeSpan?
```

### UpdateInstalledEvent

```
Triggered: After application update
Properties:
├── PreviousVersion: Version
└── NewVersion: Version
```

## Notification Definition

```
NotificationDefinition extends ProviderDefinition
├── OnHealthIssue: bool
├── OnHealthRestored: bool
├── OnApplicationUpdate: bool
├── OnGrab: bool
├── SupportsOnGrab: bool
├── IncludeManualGrabs: bool
├── SupportsOnHealthIssue: bool
├── SupportsOnHealthRestored: bool
├── IncludeHealthWarnings: bool
└── SupportsOnApplicationUpdate: bool
```

## Notification Interface

```
INotification extends IProvider
├── OnGrab(GrabMessage): void
├── OnHealthIssue(HealthCheck): void
├── OnHealthRestored(HealthCheck): void
├── OnApplicationUpdate(ApplicationUpdateMessage): void
├── SupportsOnGrab: bool
├── SupportsOnHealthIssue: bool
├── SupportsOnHealthRestored: bool
└── SupportsOnApplicationUpdate: bool
```

## Message Types

### GrabMessage

```
GrabMessage
├── Release: ReleaseInfo
├── Successful: bool
├── Host: string
├── Source: string
├── GrabTrigger: GrabTrigger
├── Redirect: bool
├── Message: string
├── DownloadClientType: string
├── DownloadClientName: string
└── DownloadId: string
```

### ApplicationUpdateMessage

```
ApplicationUpdateMessage
├── Message: string
├── PreviousVersion: Version
└── NewVersion: Version
```

## Supported Notification Providers

### Email

```
Settings:
├── Server: string (SMTP host)
├── Port: int (default: 587)
├── UseEncryption: bool
├── Username: string
├── Password: string
├── From: string
├── To: string[]
├── Cc: string[]
└── Bcc: string[]

Implementation:
├── Uses MailKit library
├── Supports SSL/TLS
├── HTML and plain text bodies
```

### Webhook

```
Settings:
├── Url: string
├── Method: int (POST = 1, PUT = 2)
├── Username: string (optional basic auth)
└── Password: string

Payload Format:
{
  "eventType": "Grab",
  "instanceName": "Prowlarr",
  "release": {
    "guid": "...",
    "title": "...",
    "indexer": "...",
    "size": 1234567,
    "downloadUrl": "..."
  },
  "downloadClient": "...",
  "downloadClientName": "..."
}
```

### Discord

```
Settings:
├── WebhookUrl: string
├── Username: string (optional)
├── Avatar: string (optional)
├── OnGrabFields: int[] (fields to include)
├── OnHealthIssueFields: int[]
├── OnApplicationUpdateFields: int[]

Embed Fields:
├── 0: Release
├── 1: Indexer
├── 2: DownloadClient
├── 3: GrabTrigger
├── 4: Source
├── 5: Host

Embed Colors:
├── Success: Green
├── Warning: Yellow
├── Error: Red
```

### Other Providers

| Provider | Protocol | Features |
|----------|----------|----------|
| Apprise | HTTP | Unified notification service |
| Gotify | HTTP REST | Self-hosted push server |
| Join | HTTP | Android notifications |
| Mailgun | HTTP REST | Email via Mailgun API |
| Notifiarr | HTTP | Arr-specific notification service |
| Ntfy | HTTP | Self-hosted push |
| Prowl | HTTP | iOS push notifications |
| PushBullet | HTTP REST | Cross-platform push |
| Pushcut | HTTP | iOS automation |
| Pushover | HTTP | Push notifications |
| SendGrid | HTTP REST | Email via SendGrid API |
| Slack | HTTP | Slack webhooks |
| Telegram | HTTP | Telegram bot API |
| Custom Script | Process | Execute local script |

## Notification Service Flow

### Event Handling

```
FUNCTION HandleEvent(event):
    // Get enabled notifications for event type
    notifications = NotificationFactory.GetEnabledByEventType(event)

    // Filter blocked notifications
    notifications = FilterBlocked(notifications)

    // Apply event-specific filters
    notifications = ApplyEventFilters(notifications, event)

    // Send to each notification
    FOR EACH notification IN notifications:
        TRY:
            SendNotification(notification, event)
            NotificationStatusService.RecordSuccess(notification.Id)
        CATCH exception:
            LogWarning("Notification failed: {exception}")
            NotificationStatusService.RecordFailure(notification.Id)
```

### Event Filters

```
FUNCTION ShouldHandleOnGrab(notification, event):
    IF NOT notification.OnGrab:
        RETURN false

    // Check manual grab filter
    IF event.GrabTrigger == Manual:
        RETURN notification.IncludeManualGrabs

    RETURN true

FUNCTION ShouldHandleHealthFailure(notification, check):
    IF NOT notification.OnHealthIssue:
        RETURN false

    // Check warning filter
    IF check.Type == Warning:
        RETURN notification.IncludeHealthWarnings

    RETURN true

FUNCTION ShouldHandleIndexer(notification, indexer):
    // If notification has no tags, handle all
    IF notification.Tags.IsEmpty:
        RETURN true

    // Check for matching tags
    RETURN notification.Tags.Intersect(indexer.Tags).Any()
```

## Notification Queue

### Health Check Queue

```
Queue Behavior:
├── Health events queued during startup grace period
├── Processed after grace period ends
├── Duplicate events deduplicated
└── Failed -> Restored pairs cancelled out

FUNCTION ProcessQueue():
    FOR EACH queuedEvent IN healthQueue:
        IF NOT HasBeenRestored(queuedEvent):
            SendHealthNotification(queuedEvent)
    healthQueue.Clear()
```

## Status Tracking

### NotificationStatus

```
NotificationStatus extends ProviderStatusBase
├── ProviderId: int
├── InitialFailure: DateTime?
├── MostRecentFailure: DateTime?
├── EscalationLevel: int
└── DisabledTill: DateTime?
```

### Failure Handling

```
Escalation Policy:
├── Minimum time before considering failure: 5 minutes
├── Maximum escalation level: 5
├── Backoff periods: 5m, 15m, 30m, 1h, 3h

FUNCTION RecordFailure(notificationId):
    status = GetStatus(notificationId)

    IF status.InitialFailure == null:
        status.InitialFailure = Now

    status.MostRecentFailure = Now
    status.EscalationLevel++

    backoff = GetBackoffPeriod(status.EscalationLevel)
    status.DisabledTill = Now + backoff

    Save(status)
```

## Notification Testing

### Test Workflow

```
POST /api/v1/notification/test

Body: NotificationResource

Process:
1. Create notification instance from resource
2. Validate settings
3. Send test message
4. Return validation results

Test Message Content:
├── Health: "Test health check warning"
├── Grab: Sample release info
├── Update: Version 1.0 -> 1.1
```

## Implementation Example: Discord

```
FUNCTION OnGrab(grabMessage):
    embed = new DiscordEmbed
    {
        Title = grabMessage.Release.Title,
        Description = BuildDescription(grabMessage),
        Timestamp = DateTime.UtcNow,
        Color = grabMessage.Successful ? Green : Red,
        Footer = new DiscordFooter { Text = "Prowlarr" }
    }

    // Add configured fields
    IF Settings.OnGrabFields.Contains(FieldType.Release):
        embed.AddField("Release", grabMessage.Release.Title)

    IF Settings.OnGrabFields.Contains(FieldType.Indexer):
        embed.AddField("Indexer", grabMessage.Release.Indexer)

    IF Settings.OnGrabFields.Contains(FieldType.DownloadClient):
        embed.AddField("Download Client", grabMessage.DownloadClientName)

    // Send webhook
    payload = new DiscordPayload
    {
        Username = Settings.Username ?? "Prowlarr",
        Embeds = [embed]
    }

    HttpClient.Post(Settings.WebhookUrl, JSON.Serialize(payload))
```

## Custom Script Notification

```
Settings:
├── Path: string (script path)
├── Arguments: string (optional)

Environment Variables Set:
├── prowlarr_eventtype: "Grab" | "Health" | "ApplicationUpdate"
├── prowlarr_release_title: (for grabs)
├── prowlarr_release_indexer: (for grabs)
├── prowlarr_release_size: (for grabs)
├── prowlarr_health_issue_type: (for health)
├── prowlarr_health_issue_message: (for health)
├── prowlarr_update_previousversion: (for updates)
├── prowlarr_update_newversion: (for updates)

Execution:
├── Script executed synchronously
├── Timeout: 30 seconds
├── Exit code 0 = success
├── Non-zero = logged as failure
```

## Webhook Payload Examples

### Grab Event

```json
{
  "eventType": "Grab",
  "instanceName": "Prowlarr",
  "applicationUrl": "http://localhost:9696",
  "release": {
    "guid": "abc-123",
    "title": "Movie.2024.1080p.BluRay.x264",
    "indexer": "MyIndexer",
    "indexerId": 1,
    "size": 15000000000,
    "protocol": "torrent",
    "downloadUrl": "http://...",
    "infoUrl": "http://...",
    "categories": [
      { "id": 2040, "name": "Movies/HD" }
    ],
    "seeders": 100,
    "leechers": 50
  },
  "downloadClient": "qBittorrent",
  "downloadClientName": "qBittorrent",
  "downloadId": "abc123def456"
}
```

### Health Event

```json
{
  "eventType": "Health",
  "instanceName": "Prowlarr",
  "level": "Warning",
  "message": "Indexer is unavailable",
  "type": "IndexerStatusCheck",
  "wikiUrl": "https://wiki.servarr.com/..."
}
```

### Application Update Event

```json
{
  "eventType": "ApplicationUpdate",
  "instanceName": "Prowlarr",
  "message": "Prowlarr updated from 1.0.0 to 1.1.0",
  "previousVersion": "1.0.0",
  "newVersion": "1.1.0"
}
```

## Tag-Based Filtering

```
Notification tags filter which indexers trigger notifications:

Example:
├── Notification "Premium Alerts" has tag [1] ("Premium")
├── Indexer "PrivateTracker" has tags [1, 2]
├── Indexer "PublicTracker" has tag [3]

Result:
├── "Premium Alerts" receives events from "PrivateTracker"
├── "Premium Alerts" does NOT receive events from "PublicTracker"
```

## Configuration UI

### Notification Settings

```
General:
├── Name: string
├── Tags: int[] (filter by indexer tags)

Triggers:
├── On Grab: bool
├── Include Manual Grabs: bool (if On Grab enabled)
├── On Health Issue: bool
├── Include Health Warnings: bool (if On Health Issue enabled)
├── On Health Restored: bool
├── On Application Update: bool

Provider-Specific:
├── [Provider-dependent settings]
```
