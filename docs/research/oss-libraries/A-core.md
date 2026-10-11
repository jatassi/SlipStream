# A. Durability and runtime core: libraries vs custom

> Spec numbers follow the feature graph as renumbered on 2026-10-11, with seven foundation specs (S01 to S07) ahead of the walking skeleton (S09). "S01-S09" means the foundations plus the skeleton.


Scope: job queue, scheduler, durable multi-step operations, Worker outbox and intent sync, event bus and SSE, retry/rate/breaker, bootstrap config, log rotation, IDs and clock, instance lock, filesystem primitives, and the Fake world's fault and conformance tooling.

## How this was checked

- **Versions and dates**: Go module proxy (`proxy.golang.org/<mod>/@latest`, plus `@v/main.info` / `@v/master.info` for the last commit on the default branch), fetched 2026-10-11. **Licenses and imported-by counts**: pkg.go.dev, fetched 2026-10-11. **READMEs, go.mod, CHANGELOGs**: raw.githubusercontent.com, fetched 2026-10-11. Everything marked "verified" was read this session from one of those sources.
- **Not available**: the GitHub API and github.com pages returned 403 in this session, so star counts, open-issue counts and contributor counts were **not retrieved**. Maintenance signals below use release cadence, last commit date and pkg.go.dev "imported by" counts instead.
- **Go 1.27 changes the answer twice**. Verified from go.dev/doc/go1.27 and pkg.go.dev: (1) a new standard **`uuid`** package with `New`, `NewV4`, `NewV7`, `Parse`; (2) **`httptest.NewTestServer`**, a test server on "an in-memory network implementation ... suitable for use with the testing/synctest package", plus `synctest.Sleep`. Go 1.26 added **`slog.NewMultiHandler`** (verified, go.dev/doc/go1.26).

## The finding that shapes the rest

The operations the specs call durable are **domain records the user sees**, not opaque jobs. Examples: an update's phases (S57, ADR 0015), a Move's switch (S48, ADR 0016), a Handover (S50, ADR 0019), Reconciling (S49), a Transfer (S31, ADR 0004) and a Conversion (S44, ADR 0007). Each has a UI, external side effects that can't be replayed (tracker `stopped`/`started`, a binary swap, a key change), and recovery rules that ADRs already specify. Three ADR facts rule out a general workflow engine as their store:

1. **The update's own state can't live in the database it may roll back.** ADR 0015: if the health trial fails twice, the new binary "restores that Backup and puts the old binary back". An engine that checkpoints in the main SQLite file loses its own progress record at exactly that step. The update run needs a small state file in the data dir, outside the DB, and written atomically.
2. **Recovery crosses binary versions.** The old binary writes "swap" and the new binary resumes "health trial". DBOS recovers workflows per application version, and go-workflows replays deterministic history, so both pin recovery to the code that started the run. ADR 0003 freezes only the handshake and update path. It does not freeze a workflow engine's history format.
3. **Reconciling and lockstep need one admission gate.** ADR 0019: "starts nothing new and cancels in-flight jobs from the Backup". ADR 0003: an outdated Worker "finishes in-flight jobs ... but starts nothing new". So every job start in the app has to pass one gate. That is simple when the queue is ours and awkward to retrofit onto an engine's dispatcher.

So the core recommendation is to **build a small durable core**: a job table, a phase-runner and a scheduler (roughly 1.2 to 1.8k lines of Go in total), on sqlc and goose. Use libraries only at the leaves, where they are small, have no dependencies and are easy to fake: `x/time/rate`, `cenkalti/backoff/v5`, `gofrs/flock`, lumberjack, the std `uuid`, and `x/sync`.

---

## 1. Job queue and background work

- **Specs**: S13 (task runner, running-tasks pill), S19 (metadata refresh), S22 (scan), S33 (import), S34 (automatic search, Backlog, RSS), S37 (notification delivery with backoff), S49 (cancel in-flight jobs on Reconciling), S57 (drain for 10 min), S01-S09 (lays the table down).
- **Current plan**: unspecified. ADR 0013 fixes SQLite + sqlc + goose per module. ADR 0003 fixes Worker statelessness: the queue lives on the Controller, and Workers execute through RPC.

