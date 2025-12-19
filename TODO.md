# SlipStream Development TODO

This file tracks development progress across sessions. Mark items `[x]` when complete.

---

## Phase 0: Project Setup

- [x] Create `ARCHITECTURE.md` in repository
- [x] Create `TODO.md` in repository (this file)
- [x] Initialize Go module (`go mod init`)
- [x] Set up project directory structure
- [x] Initialize Vite + React frontend in `web/`
- [x] Configure shadcn/ui with Base UI
- [x] Set up Tailwind CSS v4
- [x] Configure development workflow (Makefile)

---

## Phase 1: Core Infrastructure

- [x] SQLite database setup with goose migrations
- [x] sqlc configuration and initial queries
- [x] Echo HTTP server skeleton
- [x] Configuration loading (Viper)
- [x] Structured logging (zerolog)
- [x] Single-user authentication (password protection)
- [x] WebSocket infrastructure for real-time updates

---

## Phase 2: Library Management

- [x] Media file scanning and detection
- [x] Movie library data model and CRUD
- [x] TV show library data model and CRUD (series/seasons/episodes)
- [x] File renaming and organization
- [x] Quality profiles system

---

## Testing

- [x] Scanner/parser unit tests (filename parsing, extension detection)
- [x] Quality profile unit tests (matching, upgrades, serialization)
- [x] Organizer/templates unit tests (naming templates, path generation)
- [x] Movie service integration tests
- [x] TV service integration tests
- [x] API endpoint integration tests
- [x] Test coverage reporting

---

## Frontend Development

- [ ] Dashboard/home page (system status, recent activity)
- [ ] Movies list and detail views
- [ ] Series list and detail views (with season/episode hierarchy)
- [ ] Add movie/series workflow (search + add)
- [ ] Quality profiles management UI
- [ ] Root folders management UI
- [ ] Settings pages
- [ ] Indexer configuration UI
- [ ] Download client configuration UI
- [ ] Queue/activity view
- [ ] History view
- [ ] Real-time updates via WebSocket

---

## Phase 3: Metadata Integration

- [x] TMDB API client
- [x] TVDB API client
- [x] Metadata fetching and caching
- [x] Artwork downloading

---

## Phase 4: Indexer Integration

- [ ] Torznab protocol client
- [ ] Newznab protocol client
- [ ] Indexer management (add/edit/delete/test)
- [ ] Search functionality (automatic + interactive)

---

## Phase 5: Download Client Integration

- [ ] qBittorrent API client
- [ ] Transmission API client
- [ ] SABnzbd API client
- [ ] Download queue management
- [ ] Completed download handling

---

## Phase 6: Import & Automation

- [ ] Completed download import
- [ ] Automatic search scheduling
- [ ] Background job system (scheduler)
- [ ] Activity history tracking

---

## Future (Deferred)

- [ ] Multi-user support
- [ ] Notifications (webhooks, Discord, Telegram, etc.)
- [ ] Docker distribution
- [ ] Music support (Lidarr functionality)
- [ ] Mobile app (native or PWA)

---

## Notes

- This file is the source of truth for development progress
- Update this file as tasks are completed
- See `ARCHITECTURE.md` for technical details
