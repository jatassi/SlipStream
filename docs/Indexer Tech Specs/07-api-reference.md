# API Reference

## Overview

Prowlarr exposes two API systems:
1. **REST API (v1)**: JSON-based API for UI and integrations
2. **Newznab/Torznab API**: XML-based API for search/download compatibility

## Authentication

### REST API Authentication

All REST API requests require authentication via API key:

```
Methods (in order of precedence):
1. HTTP Header: X-Api-Key: {apikey}
2. Query Parameter: ?apikey={apikey}
3. Bearer Token: Authorization: Bearer {apikey}
```

### Newznab API Authentication

```
Query Parameter: ?apikey={apikey}
```

## REST API Endpoints

### Search

#### Search Releases

```
GET /api/v1/search

Query Parameters:
  query: string (search term)
  type: string (search | tvsearch | movie | music | book)
  indexerIds: int[] (optional, specific indexers)
  categories: int[] (Newznab category IDs)
  limit: int (default: 100)
  offset: int (default: 0)

Response: ReleaseResource[]
```

#### Grab Release

```
POST /api/v1/search

Body: ReleaseResource
  {
    "guid": "release-guid",
    "indexerId": 1
  }

Response: ReleaseResource
```

#### Bulk Grab

```
POST /api/v1/search/bulk

Body: ReleaseResource[]
Response: ReleaseResource[]
```

### Indexers

#### List Indexers

```
GET /api/v1/indexer

Response: IndexerResource[]
```

#### Get Indexer

```
GET /api/v1/indexer/{id}

Response: IndexerResource
```

#### Create Indexer

```
POST /api/v1/indexer

Body: IndexerResource
Response: IndexerResource (201 Created)
```

#### Update Indexer

```
PUT /api/v1/indexer/{id}

Body: IndexerResource
Response: IndexerResource (202 Accepted)
```

#### Delete Indexer

```
DELETE /api/v1/indexer/{id}

Response: 200 OK
```

#### Get Indexer Schema

```
GET /api/v1/indexer/schema

Response: IndexerResource[] (templates with presets)
```

#### Test Indexer

```
POST /api/v1/indexer/test

Body: IndexerResource
Response: 200 OK or validation errors
```

#### Test All Indexers

```
POST /api/v1/indexer/testall

Response: ProviderTestAllResult[]
```

#### Indexer Statistics

```
GET /api/v1/indexer/stats

Query Parameters:
  startDate: DateTime
  endDate: DateTime
  indexers: string (comma-separated IDs)
  protocols: string (usenet|torrent)
  tags: string (comma-separated)

Response: IndexerStatsResource
```

#### Indexer Status

```
GET /api/v1/indexer/status

Response: IndexerStatusResource[]
```

### Applications

#### List Applications

```
GET /api/v1/applications

Response: ApplicationResource[]
```

#### CRUD Operations

```
GET    /api/v1/applications/{id}
POST   /api/v1/applications
PUT    /api/v1/applications/{id}
DELETE /api/v1/applications/{id}
```

#### Schema and Testing

```
GET  /api/v1/applications/schema
POST /api/v1/applications/test
POST /api/v1/applications/testall
```

### Download Clients

#### CRUD Operations

```
GET    /api/v1/downloadclient
GET    /api/v1/downloadclient/{id}
POST   /api/v1/downloadclient
PUT    /api/v1/downloadclient/{id}
DELETE /api/v1/downloadclient/{id}
```

#### Schema and Testing

```
GET  /api/v1/downloadclient/schema
POST /api/v1/downloadclient/test
POST /api/v1/downloadclient/testall
```

### Notifications

#### CRUD Operations

```
GET    /api/v1/notification
GET    /api/v1/notification/{id}
POST   /api/v1/notification
PUT    /api/v1/notification/{id}
DELETE /api/v1/notification/{id}
```

#### Schema and Testing

```
GET  /api/v1/notification/schema
POST /api/v1/notification/test
POST /api/v1/notification/testall
```

### History

#### Query History

```
GET /api/v1/history

Query Parameters:
  page: int (default: 1)
  pageSize: int (default: 10)
  sortKey: string (date, indexerId, etc.)
  sortDirection: string (ascending | descending)
  eventType: string[] (optional)
  successful: bool (optional)
  downloadId: string (optional)
  indexerIds: int[] (optional)

Response: PagingResource<HistoryResource>
```

#### History Since Date

```
GET /api/v1/history/since

Query Parameters:
  date: DateTime
  eventType: string (optional)

Response: HistoryResource[]
```

### Tags

#### CRUD Operations

```
GET    /api/v1/tag
GET    /api/v1/tag/{id}
POST   /api/v1/tag
PUT    /api/v1/tag/{id}
DELETE /api/v1/tag/{id}
```

#### Tag Details

```
GET /api/v1/tag/detail
GET /api/v1/tag/detail/{id}

Response: TagDetailsResource (includes usage counts)
```