| Candidate | Repo | License | Latest release | Last commit | Signal | Verified |
|---|---|---|---|---|---|---|
| River + `riversqlite` | github.com/riverqueue/river | MPL-2.0 | v0.49.0, 2026-10-05 | 2026-10 | Very active, company-backed; 269 importers (pkg.go.dev). SQLite driver added v0.23.0 (2025-06-04). The package doc still says "currently in early testing ... minimal real world use" and recommends `SetMaxOpenConns(1)`. Root go.mod requires `jackc/pgx/v5` and `robfig/cron/v3`. Has its own migration line (v8 is SQLite-only). | yes |
| backlite | github.com/mikestefanello/backlite | MIT | v0.8.0, 2026-10-06 | 2026-10 | Single maintainer, 15 importers. SQLite-native, generics-typed tasks, tx insert, retries, retention, no polling (tracks next run in memory), web UI. Schema installed by `client.Install()`. go.mod pulls goquery (UI) and modernc. About 2.1k lines without tests. | yes |
| goqite | github.com/maragudk/goqite (module `maragu.dev/goqite`) | MIT | v0.4.0, 2025-11-20 | 2026-08 | Single maintainer, 2 importers. SQS-style: visibility timeout, `Extend`, max receive, `jobs` runner. No non-test deps. About 1k lines. | yes |
| neoq | github.com/acaloiaro/neoq | MIT | v0.72.2, 2026-10-04 | unverified | Backends: memory, Postgres, Redis only (no SQLite). README says the repo moved to code.adriano.fyi. | yes |
| Asynq | github.com/hibiken/asynq | MIT | v0.26.0, 2026-02-03 | unverified | Redis required. Out. | yes (release) |

**Verdict: build**, borrowing goqite's visibility-timeout/extend design and River's rescuer, unique-key and snooze semantics.

The custom version is one `job` table (kind, args JSON, state, attempt, run_at, lease_until, unique_key, error), a sqlc query set, a poller with a wakeup channel (single process, so inserts can nudge the poller directly, as backlite does), a rescuer for expired leases, and the admission gate from the finding above. Estimate: 500 to 800 lines plus tests. Risk: lease and rescue edge cases. Those are well understood, and `synctest` covers them deterministically.

Costs of the alternatives:
- **River** brings a second migration system to fold into goose (possible: `river migrate-get` dumps SQL), a single-connection pool recommendation that fights the app's WAL read pool, pgx in the module graph, and an SQLite driver its own docs call early. MPL-2.0 is fine for GPL-3.0.
- **backlite** is the best fallback (**wrap**) if the maintainer prefers a dependency. It is SQLite-native, MIT, active and tx-aware, with a small API. But it has one maintainer and its own schema installer.

**Criticality**: critical infra. A lost import or notification job loses data, and the Reconciling gate is a ratio-safety rule.

**ADR conflicts**: River's migrator and backlite's `Install()` sit beside goose per module (ADR 0013) unless their SQL is copied into a goose migration.

## 2. Scheduler for Scheduled tasks

- **Specs**: S13 (runner, Tasks page with last run, next run, running state, Run Now), S18 (storage check), S22 (daily scan), S34 (Backlog search and RSS intervals), S49 (Backups on a schedule), S57 (update check).
- **Current plan**: unspecified. Parity P-229 says v1 task schedules were **fixed intervals** apart from search and RSS intervals, so nothing needs cron syntax.

| Candidate | Repo | License | Latest release | Last commit | Signal | Verified |
|---|---|---|---|---|---|---|
| gocron v2 | github.com/go-co-op/gocron | MIT | v2.22.0, 2026-07-09 | unverified | Active, 502 importers. In-memory schedule. Deps: `jonboulle/clockwork`, `robfig/cron/v3`, `google/uuid`. Singleton mode, limits, event listeners. v1 used it. | yes |
| robfig/cron v3 | github.com/robfig/cron | MIT | v3.0.1, 2020-01-04 | 2021-01 | Unmaintained since 2021, but very widely used (5,585 importers). | yes |
| gronx | github.com/adhocore/gronx | MIT | v1.20.5, 2026-09-27 | unverified | Cron-expression parser and next-tick only, zero deps, 126 importers. | yes |
| go-quartz | github.com/reugn/go-quartz | MIT | v0.15.2, 2025-09-12 | unverified | Quartz-style, in-memory by default. | yes (release) |

**Verdict: build.** The tasks need state no scheduler library keeps:
- `last_run_at` persisted, so a daily scan doesn't re-fire on every restart;
- a "next run" derived as last run + interval for the Tasks page;
- Run Now, and a per-task singleton;
- a pause behind the admission gate during Reconciling and drain.

A custom loop is one table, one goroutine and a timer reset to the earliest due task, then enqueue into the job table from section 1. Estimate: about 150 to 250 lines. Under `synctest` it needs no clock injection.

gocron would still need that table, adds clockwork as a second clock to reason about, and pulls an unmaintained cron parser. If cron expressions are ever wanted, **adopt gronx** for parsing only.

**Criticality**: important. A missed task is recoverable. A task firing during Reconciling is not, which is the gate's job.

**ADR conflicts**: none.

## 3. Durable, resumable multi-step operations

- **Specs**: S57 (update phases), S48 (Moves), S50 (Handover), S49 (restore and Reconciling), S31 (Transfers resume), S44 (Conversions), S33 (import review states).
- **Current plan**: ADR 0015 says "stored, resumable phases". ADR 0016 gives the phase order and crash points. ADR 0019 gives the Reconciling state. ADR 0004 gives chunk-verified resume. No engine is named, so custom is the default.

