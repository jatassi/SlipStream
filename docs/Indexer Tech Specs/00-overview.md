# Prowlarr Technical Specification

## Overview

Prowlarr is an indexer manager/proxy application that serves as a centralized interface between usenet/torrent indexers and media management applications (Sonarr, Radarr, Lidarr, Readarr, etc.).

## Core Functionality

1. **Indexer Management**: Configure and manage connections to 500+ usenet and torrent indexers
2. **Search Aggregation**: Perform searches across multiple indexers simultaneously
3. **Application Sync**: Push indexer configurations to connected media applications
4. **Download Orchestration**: Send releases to download clients (torrent/usenet)
5. **Protocol Translation**: Expose any indexer via standardized Newznab/Torznab APIs

## Document Index

| Document | Description |
|----------|-------------|
| [01-architecture.md](01-architecture.md) | System architecture and component overview |
| [02-indexer-system.md](02-indexer-system.md) | Indexer implementation patterns and protocols |
| [03-cardigann-definitions.md](03-cardigann-definitions.md) | YAML-based dynamic indexer definitions |
| [04-search-system.md](04-search-system.md) | Search workflow and result aggregation |
| [05-application-sync.md](05-application-sync.md) | Integration with Sonarr/Radarr/Lidarr |
| [06-download-clients.md](06-download-clients.md) | Download client integration |
| [07-api-reference.md](07-api-reference.md) | REST API endpoints and Newznab compatibility |
| [08-authentication.md](08-authentication.md) | Authentication mechanisms |
| [09-data-model.md](09-data-model.md) | Database schema and data persistence |
| [10-notifications.md](10-notifications.md) | Event-driven notification system |

## Key Concepts

### Indexer Types

| Type | Protocol | Description |
|------|----------|-------------|
| Usenet | Newznab | NZB file indexers using Newznab XML API |
| Torrent | Torznab | Torrent indexers using Torznab XML API (Newznab extension) |
| RSS | RSS/Atom | Generic RSS feed parsing |

### Privacy Levels

| Level | Description |
|-------|-------------|
| Public | Open access, no authentication required |
| Semi-Private | Registration required, some restrictions |
| Private | Invite-only, strict ratio requirements |

### Supported Protocols

- **HTTP/HTTPS**: Primary communication protocol
- **Newznab API**: Standardized XML-based search/download API
- **Torznab API**: Torrent extension of Newznab
- **JSON-RPC**: Used by some download clients
- **XMLRPC**: Used by rTorrent
- **WebSocket/SignalR**: Real-time UI updates

## System Requirements

### Functional Requirements

1. Support multiple concurrent indexer connections
2. Aggregate search results with deduplication
3. Map indexer categories to standardized Newznab categories
4. Handle authentication for private indexers
5. Rate limit requests to protect indexer accounts
6. Proxy downloads through the application
7. Sync indexer configurations to external applications
8. Track search and download history

### Non-Functional Requirements

1. Support SQLite and PostgreSQL databases
2. Cross-platform operation (Windows, Linux, macOS, Docker)
3. RESTful API with JSON responses
4. Real-time updates via WebSocket
5. Graceful handling of indexer failures
6. Configurable logging levels

## High-Level Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     Prowlarr Application                     │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐  │
│  │   REST API  │  │  SignalR    │  │  Newznab/Torznab   │  │
│  │   (JSON)    │  │  (WebSocket)│  │    Endpoints       │  │
│  └──────┬──────┘  └──────┬──────┘  └─────────┬───────────┘  │
│         │                │                    │              │
│  ┌──────▼────────────────▼────────────────────▼───────────┐ │
│  │                  Business Logic Layer                   │ │
│  │  ┌───────────┐ ┌───────────┐ ┌───────────┐ ┌─────────┐ │ │
│  │  │  Search   │ │  Indexer  │ │   App     │ │Download │ │ │
│  │  │  Service  │ │  Service  │ │   Sync    │ │ Service │ │ │
│  │  └───────────┘ └───────────┘ └───────────┘ └─────────┘ │ │
│  └─────────────────────────┬───────────────────────────────┘ │
│                            │                                 │
│  ┌─────────────────────────▼───────────────────────────────┐ │
│  │                    Data Access Layer                     │ │
│  │              (Repository Pattern + Dapper)               │ │
│  └─────────────────────────┬───────────────────────────────┘ │
└────────────────────────────┼─────────────────────────────────┘
                             │
              ┌──────────────┼──────────────┐
              ▼              ▼              ▼
         ┌────────┐    ┌──────────┐   ┌──────────┐
         │ SQLite │    │PostgreSQL│   │ External │
         └────────┘    └──────────┘   │ Services │
                                      └──────────┘
```

## External Integrations

```
                    ┌──────────────────┐
                    │     Prowlarr     │
                    └────────┬─────────┘
                             │
        ┌────────────────────┼────────────────────┐
        │                    │                    │
        ▼                    ▼                    ▼
┌───────────────┐   ┌───────────────┐   ┌───────────────┐
│   Indexers    │   │  Applications │   │   Download    │
│               │   │               │   │   Clients     │
│ - Newznab     │   │ - Sonarr      │   │ - qBittorrent │
│ - Torznab     │   │ - Radarr      │   │ - SABnzbd     │
│ - Custom      │   │ - Lidarr      │   │ - Transmission│
│ - Cardigann   │   │ - Readarr     │   │ - NzbGet      │
└───────────────┘   └───────────────┘   └───────────────┘
```

## Glossary

| Term | Definition |
|------|------------|
| **Indexer** | A service that indexes content (torrents or NZBs) and provides search capability |
| **Newznab** | Standardized API for usenet indexers |
| **Torznab** | Extension of Newznab for torrent indexers |
| **Cardigann** | YAML-based meta-indexer system for defining custom indexers |
| **Release** | A single piece of indexed content (movie, episode, album, etc.) |
| **Grab** | The action of sending a release to a download client |
| **Application** | External media management software (Sonarr, Radarr, etc.) |
| **Category Mapping** | Translation between indexer-specific categories and Newznab standard categories |
| **App Profile** | Configuration template for synchronizing indexers to applications |
