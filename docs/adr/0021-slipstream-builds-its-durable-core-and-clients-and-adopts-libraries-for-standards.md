# SlipStream builds its durable core and clients, and adopts libraries for standards

SlipStream takes a library wherever the work is a published standard, a file format or accessibility. It builds the code that carries its own guarantees: the durable core and every client for an external service. The standards and formats are WebAuthn (go-webauthn), age, zstd, minisign, BLAKE3 (zeebo/blake3), argon2id, SMTP (go-mail), STUN (pion/stun), HTML scraping (goquery, cascadia) and YAML (goccy/go-yaml for Cardigann, go.yaml.in/yaml/v3 for config). Accessible UI primitives come from Base UI, forms from TanStack Form and icons from Hugeicons' free set. Small leaf utilities are adopted too: cenkalti/backoff, x/time/rate, gofrs/flock, lumberjack and x/sync.

SlipStream builds the job queue, scheduler, phase runner, Worker outbox, intent sync and event hub on its own sqlc tables. It also hand-writes the clients for TMDB, TVDB, OMDb, Plex, plex.tv, Jellyfin, Emby, notifiers, Download Clients and Subtitle Providers, all over one shared retry and rate-limit layer. golift/starr for Radarr and Sonarr is the one wrapped client, because it already takes an injected base URL and HTTP client. The release-name parser and the naming grammar are ports of Sonarr's and Radarr's, tested against their GPL-3 parser fixtures. The Cardigann template evaluator is ported from Prowlarr, never from Jackett, which is GPL-2.0-only.

Three things rule libraries out of that second group. ADR 0015's rollback restores a Backup over the database, which would erase a workflow engine's checkpoint. ADR 0019's Reconciling and ADR 0003's lockstep need one gate that every job start passes. Most client libraries hardcode hosts or hold a package-global HTTP client, which breaks ADR 0014's `Endpoints` and wire-level fakes. A third-party parser also breaks ADR 0006, because TRaSH Custom Formats test the values Sonarr and Radarr parse. Decided by the maintainer on 2026-10-11 in the [library research](../research/oss-libraries.md), which holds every pick with its evidence.

## Considered Options

- **River, backlite or goqite for jobs.** Rejected. River's SQLite driver calls itself early testing and pulls pgx. Each queue brings its own migrations beside goose (ADR 0013), and none has the Reconciling gate.
- **DBOS, go-workflows or Temporal for Moves, Handover and updates.** Rejected. Their checkpoints live in the database ADR 0015 may restore, and their recovery is pinned to the binary version that started the run.
- **Generated SDKs (plexgo, jellyfin-go) and shoutrrr.** Rejected. They run to hundreds of thousands of generated lines, or they hardcode provider hosts, for about 25 calls and a dozen payloads.
- **A fully custom build, crypto and UI primitives included.** Rejected. Writing COSE, CBOR, focus traps and combobox semantics again risks security and WCAG failures that the libraries have already fixed.
- **Lucide icons.** Replaced by Hugeicons at the maintainer's pick. Only the MIT free set ships.

## Consequences

- The durable core is about 1.2k to 1.8k lines of Go, and the Sonarr and Radarr parser port is 4k to 6k. Each hand-written client is a few hundred lines with its own fake.
- The license allow-list in CI also denies BSD-4-Clause, AGPL, GPL-2.0-only and unlicensed code. That rules out bogdanfinn/tls-client, configarr and vendoring Prowlarr's unlicensed Indexers repository.
- Jackett is never a porting source. AGENTS.md says so, because CI can't detect ported code.