| Candidate | Repo | License | Latest release | Last commit | Signal | Verified |
|---|---|---|---|---|---|---|
| DBOS Transact Go | github.com/dbos-inc/dbos-transact-golang | MIT | v1.6.0, 2026-10-07 | 2026-10 | Company-backed, releases every ~2 weeks. **SQLite is supported**: a `dbos/driver/sqlite` package on modernc exists in every tagged release from v1.0.0 (2026-07-30). The README still pitches Postgres. Recovery is tied to `ApplicationVersion`; there is an opt-in patching system and rewind/fork tools. Runs its own system-DB migrations. go.mod pulls pgx, cobra, viper. | yes |
| go-workflows | github.com/cschleiden/go-workflows | MIT | v1.4.2, 2026-06-05 | 2026-08 (nightly) | Single maintainer. Temporal-style deterministic replay: no `select` or map iteration in workflow code. SQLite backend on modernc **v1.27.0**. Applies its own migrations with golang-migrate. go.mod pulls MySQL, pgx, Redis and OpenTelemetry. | yes |
| Temporal Go SDK | go.temporal.io/sdk | MIT | v1.49.0, 2026-09-14 | unverified | Needs a Temporal server process. Out under "no external services". | yes (release) |
| qmuntal/stateless | github.com/qmuntal/stateless | BSD-2-Clause | v1.8.0, 2026-02-10 | 2026-02 | In-memory state machine with external state storage hooks and DOT graph export. Not durable by itself. 28 importers. | yes |
| looplab/fsm | github.com/looplab/fsm | Apache-2.0 | v1.0.4, 2026-08-27 | unverified | Simple event-driven FSM, in-memory, 683 importers. | yes (release) |

**Verdict: build** a phase-runner, and borrow DBOS's model: a checkpoint per step, recovery on startup by executor, and fork/rewind as an admin escape hatch.

Each operation gets a domain table with a `phase` enum, its own columns, and one idempotent function per phase. A shared driver claims pending operations at startup, runs the current phase, commits the next phase, and publishes an SSE event per transition. The update run uses the same driver over a JSON state file in the data dir (atomic write, section 11), because of the finding's point 1. Estimate: driver 300 to 500 lines. The per-phase code is written either way, since it's the domain logic.

What an engine would cost:
- **DBOS** is now the strongest off-the-shelf option on SQLite. But it adds a second migration owner, a second store of truth for state the UI already reads from domain tables, and version-pinned recovery to work around across binary swaps.
- **go-workflows**' determinism rules and dependency weight are a poor fit.
- **Plain FSM libraries** add little over a Go `switch` on an enum for the mostly linear flows here. Revisit stateless only if a flow grows real branching and the DOT export helps review.

**Criticality**: critical infra. Updates, Moves and Handover are where data loss, ratio loss and "the whole app stops" all live.

**ADR conflicts**:
- DBOS or go-workflows storing update progress in the main DB contradicts the ADR 0015 restore-then-roll-back sequence.
- Their own migrators sit beside goose per module (ADR 0013).
- Version-pinned recovery strains ADR 0003's lockstep model, where only the handshake and update path are frozen.

## 4. Worker outbox and Controller intent sync

- **Specs**: S01-S09 (Node link), S15 (offline Workers, requeue, drain), S28 (seeding from last intent), S31, S49 (Workers report what they hold).
- **Current plan**: ADR 0003 decides the semantics. ADR 0013 decides the transport (ConnectRPC over yamux) and that a Worker's outbox lives in a local SQLite file it may lose.

| Candidate | Repo | License | Latest release | Last commit | Signal | Verified |
|---|---|---|---|---|---|---|
| Watermill | github.com/ThreeDotsLabs/watermill | MIT | v1.5.3, 2026-08-25 | unverified | Mature, 825 importers. A general message router: middleware, retries, poison queue. | yes |
| watermill-sqlite (`wmsqlitemodernc`) | github.com/ThreeDotsLabs/watermill-sqlite | MIT | wmsqlitemodernc v0.1.2, 2025-09-22 | unverified | README: "Beta Version Warning ... not been widely tested in production." Local pub/sub over one SQLite file; does not cross Nodes. | yes |
| watermill-sql v4 | github.com/ThreeDotsLabs/watermill-sql | MIT | v4.1.5, 2026-05-14 | unverified | MySQL/Postgres. | yes (release) |

**Verdict: build.**
- **Outbox**: a Worker-local table of results with a sequence number. The Worker pushes over a ConnectRPC stream and the Controller acks the highest applied sequence. Apply is idempotent by (node, seq).
- **Intent**: level-triggered. The Controller serves a versioned snapshot per Role, the Worker diffs it against what it runs, and the latest intent is stored on the Worker for "keep seeding from its last copy".

