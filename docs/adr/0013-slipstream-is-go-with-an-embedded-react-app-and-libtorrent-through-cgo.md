# SlipStream is Go with an embedded React app and libtorrent through cgo

SlipStream v2 is one Go binary for the Controller and every Worker. It links libtorrent-rasterbar through a small C shim with cgo, keeps its durable state in one SQLite database on the Controller, and serves a React single-page app embedded in the binary. Go was the only candidate with a maintained in-process library for every built-in. It runs TRaSH's .NET-syntax regexes through `regexp2` with a per-match timeout (ADR 0006), and serves ACME TLS-ALPN-01 certificates through CertMagic (ADR 0012). v1 is Go, so the maintainer already reads it. Decided in [Choose the stack](https://github.com/jatassi/SlipStream/issues/11). Research in [Stack candidates against the built-ins' needs](https://github.com/jatassi/SlipStream/issues/8) and [Torrent engine bakeoff](https://github.com/jatassi/SlipStream/issues/33).

The stack:

- **Server.** Go 1.27, `log/slog`, plain constructors with no DI framework. The JSON REST API is code-first with Huma v2 on `net/http`, which generates the OpenAPI 3.1 document. CI fails when the committed document drifts from the Go types.
- **Live updates.** One Server-Sent Events stream per session. Its typed events are declared in the OpenAPI document.
- **Database.** SQLite in WAL mode on the Controller, through modernc's pure-Go driver. Queries are SQL, with sqlc generating typed Go. goose migrations are kept per module, so the core and each Content Type own their tables (ADR 0001). A Worker keeps its outbox and torrent resume data in a local SQLite file it can lose (ADR 0003). There is no Postgres option.
- **Node link.** One TLS 1.3 connection, authenticated by each Node's pinned Ed25519 key and multiplexed by yamux. Each side serves ConnectRPC protobuf services over it with connect-go 2.x, so calls go both ways whichever side dialed. The handshake and update services sit in their own proto package, and `buf breaking` in CI blocks any breaking change to it, which keeps ADR 0003's update path frozen. On a single machine, Role code calls the same Go service interfaces directly.
- **UI.** React 19.3 and TypeScript 7, built by Vite 8 with React Compiler on, using TanStack Router, Query and Virtual, Tailwind 4 and Motion. Oxlint, Oxfmt and Vitest are the tooling. Orval generates TanStack Query hooks, Zod schemas and MSW mocks from the OpenAPI document.
- **Indexers.** The Cardigann runner reads the YAML catalog at runtime (ADR 0010). Each of Prowlarr's 53 active site-specific C# indexers is ported to a native Go indexer behind one indexer contract, and its 11 obsolete ones are skipped.
- **Native code.** libtorrent-rasterbar, built static, through the C shim. The built-in client announces as libtorrent, with the `-LT` peer_id prefix and libtorrent's version. libfvad is vendored and built through cgo for Timing sync, beside gonum's FFT. ffmpeg and `dovi_tool` run as child processes, suspended and resumed through `golang.org/x/sys` (SIGSTOP and SIGCONT on Linux and macOS, `NtSuspendProcess` on Windows).

## Considered Options

- **Rust.** Rejected. It has turmoil, the strongest multi-Node simulator found, but no regex engine that runs .NET syntax, so TRaSH's regexes could not run verbatim as ADR 0006 requires.
- **.NET.** Rejected, though it came second. Native regex and copy-portable Prowlarr indexers count in its favour. Against it, the ACME TLS-ALPN-01 challenge would be wired into Kestrel by hand, the runtime idles heavier on small Workers, and the maintainer would review C# instead of Go. By the time of the decision, its research gaps on WireGuard and QUIC no longer applied (ADR 0004, ADR 0012).
- **A native Go torrent engine.** Rejected in the bakeoff. anacrolix and rain used 13 to 26 times libtorrent's memory per thousand idle seeds, and both drew tracker rejections on multi-URL private torrents.
- **A spec-first API**, with the OpenAPI YAML written by hand and Go generated from it. Rejected. It is a second copy of the types to keep in step.
- **A WebSocket for live updates**, as v1 used. Rejected. The UI never pushes over it, and SSE reconnects on its own and passes through Serving Workers and reverse proxies as plain HTTP.
- **A server-rendered React framework** (Next.js, React Router framework mode, TanStack Start with server rendering, Server Components). Rejected. Each needs a JavaScript server at runtime, and the Go binary serves static files.

## Consequences

- cgo is on, so Go loses its pure cross-compile. Each OS and architecture builds in its own native CI job with libtorrent, boost and OpenSSL. Packaging, the updater and the update dashboard owns that matrix.
- Porting the C# indexers is about 99 files and 1.15 MB of C# to rewrite in Go, measured on 2026-10-11. The port takes the 53 active indexers that [Prowlarr catalog: vendor or reimplement](https://github.com/jatassi/SlipStream/issues/9) counted. Decided in the [coherence pass](https://github.com/jatassi/SlipStream/pull/112), Q22.
- The built-in client passes any allow-list tracker that carries Gazelle's generic `-LT` row. A new SlipStream prefix would fail every allow-list tracker until its staff added it. Decided in the [coherence pass](https://github.com/jatassi/SlipStream/pull/112), Q1.
- Go has no multi-Node network simulator. Simulation tests run yamux over `net.Pipe` behind a fault wrapper, so they exercise the real Node link code, and `testing/synctest` fakes time.
- yamux is MPL-2.0, which is compatible with GPL-3.0 (ADR 0010). It goes in the notices file.
- TanStack DB and Vite+ were left out until they mature. The walking skeleton checks that React's `<ViewTransition>` works with TanStack Router's navigation.
- hashicorp/yamux's fixed 256 KiB stream window caps one stream at about 21 Mbit/s at 100 ms round trip (computed, not measured), which relayed playback and Transfers would hit. The Node link keeps yamux behind an interface, and the Node link spec measures one stream under added latency. libp2p/go-yamux speaks the same protocol and tunes its window to the round trip, so it replaces hashicorp's if the measurement shows the cap. Decided in the [library research](../research/oss-libraries.md).
- Which libraries sit beside this stack, and what SlipStream builds itself, is [ADR 0021](0021-slipstream-builds-its-durable-core-and-clients-and-adopts-libraries-for-standards.md).