### Commands

#### Queue Command

```
POST /api/v1/command

Body:
{
  "name": "ApplicationIndexerSync",
  "body": {}
}

Response: CommandResource (201 Created)
```

#### List Commands

```
GET /api/v1/command

Response: CommandResource[]
```

#### Cancel Command

```
DELETE /api/v1/command/{id}

Response: 200 OK
```

### Configuration

#### Host Configuration

```
GET /api/v1/config/host
PUT /api/v1/config/host

Response: HostConfigResource
```

#### UI Configuration

```
GET /api/v1/config/ui
PUT /api/v1/config/ui

Response: UiConfigResource
```

#### Download Client Configuration

```
GET /api/v1/config/downloadclient
PUT /api/v1/config/downloadclient

Response: DownloadClientConfigResource
```

### System

#### System Status

```
GET /api/v1/system/status

Response: SystemResource
{
  "appName": "Prowlarr",
  "version": "1.0.0.0",
  "buildTime": "2024-01-01T00:00:00Z",
  "isDebug": false,
  "isProduction": true,
  "isAdmin": true,
  "isDocker": false,
  "branch": "main",
  "authentication": "forms",
  "databaseType": "sqlite",
  "startTime": "2024-01-01T00:00:00Z"
}
```

#### Shutdown/Restart

```
POST /api/v1/system/shutdown
POST /api/v1/system/restart

Response: { "shuttingDown": true } or { "restarting": true }
```

#### Health Check

```
GET /api/v1/health

Response: HealthResource[]
```

#### Backup

```
GET    /api/v1/system/backup
DELETE /api/v1/system/backup/{id}
POST   /api/v1/system/backup/restore/{id}
POST   /api/v1/system/backup/restore/upload (multipart)
```

#### Tasks

```
GET /api/v1/system/task
GET /api/v1/system/task/{id}

Response: TaskResource[]
```

### Logs

#### Query Logs

```
GET /api/v1/log

Query Parameters:
  page: int
  pageSize: int
  sortKey: string
  sortDirection: string
  level: string (debug|info|warn|error|fatal)

Response: PagingResource<LogResource>
```

#### Log Files

```
GET    /api/v1/log/file
GET    /api/v1/log/file/{id}
DELETE /api/v1/log/file/{id}
```

## Newznab/Torznab API

### Endpoints

```
Standard endpoint:
GET /api/v1/indexer/{indexerId}/newznab

Legacy endpoint:
GET /{indexerId}/api
```

### Request Parameters

#### Capabilities Query

```
GET ?t=caps

Response: XML capabilities document
```

#### Search Query

```
GET ?t=search&q={query}

Additional Parameters:
  apikey: string (required)
  cat: string (comma-separated category IDs)
  limit: int (max results)
  offset: int (pagination)
```

#### TV Search

```
GET ?t=tvsearch

Parameters:
  q: string (query)
  season: int
  ep: int (or string for date-based shows)
  imdbid: string (tt1234567)
  tvdbid: int
  rid: int (TVRage)
  tvmazeid: int
  traktid: int
  tmdbid: int
```

#### Movie Search

```
GET ?t=movie

Parameters:
  q: string
  imdbid: string
  tmdbid: int
  traktid: int
  year: int
  genre: string
```

#### Music Search

```
GET ?t=music

Parameters:
  q: string
  artist: string
  album: string
  track: string
  label: string
  year: int
  genre: string
```

#### Book Search

```
GET ?t=book

Parameters:
  q: string
  author: string
  title: string
  publisher: string
  year: int
```

### Download Endpoint

```
GET /api/v1/indexer/{indexerId}/download
GET /{indexerId}/download

Parameters:
  link: string (encrypted download URL)
  file: string (release title, URL-encoded)
  apikey: string
```

### Response Format

#### Capabilities Response

```xml
<?xml version="1.0" encoding="UTF-8"?>
<caps>
  <server title="Prowlarr"/>
  <limits max="100" default="50"/>
  <searching>
    <search available="yes" supportedParams="q"/>
    <tv-search available="yes" supportedParams="q,season,ep,imdbid,tvdbid"/>
    <movie-search available="yes" supportedParams="q,imdbid,tmdbid"/>
    <music-search available="yes" supportedParams="q,artist,album"/>
    <book-search available="yes" supportedParams="q,author,title"/>
  </searching>
  <categories>
    <category id="2000" name="Movies">
      <subcat id="2040" name="Movies/HD"/>
    </category>
  </categories>
</caps>
```

#### Search Response