Estimate: about 300 to 500 lines, mostly protobuf messages, which the frozen-handshake discipline wants explicit anyway. Watermill would add a second messaging abstraction above ConnectRPC with no transport between Nodes, and its SQLite pub/sub is beta.

**Criticality**: critical infra. A lost import result or a wrong intent costs data or ratio.

**ADR conflicts**: none for build. Watermill doesn't conflict, but it duplicates the Node link's role (ADR 0013).

## 5. In-process event bus feeding SSE

- **Specs**: S01-S09 (all live over SSE), S13 (running-tasks pill, activity indicator), S28, S39 (live pipeline), S14 (live log viewer), S57 (update events).
- **Current plan**: ADR 0013 fixes one SSE stream per session with typed events declared in the OpenAPI document. Huma v2 is decided.

| Candidate | Repo | License | Latest release | Last commit | Signal | Verified |
|---|---|---|---|---|---|---|
| Huma `sse` package | github.com/danielgtaylor/huma (`/v2/sse`) | MIT | v2.39.1, 2026-07-29 | unverified | Already in the stack. `sse.Register(api, op, eventTypeMap, f)` puts each typed event in OpenAPI; `Message{ID int, Data, Retry, Comment}`; flushing handled; `WriteTimeout`. No replay buffer. | yes |
| go-sse | github.com/tmaxmax/go-sse | MIT | v0.11.0, 2025-05-13 | 2026-10 | Server and client, pluggable Provider ("Joe"), Replayer interface for Last-Event-ID replay. Pre-1.0. | yes |
| r3labs/sse v2 | github.com/r3labs/sse | MPL-2.0 | v2.10.0, 2023-01-18 | 2023-01 | Stale. | yes |

**Verdict: build the hub, adopt Huma's `sse` for the wire** (no new dependency).

The hub is a typed publisher with per-subscriber bounded channels. When a subscriber falls behind, it drops that subscriber and sends a "resync" event so the client refetches through TanStack Query. A small ring buffer handles `Last-Event-ID` replay, which is the idea to borrow from go-sse. Estimate: 150 to 250 lines. Adding go-sse would duplicate Huma's typed-event registration, which is what puts the events in OpenAPI and therefore into Orval.

**Criticality**: important.

**ADR conflicts**: a library that writes SSE outside Huma's operation would drop those events from the OpenAPI document (ADR 0013).

## 6. Retry, backoff, rate limiting, circuit breaking

- **Specs**: S19 (TMDB/OMDb 429 with Retry-After), S26 (per-Indexer limits, `requestDelay`, health), S27, S30 (client health), S35, S37 (notification backoff), S18 (Health).
- **Current plan**: ADR 0017: "Credentials, cookies, rate-limit counters and health live on the Controller". Libraries are unspecified.

| Candidate | Repo | License | Latest release | Last commit | Signal | Verified |
|---|---|---|---|---|---|---|
| cenkalti/backoff v5 | github.com/cenkalti/backoff | MIT | v5.0.3, 2025-07-23 | unverified | Zero deps, 524 importers. Generic `Retry[T]`, `Permanent`, `RetryAfter(seconds)`, `WithMaxTries`, `WithMaxElapsedTime`. | yes |
| sethvargo/go-retry | github.com/sethvargo/go-retry | Apache-2.0 | v0.5.0, 2026-10-05 | unverified | Small, composable middleware-style backoffs. | yes (release) |
| avast/retry-go v5 | github.com/avast/retry-go | MIT | v5.0.0, 2025-11-24 | unverified | Option-heavy API. | yes (release) |
| hashicorp/go-retryablehttp | github.com/hashicorp/go-retryablehttp | MPL-2.0 | v0.7.8, 2025-06-18 | unverified | Wraps `http.Client`. Retry policy hidden in a transport layer. | yes (release) |
| x/time/rate | golang.org/x/time | BSD-3-Clause | v0.16.0, 2026-08-19 | unverified | Go team. Token bucket with `Wait(ctx)`. Timer-based, so it works under `synctest`. | yes |
| sony/gobreaker v2 | github.com/sony/gobreaker | MIT | v2.4.0, 2026-01-01 | unverified | In-memory generic breaker, 118 importers. | yes (release) |
| failsafe-go | github.com/failsafe-go/failsafe-go | MIT | v0.9.8, 2026-09-30 | unverified | Retry, breaker, rate limiter, bulkhead and hedging in one; pre-1.0. | yes (release) |

**Verdict**:
- **Adopt `cenkalti/backoff/v5`** for in-call retries. It has zero deps, handles Retry-After natively and is easy to reason about. Durable retries, such as notification attempts across restarts, are job rows (section 1), not in-memory loops.
- **Wrap `x/time/rate`** behind a per-Indexer limiter. It does the spacing, while daily caps and counters are rows on the Controller per ADR 0017.
- **Build the breaker.** Indexer and client health is persisted, UI-visible domain state with escalating disable windows. gobreaker's in-memory state would be a second, invisible copy.
- **Reject go-retryablehttp.** It hides retry inside the transport, and the offline rule wants the HTTP client plain and injectable.

