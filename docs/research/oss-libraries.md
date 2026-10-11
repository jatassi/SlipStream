# Open source libraries for SlipStream v2

Researched 2026-10-11 for the question: which of v2's primitives should come from open source libraries instead of being custom-built, and what does each choice trade away? Organized around the feature graph's seven foundation specs (S01 to S07), then the feature specs that follow.

Detail per primitive, with candidate tables, versions, licenses and evidence, lives in the appendices under [oss-libraries/](oss-libraries/):

- [A-core.md](oss-libraries/A-core.md) covers the job queue, scheduler, durable operations, outbox, event hub, retry, config, logging, IDs, files and the Fake world's fault tooling.
- [B-net-security.md](oss-libraries/B-net-security.md) covers the Node link, Pairing, Accounts, SMTP, NAT and relay, the updater, packaging and the ADR-chosen formats.
- [C-media.md](oss-libraries/C-media.md) covers release parsing, .torrent files, ffprobe, subtitles, Custom Format rules, naming and search.
- [D-integrations.md](oss-libraries/D-integrations.md) covers metadata, indexers and Cardigann, Download Clients, Media Servers, notifications, Radarr and Sonarr, TRaSH and Subtitle Providers.
- [E-frontend.md](oss-libraries/E-frontend.md) covers UI primitives, the phone shell, forms, live updates, passkeys and test tooling.
- [F-prior-decisions-issue-1.md](oss-libraries/F-prior-decisions-issue-1.md) lists every library verdict already reached in the 53 sub-issues of the map (#1).

## The short answer

v2 is already less custom than it feels. ADR 0013 adopted libraries for every large piece: Huma, sqlc, goose, modernc SQLite, yamux, connect-go, regexp2, CertMagic, libtorrent and the TanStack family. The open question is the layer between those libraries and the domain code, and there the recommendation splits cleanly in two.

**Adopt a library wherever the work is a standard, a format or accessibility.** Writing these yourself buys nothing and risks security or correctness bugs that a library has already paid for. The list:

- WebAuthn on the server: go-webauthn.
- Crypto and formats: filippo.io/age, klauspost zstd, aead.dev/minisign, zeebo/blake3, argon2id from x/crypto.
- SMTP: wneessen/go-mail.
- STUN: pion/stun.
- HTML scraping: goquery and cascadia.
- YAML: goccy/go-yaml for Cardigann definitions and go.yaml.in/yaml/v3 for config.
- Accessible UI primitives: Base UI.
- Leaf utilities: cenkalti/backoff, x/time/rate, gofrs/flock, lumberjack, x/sync.
- SQLite FTS5 for search, which modernc already compiles in.
- Test tooling: Playwright, Vitest browser mode, axe-core and testcontainers-go.

**Build the core and the clients yourself**, because the ADRs themselves rule the libraries out. Your instinct toward a custom build is right here, and for concrete reasons rather than taste:

- **Durable work.** The job queue, scheduler and phase runner can't come from River, backlite or DBOS. ADR 0015's rollback restores a Backup over the database, which erases any engine's checkpoint stored in it. ADR 0019's Reconciling and ADR 0003's lockstep both need one gate every job start passes through. Workflow engines also bring their own migrators beside goose. The custom version is about 1.2k to 1.8k lines (estimated in A).
- **External clients.** TMDB, TVDB, Plex, Jellyfin, notifiers and Download Client libraries mostly hardcode hosts, hold a package-global HTTP client, or retry on their own. Each of those breaks ADR 0014's `Endpoints` and wire-fake rule. plexgo, for one, lets plex.tv calls reach the internet unless every call overrides the host (read in its source, D).
- **Media semantics.** ADR 0006 runs TRaSH formats verbatim, and those formats test Sonarr's and Radarr's parsed values. A Go release parser with different semantics breaks that, so the parser and naming grammar are ports of Sonarr and Radarr (GPL-3, compatible), not dependencies. Subtitle libraries rewrite cue bytes, which ADR 0008 forbids.

A third category is **borrow**: read or port the code, take no dependency. That covers Sonarr and Radarr's parser and its roughly 1,900 GPL-3 test titles, Prowlarr's Cardigann template evaluator, tailscale's UPnP, NAT-PMP and PCP code (BSD-3), Toxiproxy's fault vocabulary, and minio/selfupdate's Windows rename trick.

No recommendation here reopens an ADR. The findings that needed an ADR line are recorded in ADRs 0012, 0013 and 0015.

## Foundation specs

| Spec | Adopt | Build (and borrow) | Notes |
|---|---|---|---|
| S01 Release build and native CI | nfpm (deb, rpm), WiX, appimagetool | A `cmd/release` tool that drives them | GoReleaser's cgo split builds, MSI, DMG and `.app` support are Pro only (closed source, about $165 a year). Add BSD-4-Clause, AGPL, GPL-2.0-only and "no license" to the license deny list. |
| S02 Node runtime and configuration | go.yaml.in/yaml/v3 with `KnownFields(true)`, gofrs/flock, lumberjack, `slog.NewMultiHandler` (std since Go 1.26), x/sync | Typed `Config` with `Endpoints` and `Timings`, std `flag`, a 60-line env overlay, a 100-line Role supervisor | Reject viper and koanf: the `Endpoints` lint reflects over one struct. `gopkg.in/yaml.v3` now marks itself unmaintained. Use no clock library, since `testing/synctest` fakes time (ADR 0014). Seeded randomness needs nothing beyond `math/rand/v2` (inferred, not researched). |
| S03 Storage and migrations | std `uuid` (new in Go 1.27, verified on go.dev/doc/go1.27), FTS5 in modernc | Job table with leases and the admission gate, phase runner, atomic-write helper on `os.Root` | Integer rowids for local tables, UUIDv7 only for ids that cross Nodes. The update run's state lives in a data-dir file, outside the database it may roll back. |
| S04 API, sign-in and live updates | Huma's `sse` package, `http.CrossOriginProtection` (std), argon2id, x/time/rate | Event hub with a replay ring buffer, session table, per-IP limiter, hashed API keys | go-sse and r3labs/sse would emit events outside Huma, dropping them from the OpenAPI document and therefore from Orval. |
| S05 Fake world and the end-to-end tier | `httptest.NewTestServer` (Go 1.27, in-memory network for synctest), testcontainers-go (conformance only), go-vcr (the site-recording tool only) | Fault wrappers as `net.Conn`, shared by `net.Pipe` and a loopback proxy | Toxiproxy is TCP-only and can't wrap `net.Pipe`, and the relay and STUN need UDP faults. The reference servers #26 already picked (Pebble, go-smtp, pion/stun, miniupnpd) stay. |
| S06 Web app foundations | Base UI, TanStack Form, TanStack Store (already shipped inside Router), Hugeicons (`@hugeicons/react` with the MIT free set, the maintainer's pick), Fontsource Geist, Playwright, Vitest browser mode, `@axe-core/playwright` | A 150-line `LiveStream` over native `EventSource`, a 60-line passkey module on native `PublicKeyCredential` JSON methods | Drop v1's vaul (its README says unmaintained), cmdk (no release in 19 months), Sonner, react-hook-form, zustand and `@simplewebauthn/browser`. Keep TanStack Virtual hooks in small leaf components: React Compiler skips memoizing any component that calls them (read in the compiler source). |
| S07 Node link and Role dispatch | hashicorp/yamux (behind an interface), connect-go 2.x | Self-signed Ed25519 certificate with the pin check in `VerifyConnection`, an h2c adapter over yamux streams (about 100 lines), Join Codes, outbox and intent sync | The pin check must sit in `VerifyConnection`, because `VerifyPeerCertificate` doesn't run on resumed TLS 1.3 sessions. See the yamux window risk below. |

S08, the Content Type contract, needs no library.

## Feature specs

| Area | Specs | Adopt | Wrap | Build or borrow |
|---|---|---|---|---|
| Accounts | S12 | go-webauthn v0.18 (per-ceremony relying party, fits ADR 0005), Base UI OTP Field | wneessen/go-mail | Sessions, PIN hashing with a key-file pepper, credential encryption with XChaCha20-Poly1305 |
| Scheduling and health | S13, S18 | | | Scheduler with persisted `last_run_at` (about 200 lines), persisted health and breaker state |
| Library files | S16, S22, S31, S33, S48 | | | Hardlink, `copy_file_range`, reflink and disk usage on x/sys, borrowing from gopsutil and KarpelesLab/reflink. gopsutil brings WMI on Windows for two calls. Path sanitizing (about 150 lines) |
| Parsing and rules | S23, S24, S53 | regexp2 (decided) | | Port Sonarr and Radarr's parser (4k to 6k lines) and copy their GPL-3 fixtures. Typed Custom Format model, not expr or CEL. FileNameBuilder token grammar, not `text/template` |
| .torrent files | S26, S32, S34 | | | A 300-line bencode decoder that keeps raw `info` bytes. Importing anacrolix's metainfo alone pulls in about 20 packages (measured) |
| Probe and subtitles | S22, S44, S46 | x/text/language and encodings | saintfish/chardet with a language hint | ffprobe wrapper (200 lines); subtitle parse and write that keeps cue bytes. go-astisub drops ASS `[Fonts]` and rebuilds SRT text (read in its source), so it serves only as a test oracle |
| Indexers | S26, S27 | goquery, cascadia, goccy/go-yaml, gofeed (generic RSS only) | | Port Prowlarr's Cardigann runtime and template evaluator. Go `text/template` rejects 40 live template strings (measured on the 579-definition v11 catalog). Never port from Jackett, which is GPL-2.0-only |
| Download Clients | S30 | | | Four thin clients, borrowing quirks from autobrr (MIT). hekmon/transmissionrpc predates Transmission 4.1's JSON-RPC 2.0 |
| Media Servers | S35, S43 | | | About 25 hand-written calls. plexgo (243k generated lines) and jellyfin-go (183k) are too large and leak hosts |
| Notifications | S37 | | | Hand-written payloads, about 1,200 lines. The maintained shoutrrr fork hardcodes five providers' hosts |
| Radarr and Sonarr | S55 | | golift/starr (MIT, injectable base URL and client) | |
| Reachability and relay | S51, S52 | pion/stun | | Port tailscale's portmapper (goupnp uses a global HTTP client). SNI routing by peeking the ClientHello, and a UDP flow table over link streams |
| Backups, Transfers, updates | S31, S49, S57 | age, zstd, minisign, zeebo/blake3 | | Updater swaps a file set and keeps the previous version, which no library does |
| Packages and tray | S56 | | fyne.io/systray | getlantern/systray is abandoned |
| UI | S11, S20, S17, S38, S39 | Base UI Drawer and Toast, date-fns 4, react-markdown (lazy) | visx only if S36 needs axes | Push navigation on React 19.3's `<ViewTransition>`, reorder on Motion with Move up and down actions, hand SVG meters, tables over TanStack Virtual |

## What the custom side costs

The build verdicts add up to real work, so here is the size of it, estimated by the researchers from v1's code and the libraries they read:

- The durable core (job queue, scheduler, phase runner, outbox, event hub) is about 1.2k to 1.8k lines.
- Integration clients run 300 to 1,500 lines each in v1. Notifications total about 1,200.
- The Sonarr and Radarr parser port is the largest single piece, at 4k to 6k lines, but its fixtures come free.
- Files, config, fault wrappers and the relay pieces are a few hundred lines each.

Every dependency avoided this way would otherwise cost the same thing again: a wrapper to make it injectable for the Fake world, plus a fake behind it. That is why the "adopt" list is short and sits at the leaves.

## Risks the research turned up

1. **yamux caps one stream at about 21 Mbit/s at 100 ms RTT** with its fixed 256 KiB window (computed in B). Relayed Media Server streams (S52) and Transfers (S31) cross that link. hashicorp/yamux has been in caretaker mode since v0.1.2 (2024-09). libp2p/go-yamux speaks the same protocol and tunes its window by RTT. S07's "one measured yamux stream under added latency" is the right place to settle it.
2. **The update run can't checkpoint in the database.** ADR 0015 restores a Backup when the health trial fails twice, so a database checkpoint would be lost at exactly that step. ADR 0015 should say the run's state lives in a data-dir file.
3. **The relay hides client addresses.** Unless the relay passes each client's original address to the Serving Node, ADR 0005's per-IP rate limiting sees every remote client as one address.
4. **Replacing the binary inside a signed macOS `.app` breaks the bundle's seal.** The updater should swap the whole bundle, with a native smoke test in S56.
5. **Documentation drift.** Prowlarr now publishes a v12 Cardigann catalog, while boundary row 7 pins `master/11`. Boundary row 16 doesn't say which Transmission protocol its fake speaks, and Transmission 4.1.0 deprecated the old one.
6. **License traps.** bogdanfinn/tls-client is BSD-4-Clause, which is GPL-incompatible. configarr is AGPL. Jackett is GPL-2.0-only. Prowlarr's Indexers repo has no license file. None of these is a pick, and the deny list should catch them.

## Decisions

The maintainer accepted every recommendation below and in the appendices on 2026-10-11, and picked Hugeicons over Lucide. [ADR 0021](../adr/0021-slipstream-builds-its-durable-core-and-clients-and-adopts-libraries-for-standards.md) records the rule.

1. Build the job queue, scheduler and phase runner on our own tables, rather than River, backlite or DBOS? **Recommend yes.**
2. Keep the update run's phase state in a data-dir file, outside the database, and add that line to ADR 0015? **Recommend yes.**
3. Hand-write the metadata, Media Server and notification clients rather than use SDKs (plexgo, jellyfin-go) or shoutrrr? **Recommend yes.** Wrap golift/starr for Radarr and Sonarr as the one exception, since it already takes an injected base URL and client.
4. Port Sonarr and Radarr's parser and naming grammar, and copy their GPL-3 test titles as golden files with attribution? **Recommend yes.**
5. Port Prowlarr's Cardigann template evaluator, parse definitions with goccy/go-yaml, and support catalog v11 and v12 from the start? **Recommend yes.**
6. Base UI for accessible primitives, over React Aria or Radix? **Recommend Base UI.** It is the only one with Drawer, Toast, OTP Field and Combobox, and v1 used it. This one also belongs to [Design system and DESIGN.md](https://github.com/jatassi/SlipStream/issues/94).
7. TanStack Form for forms, over react-hook-form? **Recommend TanStack Form.** It takes Orval's Zod schemas directly and has no React Compiler incompatibility.
8. Passkeys on the browser's native JSON methods, with no passkey offered before Safari 18.4, dropping `@simplewebauthn/browser`? **Recommend yes.** The PIN covers older devices under ADR 0005.
9. hashicorp/yamux behind an interface now, switching to libp2p/go-yamux if S07's latency measurement shows the cap? **Recommend yes.**
10. Join Codes as a pasted high-entropy code carrying the Controller's key and a 128-bit secret, with no PAKE? **Recommend yes.** Typeable short codes would need a PAKE, and the best Go option (filippo.io/cpace) calls itself experimental.
11. Must the relay pass each client's original address to the Serving Node? **Recommend yes.**
12. A custom `cmd/release` with nfpm, WiX and appimagetool, rather than GoReleaser Pro? **Recommend custom.**
13. Speak both Transmission protocols, picked by the version header? **Recommend yes.** 4.0.x is still widely deployed.
14. Hash the PIN as HMAC with a key-file pepper, then argon2id? **Recommend yes.** A 4-digit PIN is otherwise trivial to brute-force from a leaked copy of the database.

## How this was checked

Five researchers read primary sources on 2026-10-11: proxy.golang.org, pkg.go.dev, deps.dev, npm, MDN browser-compat-data, raw LICENSE files, and module source code downloaded to a scratch folder. The GitHub API returned 403, so star and issue counts are mostly missing and maintenance is judged from release and commit dates. Some claims were measured in scratch programs:

- the Cardigann catalog against yaml.v3, cascadia, regexp2 and `text/template`;
- FTS5 trigram and diacritic matching in modernc v1.60.1;
- lingua-go's binary cost (about 132 MB, which rules it out);
- the dependency count of anacrolix's metainfo;
- the Sonarr and Radarr fixture counts.

Each appendix marks every other claim as verified, inferred or unverified. Nothing was changed in either repository.