```xml
<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:newznab="..." xmlns:torznab="...">
  <channel>
    <title>Prowlarr - Indexer Name</title>
    <description>Search Results</description>
    <newznab:response offset="0" total="100"/>
    <item>
      <title>Release Title</title>
      <guid>unique-guid</guid>
      <link>download-url</link>
      <comments>info-page-url</comments>
      <pubDate>Mon, 01 Jan 2024 12:00:00 +0000</pubDate>
      <size>1500000000</size>
      <enclosure url="download-url" length="1500000000" type="..."/>
      <newznab:attr name="category" value="2040"/>
      <newznab:attr name="size" value="1500000000"/>
      <torznab:attr name="seeders" value="100"/>
      <torznab:attr name="peers" value="150"/>
    </item>
  </channel>
</rss>
```

#### Error Response

```xml
<?xml version="1.0" encoding="UTF-8"?>
<error code="100" description="Incorrect user credentials"/>

Error Codes:
100 - Incorrect credentials
101 - Account suspended
102 - No permission
200 - Missing parameter
201 - Incorrect parameter
300 - No such function
500 - Request limit reached
```

### HTTP Status Codes

| Code | Meaning |
|------|---------|
| 200 | Success |
| 400 | Bad request (missing/invalid parameters) |
| 401 | Unauthorized (invalid API key) |
| 410 | Gone (indexer disabled) |
| 429 | Too many requests (rate limited) |
| 500 | Internal server error |

## Resource Models

### IndexerResource

```json
{
  "id": 1,
  "name": "Indexer Name",
  "implementation": "Newznab",
  "configContract": "NewznabSettings",
  "enable": true,
  "priority": 25,
  "appProfileId": 1,
  "downloadClientId": null,
  "protocol": "torrent",
  "privacy": "private",
  "supportsRss": true,
  "supportsSearch": true,
  "supportsRedirect": true,
  "supportsPagination": true,
  "capabilities": {
    "supportsRawSearch": false,
    "searchParams": ["q"],
    "tvSearchParams": ["q", "season", "ep", "imdbid"],
    "movieSearchParams": ["q", "imdbid", "tmdbid"],
    "categories": [
      { "id": 2040, "name": "Movies/HD" }
    ]
  },
  "fields": [
    { "name": "baseUrl", "value": "https://..." },
    { "name": "apiKey", "value": "***" }
  ],
  "tags": [1, 2]
}
```

### ReleaseResource

```json
{
  "guid": "unique-guid",
  "title": "Release Title",
  "sortTitle": "release title",
  "indexerId": 1,
  "indexer": "Indexer Name",
  "protocol": "torrent",
  "size": 1500000000,
  "age": 2,
  "ageHours": 48.5,
  "publishDate": "2024-01-01T12:00:00Z",
  "downloadUrl": "https://...",
  "infoUrl": "https://...",
  "categories": [
    { "id": 2040, "name": "Movies/HD" }
  ],
  "seeders": 100,
  "leechers": 50,
  "magnetUrl": "magnet:?xt=...",
  "infoHash": "abc123...",
  "imdbId": 1234567,
  "tmdbId": 12345,
  "tvdbId": 123456,
  "indexerFlags": ["freeleech"]
}
```

### HistoryResource

```json
{
  "id": 1,
  "indexerId": 1,
  "indexer": "Indexer Name",
  "date": "2024-01-01T12:00:00Z",
  "eventType": "releaseGrabbed",
  "successful": true,
  "data": {
    "query": "search term",
    "categories": [2040],
    "elapsedTime": 1234.5
  }
}
```

### CommandResource

```json
{
  "id": 1,
  "name": "ApplicationIndexerSync",
  "commandName": "Application Indexer Sync",
  "message": "Syncing indexers...",
  "priority": "normal",
  "status": "started",
  "queued": "2024-01-01T12:00:00Z",
  "started": "2024-01-01T12:00:01Z",
  "ended": null,
  "duration": null,
  "trigger": "manual"
}
```

### HealthResource

```json
{
  "source": "IndexerStatusCheck",
  "type": "warning",
  "message": "Indexer is unavailable",
  "wikiUrl": "https://wiki.servarr.com/..."
}
```

## Pagination

### Request

```
GET /api/v1/resource?page=1&pageSize=10&sortKey=date&sortDirection=descending
```

### Response

```json
{
  "page": 1,
  "pageSize": 10,
  "sortKey": "date",
  "sortDirection": "descending",
  "totalRecords": 100,
  "records": [...]
}
```

## SignalR Real-Time Updates

### Connection

```
Hub URL: /signalr/messages
Transport: WebSocket (preferred) or Long Polling
```

### Messages

```
Resource Created:
{
  "name": "indexer",
  "action": "created",
  "resource": {...}
}

Resource Updated:
{
  "name": "indexer",
  "action": "updated",
  "resource": {...}
}

Resource Deleted:
{
  "name": "indexer",
  "action": "deleted",
  "resource": { "id": 1 }
}

Command Status:
{
  "name": "command",
  "action": "updated",
  "resource": {...}
}

Health Check:
{
  "name": "health",
  "action": "sync",
  "resource": [...]
}
```