**Criticality**: important. Over-hammering a private tracker is a ratio and account risk, which makes the per-Indexer limiter critical.

**ADR conflicts**: none.

## 7. Bootstrap configuration and the typed `Endpoints` block

- **Specs**: S01-S09 (`Endpoints` block and its lint, data-dir flag, headless), S10 (Fake world refuses to start while any entry points at a real host; one checked file lists every harness override), S14 (port and fallback), S57 (extra release key from local config or env).
- **Current plan**: ADR 0014 fixes a typed `Endpoints` config block and a `Timings` config. User settings live in the DB. The library is unspecified. v1 used viper.

| Candidate | Repo | License | Latest release | Last commit | Signal | Verified |
|---|---|---|---|---|---|---|
| koanf v2 | github.com/knadh/koanf | MIT | v2.3.8, 2026-10-07 | unverified | Modular providers and parsers, 832 importers. Core deps: `go-viper/mapstructure/v2` and `mitchellh/copystructure` (archived upstream, inferred). | yes (deps) |
| viper | github.com/spf13/viper | MIT | v1.21.0, 2025-09-08 | unverified | Heavy deps, case-insensitive keys, global-ish. | yes (release) |
| caarlos0/env v11 | github.com/caarlos0/env | MIT | v11.4.1, 2026-05-01 | unverified | Env to struct by tags, zero deps, 1,037 importers. Has retracted versions for accidental breaks (v11.0.1, v11.2.0, v11.3.0). | yes |
| peterbourgon/ff v4 | github.com/peterbourgon/ff | Apache-2.0 | v4.0.0-beta.1, 2025-08-02 | unverified | Flags-first; v4 still beta. | yes (release) |
| go.yaml.in/yaml v3 / v4 | github.com/yaml/go-yaml | Apache-2.0, MIT | v3.0.5, 2026-07-26; v4.0.0-rc.6, 2026-06-17 | unverified | The YAML org's continuation. `gopkg.in/yaml.v3` README now says "THIS PROJECT IS UNMAINTAINED". | yes |

**Verdict: build** a typed `Config` struct with nested `Endpoints` and `Timings`, std `flag` for the few flags, a ~60-line env overlay walking struct tags, and an optional file decoded with **`go.yaml.in/yaml/v3`** using `KnownFields(true)`.

The Endpoints lint ("no entry may point at a real host in the Fake world") and the "every harness override is listed in one file" check both reflect over that one struct. Estimate: 150 to 250 lines. koanf is the right pick only if the layering grows. Reject viper.

**Criticality**: critical infra for offline safety: a missed `Endpoints` entry reaches the internet unnoticed (ADR 0014). Otherwise convenience.

**ADR conflicts**: none.

## 8. Log rotation with slog

- **Specs**: S14 (log level and rotation, live log viewer, log download).
- **Current plan**: ADR 0013 fixes `log/slog`. Rotation is unspecified. v1 used lumberjack.

| Candidate | Repo | License | Latest release | Last commit | Signal | Verified |
|---|---|---|---|---|---|---|
| lumberjack v2 | gopkg.in/natefinch/lumberjack.v2 | MIT | v2.2.1, 2023-02-06 | unverified | Frozen but stable, 9,760 importers. Size-based rotation plus `MaxBackups`, `MaxAge` and gzip. An `io.Writer`. | yes |
| timberjack | github.com/DeRuina/timberjack | MIT | v1.4.8, 2026-09-24 | unverified | Active lumberjack fork adding time-based and clock-scheduled rotation and zstd. Dep `klauspost/compress`. Single maintainer, 16 importers. | yes |
| samber/slog-multi | github.com/samber/slog-multi | MIT | v1.8.0, 2026-03-25 | unverified | Fan-out handlers. Superseded by std `slog.NewMultiHandler` (Go 1.26). | yes |

**Verdict: adopt lumberjack** as the file `io.Writer`. It is small, has no runtime deps, and rotation is not where bugs hide. Swap to timberjack only if daily rotation is wanted.

Fan out with std **`slog.NewMultiHandler`** to the file handler and a custom ring-buffer handler that feeds the live log viewer (S14) through the event hub (section 5). Building rotation yourself would be about 150 lines, including Windows' rename-while-open care, so it's not worth owning.

**Criticality**: convenience.

**ADR conflicts**: none.

## 9. IDs and clock

- **Specs**: S01-S09, S08 (Work tree ids), S15 (Node ids), S31 (Transfer ids cross Nodes).
- **Current plan**: ADR 0013/0014 fix `testing/synctest` for time in simulation tests. IDs are unspecified. v1 used google/uuid.

| Candidate | Repo | License | Latest release | Last commit | Signal | Verified |
|---|---|---|---|---|---|---|
| std `uuid` (Go 1.27) | go.dev/pkg uuid | BSD-3-Clause | Go 1.27 | n/a | `New`, `NewV4`, `NewV7`, `Parse`, `MarshalText`, `Compare`. No `sql.Scanner`/`driver.Valuer`. | yes |
| google/uuid | github.com/google/uuid | BSD-3-Clause | v1.6.0, 2024-01-23 | unverified | v7 support, Scanner/Valuer. | yes |
| gofrs/uuid v5 | github.com/gofrs/uuid | MIT | v5.5.1, 2026-08-03 | unverified | | yes (release) |
| oklog/ulid v2 | github.com/oklog/ulid | Apache-2.0 | v2.1.2, 2026-07-23 | unverified | | yes (release) |
| clockwork / benbjohnson/clock / coder/quartz | various | Apache-2.0 / MIT / MIT-0 | v0.5.0, 2024-11 / v1.3.5, 2023-05 / v0.3.1, 2026-04 | unverified | Injected fake clocks. | yes (release) |

**Verdict**:
- **Adopt the std `uuid`** (`NewV7` for ids that sort by time). Add a ~20-line `ID` type with `Scan`/`Value` for sqlc, stored as 16-byte BLOB or TEXT. This drops google/uuid.
- **Build nothing for the clock.** Call `time` directly and let `synctest` fake it, as ADR 0013/0014 already decide. Clock-injection libraries would be a second time model, and gocron and backlite-style libraries that take one add friction.

**Criticality**: convenience.

**ADR conflicts**: a clock-injection library cuts against the ADR 0014 `synctest` decision.

## 10. Instance lock, filesystem watching, disk space

- **Specs**: S01-S09 (per-data-dir instance lock), S16 (filesystem browser, storage usage per volume), S18 (scheduled storage check), S57 ("check free disk"), S22.
- **Current plan**: offline-boundaries row 39: "free space and link support read through injectable readers". Libraries are unspecified.

| Candidate | Repo | License | Latest release | Last commit | Signal | Verified |
|---|---|---|---|---|---|---|
| gofrs/flock | github.com/gofrs/flock | BSD-3-Clause | v0.13.1, 2026-08-27 | unverified | `flock` on Unix and `LockFileEx` on Windows. Deps: `x/sys` only. 1,499 importers. | yes |
| rogpeppe/go-internal `lockedfile` | github.com/rogpeppe/go-internal | BSD-3-Clause | v1.16.0, 2026-07-01 | unverified | Port of cmd/go's internal lock; the module carries many unrelated packages. | yes (release) |
| fsnotify | github.com/fsnotify/fsnotify | BSD-3-Clause | v1.10.1, 2026-05-04 | unverified | README: no NFS/SMB/FUSE support, no recursive watches. | yes |
| gopsutil v4 `disk` | github.com/shirou/gopsutil | BSD-3-Clause | v4.26.9, 2026-09-27 | unverified | Active. `disk` on Windows imports `yusufpapurcu/wmi` (and go-ole) through `internal/common`, and on macOS `ebitengine/purego`. | yes |

**Verdict**:
- **Adopt gofrs/flock** for the instance lock.
- **Don't adopt fsnotify.** Library roots are often NAS shares it can't watch, and the specs rely on scans and download-complete events.
- **Build disk usage** on `x/sys`: `unix.Statfs` and `windows.GetDiskFreeSpaceEx`, plus a volume list from `/proc/self/mountinfo`, `unix.Getfsstat` and `GetVolumePathNameW`. Borrow gopsutil's parsing. Estimate: 150 to 250 lines behind the injectable reader the Fake world already needs. gopsutil would bring WMI/COM on Windows for two calls.

**Criticality**: the lock is critical (two Controllers on one DB corrupt it). Disk space is important, since it gates updates and Moves.

**ADR conflicts**: none.

## 11. Copy, hardlink, reflink, atomic writes

- **Specs**: S33 (hardlink or copy), S31 (staging), S48 (cross-device Move on one Node), S57 (stage files, swap binary), section 3's update-state file.
- **Current plan**: ADR 0004 covers cross-Node copies (BLAKE3 chunks). The local primitives are unspecified.

| Candidate | Repo | License | Latest release | Last commit | Signal | Verified |
|---|---|---|---|---|---|---|
| otiai10/copy | github.com/otiai10/copy | MIT | v1.14.1, 2025-01-05 | unverified | Recursive copy with options. | yes (release) |
| KarpelesLab/reflink | github.com/KarpelesLab/reflink | MIT | v1.0.2, 2025-03-18 | unverified | Linux btrfs/xfs only. README lists macOS `clonefile` and Windows `DUPLICATE_EXTENTS_TO_FILE` as future work. 6 importers. | yes |
| google/renameio v2 | github.com/google/renameio | Apache-2.0 | v2.0.2, 2026-01-10 | unverified | README: "this package does not export any functions on Windows". | yes |

**Verdict: build**, borrowing from reflink:
- `os.Link` for hardlinks;
- `io.Copy` between `*os.File`s, which uses `copy_file_range` on Linux;
- reflink through `unix.IoctlFileClone` (Linux) and `unix.Clonefile` (macOS), with Windows ReFS block cloning optional later;
- an atomic-write helper (temp file in the same dir, `fsync`, `os.Rename`, `fsync` the dir on Unix).

Estimate: 200 to 300 lines. renameio is out because it has no Windows support. **Use `os.Root`** (Go 1.24+) to confine path operations under a library root, which also fits the cross-platform separator rule.

**Criticality**: critical infra (data loss on a botched cross-device move or a torn state file).

**ADR conflicts**: none.

## 12. Fault injection and network fakes for the Fake world

- **Specs**: S10 (fault proxy, synctest clock), S01-S09 (`net.Pipe` link), S31 and S48 (crash at each phase of a Move), S27 (recorded exchanges), S35 and S59 (Media Server conformance suite against real products), S51 and S52 (UDP: STUN, relay).
- **Current plan**: ADR 0014 (wire-level fakes, fault proxy keyed by proxy port, `synctest`) and ADR 0003 (in-memory link with fault injection, plus loopback processes behind a fault proxy).

| Candidate | Repo | License | Latest release | Last commit | Signal | Verified |
|---|---|---|---|---|---|---|
| Toxiproxy v2 | github.com/Shopify/toxiproxy | MIT | v2.12.0, 2025-03-18 | 2026-10 (main) | Mature. TCP-only proxy with toxics: latency, bandwidth, slow_close, timeout, reset_peer, slicer, limit_data. go.mod requires zerolog, prometheus client, gorilla/mux and urfave/cli. | yes |
| go-vcr v4 | gopkg.in/dnaeon/go-vcr.v4 | BSD-2-Clause | v4.0.7, 2026-06-25 | unverified | Client-side `http.RoundTripper` recorder and replayer. | yes (release) |
| testcontainers-go | github.com/testcontainers/testcontainers-go | MIT | v0.44.0, 2026-08-07 | unverified | Docker-driven containers from Go tests, 1,755 importers. | yes (release) |
| ory/dockertest v3 | github.com/ory/dockertest | Apache-2.0 | v3.12.0, 2025-03-12 | unverified | Older, smaller alternative. | yes (release) |
| std `httptest.NewTestServer` (Go 1.27) | Go std | BSD-3-Clause | Go 1.27 | n/a | In-memory network suited to `synctest`. | yes |

**Verdict**:

- **Fault proxy: build, borrowing Toxiproxy's toxic vocabulary.** Write each fault (latency, bandwidth cap, stall, reset, half-close, slice, cut after N bytes) once as a `net.Conn` wrapper. The same code then wraps `net.Pipe` in the simulation tier and sits in a ~150-line loopback TCP proxy in the end-to-end tier, which gives ADR 0003's two fakes one vocabulary.
  - Toxiproxy's toxics run inside its own stream pipeline, so they can't wrap a `net.Pipe`.
  - It is TCP-only, while S51 and S52 need UDP faults.
  - It brings zerolog and Prometheus into the harness.
  - Estimate: 400 to 600 lines total.
- **Simulation-tier HTTP fakes: adopt std `httptest.NewTestServer`.** It requires every product HTTP client to take an injected `http.Client` or `DialContext`, which ADR 0014 already demands.
- **go-vcr: adopt only in the S27 recording tool** that captures a ported site once. The scrubbed cassettes are then served by `fixture-site`. Never put it in the product's transport, where it would bypass the parsing, auth and TLS paths ADR 0014 wants exercised.
- **testcontainers-go: adopt for the Media Server conformance suite** (S35, S59) on Linux CI, in a test-only module. A CI check already keeps `cmd/slipstream` free of `fakes/`; extend it to cover this.

**Criticality**: important. A weak harness lets data-loss and ratio bugs through, but the harness doesn't ship.

**ADR conflicts**: go-vcr in product code would conflict with ADR 0014. Toxiproxy as the only fault layer leaves ADR 0003's in-memory link without the same faults.

## 13. Process structure (small, but in this area)

- **Specs**: S01-S09 (one binary that runs any set of Roles), S15 (Roles start and stop on a Node), S57 (drain).
- **Candidates**: `golang.org/x/sync` (BSD-3-Clause, v0.24.0, 2026-09-23: `errgroup`, `semaphore`, `singleflight`), `thejerf/suture/v4` (MIT, v4.0.6, 2024-11-29, last commit 2026-07: Erlang-style supervisor), `oklog/run` (Apache-2.0, v1.2.0, 2025-06-27). All verified.

**Verdict**:
- **Adopt `x/sync`.**
- **Build** a ~100-line Role supervisor: start, stop, restart with backoff, recover panics into Health.
- Suture is well made but adds a model for a handful of long-lived goroutines.

**Criticality**: important.

---

## Summary table

| Primitive | Specs | Verdict | Pick | Criticality |
|---|---|---|---|---|
| Job queue | S01-S09, S13, S19, S22, S33, S34, S37, S49, S57 | build (fallback: wrap backlite) | own table + sqlc, admission gate | critical |
| Scheduler | S13, S18, S22, S34, S49, S57 | build | persisted intervals; gronx only if cron is wanted | important |
| Durable operations | S31, S44, S48, S49, S50, S57 | build, borrow DBOS ideas | phase-runner; update state in a data-dir file | critical |
| Outbox and intent | S01-S09, S15, S28, S31, S49 | build | ConnectRPC stream + seq ack; level-triggered intent | critical |
| Event bus and SSE | S01-S09, S13, S14, S28, S39, S57 | build hub, adopt Huma `sse` | ring buffer for Last-Event-ID | important |
| Retry | S19, S26, S30, S35, S37 | adopt | cenkalti/backoff/v5 | important |
| Rate limiting | S19, S26, S27 | wrap | x/time/rate + persisted counters | critical (ratio/accounts) |
| Circuit breaker/health | S18, S26, S30 | build | persisted health state | important |
| Config / `Endpoints` | S01-S09, S10, S14, S57 | build | struct + flag + env + go.yaml.in/yaml/v3 | critical (offline safety) |
| Log rotation | S14 | adopt | lumberjack + `slog.NewMultiHandler` | convenience |
| IDs | S01-S09, S08, S15, S31 | adopt | std `uuid` (Go 1.27) `NewV7` | convenience |
| Clock | S10 | build nothing | `synctest` | convenience |
| Instance lock | S01-S09 | adopt | gofrs/flock | critical |
| FS watching | S22 | none | scans and events | convenience |
| Disk space / volumes | S16, S18, S57 | build, borrow gopsutil | x/sys | important |
| Copy / link / reflink / atomic write | S31, S33, S48, S57 | build, borrow reflink | x/sys + `os.Root` | critical |
| Fault proxy | S01-S09, S10, S31, S48, S51, S52 | build, borrow Toxiproxy toxics | `net.Conn` wrappers | important |
| Recording ported sites | S27 | adopt (tool only) | go-vcr v4 | convenience |
| Media Server conformance | S35, S59 | adopt (test-only) | testcontainers-go | important |
| Supervision | S01-S09, S15 | adopt x/sync, build supervisor | | important |

## Open questions for the maintainer

1. **Build the job queue on our own table and admission gate (yes), or wrap backlite to save ~600 lines?** Recommend build. The Reconciling and lockstep gate and goose ownership come for free, and backlite is a one-maintainer dependency on the most critical path.
2. **Keep the update run's phase state in a JSON file in the data dir, outside the database, so the ADR 0015 restore-and-roll-back can't erase it? (yes/no)** Recommend yes. Any DB-backed engine loses its own checkpoint at the rollback step. ADR 0015 could add one line saying so.
3. **Rule out a workflow engine (DBOS, go-workflows, Temporal) for Moves, Handover and updates for v2.0? (yes/no)** Recommend yes. DBOS on SQLite (since v1.0.0, 2026-07-30) is the only credible one. Re-check it if the phase-runner grows past ~800 lines.
4. **Pick one: integer rowids everywhere with UUIDv7 only for ids that cross Nodes or appear in URLs, or UUIDv7 for every table?** Recommend the first: SQLite rowid tables are cheaper, and only Node, Transfer, Move and outbox records need global ids.
5. **Use Go 1.27's std `uuid` and drop google/uuid? (yes/no)** Recommend yes, with a ~20-line `Scan`/`Value` wrapper for sqlc.
6. **Pick one config file format: YAML (go.yaml.in/yaml/v3, strict) or TOML (BurntSushi/toml v1.6.0, MIT), or env and flags only?** Recommend YAML. Docker and seedbox users expect it, and fakeworld writes one file per Node.
7. **Build the fault proxy as `net.Conn` wrappers shared by the simulation and end-to-end tiers, instead of embedding Toxiproxy? (yes/no)** Recommend yes. One fault vocabulary, UDP coverage for S51/S52, and no zerolog/Prometheus in the harness.
8. **Run the Media Server conformance suite through testcontainers-go on Linux only? (yes/no)** Recommend yes. Docker isn't available on hosted macOS runners (inferred), and the suite tests the servers, not the OS.
9. **Keep scheduled tasks on fixed intervals with no cron syntax, as v1 did (P-229)? (yes/no)** Recommend yes. It removes the only reason to take gocron or robfig/cron.
10. **Lumberjack (frozen since 2023, size-based) or timberjack (active fork, adds daily rotation and a compression dep)?** Recommend lumberjack, unless the log viewer or download wants per-day files.
