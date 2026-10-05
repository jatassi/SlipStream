# Stack candidates against the built-ins' needs

Research for [#8](https://github.com/jatassi/SlipStream/issues/8) on the [SlipStream v2 map](https://github.com/jatassi/SlipStream/issues/1). It lays out options and tradeoffs. It does not pick a stack.

Charting decisions this leans on come from [Chart SlipStream v2](https://github.com/jatassi/SlipStream/issues/2): the torrent client is private-tracker grade (Q13), one binary runs as Controller or worker Node (Q9), the forwarder carries Media Server traffic with TLS and a domain (Q10), file sync pushes completed downloads between Nodes (Q11), the A/V pipeline derives Renditions and schedules hardware acceleration across Nodes (Q14), v2 ships native packages plus Docker images (Q24), and CI runs the end-to-end suite with network egress blocked (Q17).

## Method

- Maturity figures (stars, license, last push, latest release) come from the GitHub REST API, crates.io API and NuGet search API, all read on 2026-10-04. The full snapshot is in [Appendix A](#appendix-a-maturity-snapshot).
- Capability claims cite the project's own README, source file or official docs.
- A claim marked **inferred** is a reading of the evidence, not something a source states.
- A feature called "not listed" means the project's own feature list omits it. It may still exist undocumented.

## Answer in brief

- **Go** is the only candidate with a maintained, in-process library for every need (**inferred** from the tables below). The weak point is the torrent engine. Go's two engines (anacrolix/torrent, rain) honour BEP 27 but neither is on a sampled private-tracker client whitelist.
- **Rust** matches Go on tunnel, TLS, QUIC, database and simulation testing. turmoil is the strongest multi-Node simulation tool found (**inferred**). Its torrent engine (librqbit) lists BEP 27 but not message stream encryption or BitTorrent v2, and is also not whitelisted.
- **.NET** has the most precedent in this exact domain (Sonarr, Radarr, Prowlarr, Jellyfin) and the best ffmpeg wrapper. No maintained userspace WireGuard was found, the Kestrel ACME integration (LettuceEncrypt) is archived and Certes is a bare ACME client, QUIC needs a system `libmsquic` on Linux, and Native AOT cannot cross-compile across operating systems.
- **The torrent engine is the deciding constraint, and it cuts across all three.** The only engine on the sampled whitelist from this candidate set is libtorrent-rasterbar (C++). No language has a maintained binding to it except Python (official) and Java (libtorrent4j). Embedding it means owning a C ABI shim in whichever language is picked.

## The needs, one by one

### 1. Private-tracker-grade torrent engine

What "private-tracker grade" has to mean, from primary sources:

- **BEP 27 compliance.** With the private flag set, a client "MUST ONLY announce itself to the private tracker" and must only connect to peers that tracker returns. It uses one tracker at a time and drops all peers when it switches ([BEP 27](https://www.bittorrent.org/beps/bep_0027.html)). **Inferred:** this rules out DHT, PEX and Local Peer Discovery for those torrents, which BEP 27's rationale discusses without a separate MUST.
- **An allowed client identity.** Private trackers publish client whitelists keyed on the client and version. The [Orpheus whitelist](https://interview.orpheus.network/bittorrent-client-whitelist.php) lists aTorrent, BitTorrent, Deluge, Flud, Halite, KTorrent, libtorrent (Rasterbar), libtorrent (rtorrent), qBittorrent, Transmission and Mainline. This is one tracker. Others publish their own lists. **Inferred:** a new engine needs whitelisting tracker by tracker, and presenting another client's BEP 20 peer ID prefix to pass a whitelist would misrepresent the client.
- **Fast resume, scrape, encryption, and many seeding torrents** at once (inferred from what whitelisted clients ship, for example the [libtorrent feature list](https://www.libtorrent.org/features-ref.html)).

| Engine | Language | License | BEP 27 | MSE encryption | BitTorrent v2 (BEP 52) | Peer ID and UA configurable | On Orpheus whitelist | Latest release |
|---|---|---|---|---|---|---|---|---|
| [libtorrent-rasterbar](https://github.com/arvidn/libtorrent) | C++ | BSD-3-Clause | Yes ([features](https://www.libtorrent.org/features-ref.html)) | Yes | Yes, since 2.0 | Yes | **Yes** | v2.1.2, 2026-09-25 |
| [anacrolix/torrent](https://github.com/anacrolix/torrent) | Go | MPL-2.0 | Yes. DHT, PEX and LPD are skipped for private torrents ([torrent.go](https://github.com/anacrolix/torrent/blob/HEAD/torrent.go), [pexconn.go](https://github.com/anacrolix/torrent/blob/HEAD/pexconn.go), [client.go](https://github.com/anacrolix/torrent/blob/HEAD/client.go)) | Yes ([README](https://github.com/anacrolix/torrent)) | Yes ([README](https://github.com/anacrolix/torrent)) | Yes, `Bep20`, `HTTPUserAgent`, `ExtendedHandshakeClientVersion` ([config.go](https://github.com/anacrolix/torrent/blob/HEAD/config.go)) | No | No GitHub releases. Tags only. Last push 2026-09-27 |
| [cenkalti/rain](https://github.com/cenkalti/rain) | Go | MIT | Yes, with private-only peer ID, handshake version and tracker UA settings ([config.go](https://github.com/cenkalti/rain/blob/HEAD/torrent/config.go)) | Yes ([README](https://github.com/cenkalti/rain)) | Not listed | Yes, for private torrents | No | v2.4.2, 2026-10-03 |
| [librqbit](https://github.com/ikatson/rqbit) | Rust | Apache-2.0 | Yes ([README](https://github.com/ikatson/rqbit)) | Not listed | Not listed | BEP 20 listed | No | v9.0.1, 2026-08-20 |
| [MonoTorrent](https://github.com/alanmcgovern/monotorrent) | C# | MIT | Yes ([README](https://github.com/alanmcgovern/monotorrent)) | Yes | Yes (draft BEPs list) | BEP 20 listed | No | v3.0.2, 2024-08-04. Last push 2026-07-31 |

Production signals, from each project's README:

- anacrolix/torrent has been "used 24/7 in production by downstream services since late 2014" and lists autobrr, bitmagnet, Gopeed and TorrServer as downstream users ([README](https://github.com/anacrolix/torrent)).
- rain runs "in production at put.io since 2019". It allocates a separate peer port per torrent to support "multiple downloads of the same torrent for different accounts on private trackers" ([README](https://github.com/cenkalti/rain)). It does not list uTP.
- librqbit lists "Sequential downloading (the default and only option)" ([README](https://github.com/ikatson/rqbit)). **Inferred:** sequential-only piece picking is a poor fit for seeding swarms, where rarest-first matters.
- MonoTorrent has 924,589 NuGet downloads. Its last stable release is over two years old.

Bindings to libtorrent-rasterbar outside C++ and Python:

| Binding | Target | State |
|---|---|---|
| [libtorrent4j](https://github.com/aldenml/libtorrent4j) | JVM | Latest release v2.1.0-39, 2026-02-07, 284 stars |
| [lt-rs](https://github.com/pliduino/lt-rs) | Rust | 0.1.4, 519 crates.io downloads, created 2025-12-02 |
| [libtorrent-sys](https://crates.io/crates/libtorrent-sys) | Rust | Last updated 2022-06-03 |
| [i96751414/libtorrent-go](https://github.com/i96751414/libtorrent-go) | Go | Last push 2024-09-03, 7 stars |
| [vktr/libtorrent-net](https://github.com/vktr/libtorrent-net) | .NET | Last push 2017-03-10 |

**Inferred:** none of these is a maintained base for production use. Embedding libtorrent from Go, Rust or .NET means writing and owning a C ABI shim (cgo, Rust `cxx` or `bindgen`, or .NET P/Invoke), plus a C++ toolchain per target in the build.

### 2. Userspace WireGuard or an equivalent tunnel

| Option | Language | License | Notes |
|---|---|---|---|
| [wireguard-go](https://github.com/WireGuard/wireguard-go) with `tun/netstack` | Go | MIT | The official Go implementation. `tun/netstack` runs the tunnel over gVisor's userspace TCP/IP stack, with HTTP client and server examples ([tun/netstack](https://github.com/WireGuard/wireguard-go/tree/master/tun/netstack)). Latest tag 0.0.20250522 |
| [tsnet](https://pkg.go.dev/tailscale.com/tsnet) | Go | BSD-3-Clause | "Embeds a Tailscale node directly into a Go program" with no root or TUN. Needs a coordination server, either Tailscale's or a custom `ControlURL` such as [Headscale](https://github.com/juanfont/headscale) (BSD-3-Clause, v0.29.4) |
| [libtailscale](https://github.com/tailscale/libtailscale) | C ABI over Go | BSD-3-Clause | Builds tsnet as a C archive with `go build -buildmode=c-archive` ([README](https://github.com/tailscale/libtailscale)). No releases |
| [BoringTun](https://github.com/cloudflare/boringtun) | Rust | BSD-3-Clause | Deployed "on millions of iOS and Android consumer devices". The README warns against linking master and points to crates.io. Crate 0.7.1, 2026-05-01, 1.36M downloads. Ships the protocol "without the network or tunnel stacks" ([README](https://github.com/cloudflare/boringtun)) |
| [GotaTun](https://github.com/mullvad/gotatun) | Rust | MPL-2.0 (BSD-3 before 2026-03-05) | Mullvad's fork of BoringTun. v0.9.2, 2026-08-31, 217K crates.io downloads |
| [smoltcp](https://github.com/smoltcp-rs/smoltcp) | Rust | 0BSD | Userspace TCP/IP stack to pair with BoringTun or GotaTun. v0.14.0, 2026-08-17 |
| .NET | C# | n/a | No maintained managed implementation found. A NuGet search for "wireguard" returns WireGuard.Net with 8,813 downloads as the top hit. **Inferred:** .NET would call a Go or Rust implementation over FFI, or run one as a sidecar |

Equivalent tunnels that avoid WireGuard: QUIC between Nodes (see need 4), or TLS over TCP with a reverse connection from the Node behind NAT. **Inferred:** tsnet is the only option found that bundles NAT traversal and relays (DERP) as a library. Using it ties the relaying Role to a Tailscale-compatible coordination server.

### 3. TLS with ACME

| Option | Language | License | Notes |
|---|---|---|---|
| [CertMagic](https://github.com/caddyserver/certmagic) | Go | Apache-2.0 | The engine behind Caddy. Pluggable storage, DNS challenge, on-demand issuance, distributed challenge solving ([README](https://github.com/caddyserver/certmagic)). v0.25.6, 2026-09-30 |
| [lego](https://github.com/go-acme/lego) | Go | MIT | ACME client with "more than 200 DNS providers" ([README](https://github.com/go-acme/lego)). v5.5.2, 2026-09-23 |
| [rustls](https://github.com/rustls/rustls) | Rust | Apache-2.0, MIT or ISC | TLS stack. v0.23.45, 2026-09-14 |
| [rustls-acme](https://github.com/FlorianUekermann/rustls-acme) | Rust | Apache-2.0 or MIT | ACME plus serving over rustls. 0.15.4, 750K downloads |
| [instant-acme](https://github.com/djc/instant-acme) | Rust | Apache-2.0 | Async ACME client. 0.8.5, 3.0M downloads |
| [Certes](https://github.com/fszlin/certes) | C# | MIT | ACME client only. v4.1.0, 2026-09-27, 3.4M NuGet downloads |
| [LettuceEncrypt](https://github.com/natemcmaster/LettuceEncrypt) | C# | Apache-2.0 | Kestrel ACME integration. **Archived** on GitHub. Last release v1.3.3, 2024-05-20 |

For offline mode, [Pebble](https://github.com/letsencrypt/pebble) is Let's Encrypt's ACME test server (Go, MPL-2.0, v2.10.1). **Inferred:** a Go stack can run Pebble in-process in tests. Rust and .NET would run it as a separate process.

### 4. Fast multi-stream file transfer between Nodes

| Option | Language | License | Notes |
|---|---|---|---|
| [quic-go](https://github.com/quic-go/quic-go) | Go | MIT | Pure Go QUIC and HTTP/3. v0.63.0, 2026-09-22. Still pre-1.0 |
| [quinn](https://github.com/quinn-rs/quinn) | Rust | Apache-2.0 | Pure Rust QUIC. quinn-proto 0.11.19, 2026-09-30 |
| [System.Net.Quic](https://learn.microsoft.com/en-us/dotnet/fundamentals/networking/quic/quic-overview) | C# | MIT | Stable since .NET 9. Depends on native MsQuic. Bundled on Windows 11 and Server 2022 or later. On Linux "you must manually install `libmsquic`". macOS is "partially supported" through Homebrew ([docs](https://learn.microsoft.com/en-us/dotnet/fundamentals/networking/quic/quic-overview)) |

All three also have HTTP/2 and plain TCP in their standard libraries. Go has two large file-movement precedents, [Syncthing](https://github.com/syncthing/syncthing) (MPL-2.0) and [rclone](https://github.com/rclone/rclone) (MIT). **Inferred:** the .NET `libmsquic` dependency conflicts with single-binary packaging on Linux and macOS unless SlipStream bundles MsQuic itself.

### 5. ffmpeg orchestration

Every candidate drives ffmpeg and ffprobe as child processes. The hardware acceleration logic (VAAPI, QSV, NVENC, AMF, VideoToolbox, RKMPP) is domain code in any language. Jellyfin, a .NET Media Server, holds this logic in one file, [EncodingHelper.cs](https://github.com/jellyfin/jellyfin/blob/HEAD/MediaBrowser.Controller/MediaEncoding/EncodingHelper.cs), which names all six backends. Jellyfin is GPL-2.0, so it is a precedent, not a source to copy.

| Option | Language | License | Notes |
|---|---|---|---|
| `os/exec` | Go | BSD-3-Clause | Standard library |
| [ffmpeg-go](https://github.com/u2takey/ffmpeg-go) | Go | Apache-2.0 | Last release v0.5.0, 2023-08-08. Last push 2024-05-24 |
| [ffmpeg-sidecar](https://github.com/nathanbabcock/ffmpeg-sidecar) | Rust | MIT | Wraps the CLI with a parsed event stream. v2.6.0, 2026-09-30, 1.95M downloads |
| [rust-ffmpeg](https://github.com/zmwangx/rust-ffmpeg) | Rust | WTFPL | Links libav* directly. v9.0.0, 2026-08-05 |
| [FFMpegCore](https://github.com/rosenbjerg/FFMpegCore) | C# | MIT | Wraps the CLI. v5.5.0, 2026-09-20, 7.7M NuGet downloads |

**Inferred:** this need does not separate the candidates. The fake for offline mode is a stub ffmpeg binary on the path, which works the same in every language.

### 6. Embedded database

| Option | Language | License | Notes |
|---|---|---|---|
| [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite) | Go | BSD-3-Clause | "A CGo-free port of the C SQLite3 library." Ships SQLite 3.53.4 on Linux, macOS, Windows, FreeBSD, NetBSD and OpenBSD across amd64 and arm64 and more. v1 already uses it (`~/Git/SlipStream-v1/go.mod`) |
| [mattn/go-sqlite3](https://github.com/mattn/go-sqlite3) | Go | MIT | cgo. v1.14.52, 2026-09-05 |
| [ncruces/go-sqlite3](https://github.com/ncruces/go-sqlite3) | Go | MIT | SQLite compiled to Wasm, no cgo. v0.35.6 |
| [bbolt](https://github.com/etcd-io/bbolt) | Go | MIT | Key-value. v1.5.0 |
| [rusqlite](https://github.com/rusqlite/rusqlite) | Rust | MIT | Can bundle SQLite. v0.40.2, 117.8M downloads |
| [sqlx](https://github.com/transact-rs/sqlx) | Rust | Apache-2.0 | Async SQL with SQLite support. Repo moved from launchbadge to transact-rs |
| [redb](https://github.com/cberner/redb) | Rust | Apache-2.0 | Pure Rust key-value. v4.3.0 |
| [Microsoft.Data.Sqlite](https://www.nuget.org/packages/Microsoft.Data.Sqlite) with [SQLitePCL.raw](https://github.com/ericsink/SQLitePCL.raw) | C# | MIT, Apache-2.0 | 10.0.12, 129.6M downloads. Uses a native SQLite library per platform |

**Inferred:** this need does not separate the candidates either. The difference is build shape. Only Go has a mature SQLite that needs no C toolchain, which matters for need 8.

### 7. Web UI

The UI is redesigned from scratch, prototype first (Q26). **Inferred:** any backend serves a built single-page app, so this need does not separate the candidates. Embedding options are Go's standard [`embed`](https://pkg.go.dev/embed) package, Rust's [rust-embed](https://crates.io/crates/rust-embed) (8.12.0, 56.9M downloads), and .NET's [Microsoft.Extensions.FileProviders.Embedded](https://www.nuget.org/packages/Microsoft.Extensions.FileProviders.Embedded) (475.5M downloads).

### 8. Single-binary cross-platform packaging plus Docker

- **Go.** cgo "is disabled by default when cross-compiling" ([cmd/cgo](https://pkg.go.dev/cmd/cgo)). A cgo-free build cross-compiles every target from one machine. [GoReleaser](https://github.com/goreleaser/goreleaser) (MIT, v2.18.2) builds archives, deb, rpm and Docker images. Embedding libtorrent would make the build cgo, which brings back a per-target C++ toolchain (**inferred**).
- **Rust.** Cross-compiling needs a target linker and C toolchain for any C dependency. [cargo-dist](https://github.com/axodotdev/cargo-dist) (Apache-2.0, v0.33.0) builds installers. [cross](https://github.com/cross-rs/cross) (Apache-2.0) has had no release since v0.2.5 on 2023-02-04.
- **.NET.** Single-file publish works for framework-dependent and self-contained apps. "Single file apps are always OS and architecture specific." Native libraries are separate files unless `IncludeNativeLibrariesForSelfExtract` extracts them to disk at startup, and under systemd that extraction needs `DOTNET_BUNDLE_EXTRACT_BASE_DIR` set ([single-file docs](https://learn.microsoft.com/en-us/dotnet/core/deploying/single-file/overview)). Native AOT "does not support cross-OS compilation" ([cross-compile docs](https://learn.microsoft.com/en-us/dotnet/core/deploying/native-aot/cross-compile)). It forbids runtime code generation and dynamic loading, and supports only some ASP.NET Core features ([Native AOT docs](https://learn.microsoft.com/en-us/dotnet/core/deploying/native-aot/)).

Docker images are straightforward in all three. **Inferred:** a static Go or Rust binary fits a scratch or distroless image. A .NET app fits Microsoft's runtime-deps images.

### 9. Fully faked offline mode

Offline mode needs a fake at every boundary, run with network egress blocked in CI (Q17). What each language offers to build those fakes:

| Concern | Go | Rust | .NET |
|---|---|---|---|
| Virtual time | [`testing/synctest`](https://pkg.go.dev/testing/synctest). A bubble has a fake clock that advances only when every goroutine is durably blocked. GA in Go 1.25. Loopback sockets block the bubble, so tests use `net.Pipe` | [`tokio::time::pause`](https://docs.rs/tokio/latest/tokio/time/fn.pause.html). Pauses Tokio's `Instant` on the current-thread runtime and auto-advances when idle | [`TimeProvider`](https://learn.microsoft.com/en-us/dotnet/standard/datetime/timeprovider-overview) in .NET 8+, with `FakeTimeProvider` in Microsoft.Extensions.TimeProvider.Testing (47.1M downloads) |
| Simulated network across many Nodes | No dedicated framework found. gVisor netstack can host in-process networks (**inferred** from its use in wireguard-go) | [turmoil](https://github.com/tokio-rs/turmoil). Runs many hosts in one thread with injected latency, drops, partitions and crashes, plus simulated sockets and filesystem. 16.1M downloads | No dedicated framework found |
| Filesystem | `io/fs` and `testing/fstest` in the standard library | turmoil-fs | [System.IO.Abstractions](https://github.com/TestableIO/System.IO.Abstractions) (151.3M downloads) |
| ACME | Pebble in-process | Pebble as a process | Pebble as a process |
| Tracker | anacrolix/torrent ships a tracker server package ([tracker/server](https://github.com/anacrolix/torrent/tree/HEAD/tracker/server)) | Not found | Not found |

**Inferred:** Rust has the strongest multi-Node simulation tool, which suits the Controller and worker Node topology. Go's synctest covers time but not cross-host faults.

## Other candidates considered

- **C++ around libtorrent-rasterbar.** Solves the torrent need natively and is what qBittorrent and Deluge build on (Deluge "uses libtorrent at its core", [README](https://github.com/deluge-torrent/deluge)). **Inferred:** every other need (ACME, userspace WireGuard, web server, offline-mode fakes) would be hand-assembled from C and C++ libraries, with the largest build and memory-safety cost of any option.
- **JVM (Kotlin or Java) with libtorrent4j.** The only maintained non-Python libtorrent binding found. **Inferred:** single-binary packaging means GraalVM native-image or a bundled runtime, and the JVM has no userspace WireGuard found in this pass. Not investigated further.
- **TypeScript on Node or Bun.** [WebTorrent](https://github.com/webtorrent/webtorrent) (MIT, v3.0.21) is the main engine. It is not on the sampled whitelist. Not investigated further.
- **A hybrid.** A primary language plus one embedded component in another, for example a Go core with libtorrent behind a C shim, or a .NET core calling a Go or Rust tunnel. **Inferred:** this trades one gap for a two-toolchain build and an FFI seam that needs its own fake in offline mode.

## Precedent in this domain

| Project | Language | What it shows |
|---|---|---|
| SlipStream v1 | Go 1.25, modernc SQLite, Echo | The chef's current stack (`~/Git/SlipStream-v1/go.mod`) |
| [Sonarr](https://github.com/Sonarr/Sonarr) and the other *arr apps | C# | The media-manager category is mostly .NET |
| [Jellyfin](https://github.com/jellyfin/jellyfin) | C# | A .NET Media Server with multi-backend hardware transcoding |
| [qBittorrent](https://github.com/qbittorrent/qBittorrent), [Deluge](https://github.com/deluge-torrent/deluge) | C++, Python over libtorrent | Whitelisted private-tracker clients |
| [Tailscale](https://github.com/tailscale/tailscale), [Syncthing](https://github.com/syncthing/syncthing), [CertMagic](https://github.com/caddyserver/certmagic) (Caddy's TLS engine) | Go | Userspace tunnel, file sync and ACME at scale |
| [rqbit](https://github.com/ikatson/rqbit) | Rust | A full torrent client with web UI as one binary |

## Tradeoffs for the decision

These are the axes the stack decision turns on. None is weighed here.

1. **Torrent engine identity.** Native Go, Rust or .NET engines need whitelisting at each private tracker. libtorrent is whitelisted but needs an FFI shim in any of the three. A prototype could measure how hard a minimal libtorrent shim is in each language.
2. **In-process coverage.** Go covers all nine needs in-process with first-party or widely deployed libraries. Rust covers all but a whitelisted engine. .NET has gaps in tunnel, ACME integration and Linux QUIC.
3. **Build shape.** A cgo-free Go build cross-compiles from one machine. Rust needs per-target toolchains for C dependencies. .NET Native AOT needs one build machine per OS.
4. **Offline-mode tooling.** Rust's turmoil simulates many Nodes with faults. Go's synctest fakes time and needs in-memory pipes for I/O. .NET has strong time and filesystem fakes but no network simulator found.
5. **Domain precedent and v1 familiarity.** .NET has the *arr and Jellyfin precedent. Go is what v1 is written in.
6. **License shape.** anacrolix/torrent, GotaTun, Syncthing and Pebble are MPL-2.0, which is file-level copyleft. The rest are permissive.

## Open questions this research could not settle

- Whether the trackers the chef uses would whitelist a new client, and on what terms. This is a question for those trackers' staff, not a source to read.
- Whether anacrolix/torrent or librqbit sustain private-tracker seeding loads (thousands of torrents, accurate stats). A prototype against a local tracker could measure it.
- Whether MonoTorrent's master branch has unreleased work worth tracking. Its last stable release is v3.0.2 from 2024.

## Appendix A. Maturity snapshot

Read from the GitHub REST API on 2026-10-04. "no release" means the repo publishes tags but no GitHub releases.

| Repository | Stars | License | Last push | Latest release |
|---|---|---|---|---|
| anacrolix/torrent | 6,121 | MPL-2.0 | 2026-09-27 | no release |
| cenkalti/rain | 1,146 | MIT | 2026-10-03 | v2.4.2, 2026-10-03 |
| arvidn/libtorrent | 6,080 | BSD-3-Clause | 2026-10-04 | v2.1.2, 2026-09-25 |
| ikatson/rqbit | 1,920 | Apache-2.0 | 2026-10-01 | v9.0.1, 2026-08-20 |
| alanmcgovern/monotorrent | 1,242 | MIT | 2026-07-31 | v3.0.2, 2024-08-04 |
| aldenml/libtorrent4j | 284 | see repo | 2026-02-07 | v2.1.0-39, 2026-02-07 |
| WireGuard/wireguard-go | 4,432 | MIT | 2026-05-22 | no release |
| tailscale/tailscale | 37,155 | BSD-3-Clause | 2026-10-04 | v1.102.5, 2026-09-29 |
| tailscale/libtailscale | 343 | BSD-3-Clause | 2026-08-31 | no release |
| juanfont/headscale | 44,349 | BSD-3-Clause | 2026-10-04 | v0.29.4, 2026-09-23 |
| google/gvisor | 19,530 | Apache-2.0 | 2026-10-05 | release-20260928.0 |
| cloudflare/boringtun | 7,208 | BSD-3-Clause | 2026-06-29 | crate 0.7.1, 2026-05-01 |
| mullvad/gotatun | 1,371 | MPL-2.0 | 2026-09-16 | v0.9.2, 2026-08-31 |
| smoltcp-rs/smoltcp | 4,611 | 0BSD | 2026-10-03 | v0.14.0, 2026-08-17 |
| caddyserver/certmagic | 5,605 | Apache-2.0 | 2026-10-01 | v0.25.6, 2026-09-30 |
| go-acme/lego | 9,905 | MIT | 2026-09-27 | v5.5.2, 2026-09-23 |
| letsencrypt/pebble | 793 | MPL-2.0 | 2026-09-28 | v2.10.1, 2026-04-20 |
| rustls/rustls | 7,654 | Apache-2.0, MIT or ISC | 2026-10-04 | v0.23.45, 2026-09-14 |
| djc/instant-acme | 229 | Apache-2.0 | 2026-09-28 | 0.8.5, 2026-02-24 |
| FlorianUekermann/rustls-acme | 192 | Apache-2.0 or MIT | 2026-07-31 | crate 0.15.4 |
| fszlin/certes | 573 | MIT | 2026-10-05 | v4.1.0, 2026-09-27 |
| natemcmaster/LettuceEncrypt | 1,683 | Apache-2.0 | 2025-04-24 | v1.3.3, 2024-05-20. Archived |
| quic-go/quic-go | 11,794 | MIT | 2026-10-05 | v0.63.0, 2026-09-22 |
| quinn-rs/quinn | 5,281 | Apache-2.0 | 2026-10-02 | quinn-proto 0.11.19, 2026-09-30 |
| microsoft/msquic | 4,780 | MIT | 2026-10-03 | v2.6.2, 2026-09-30 |
| mattn/go-sqlite3 | 9,247 | MIT | 2026-09-28 | v1.14.52, 2026-09-05 |
| ncruces/go-sqlite3 | 1,115 | MIT | 2026-10-01 | v0.35.6, 2026-09-23 |
| etcd-io/bbolt | 9,767 | MIT | 2026-09-15 | v1.5.0, 2026-06-21 |
| rusqlite/rusqlite | 4,426 | MIT | 2026-09-25 | v0.40.2, 2026-08-08 |
| transact-rs/sqlx | 17,530 | Apache-2.0 | 2026-10-04 | no release |
| cberner/redb | 4,825 | Apache-2.0 | 2026-10-05 | v4.3.0, 2026-09-15 |
| ericsink/SQLitePCL.raw | 624 | Apache-2.0 | 2026-08-13 | v3.0.5, 2026-07-27 |
| rosenbjerg/FFMpegCore | 2,100 | MIT | 2026-09-23 | v5.5.0, 2026-09-20 |
| nathanbabcock/ffmpeg-sidecar | 543 | MIT | 2026-09-30 | v2.6.0, 2026-09-30 |
| zmwangx/rust-ffmpeg | 2,003 | WTFPL | 2026-09-16 | v9.0.0, 2026-08-05 |
| u2takey/ffmpeg-go | 2,353 | Apache-2.0 | 2024-05-24 | v0.5.0, 2023-08-08 |
| tokio-rs/turmoil | 1,267 | MIT | 2026-10-04 | v0.7.2, 2026-04-24 |
| goreleaser/goreleaser | 16,088 | MIT | 2026-10-04 | v2.18.2, 2026-09-17 |
| axodotdev/cargo-dist | 2,128 | Apache-2.0 | 2026-10-04 | v0.33.0, 2026-09-11 |
| cross-rs/cross | 8,325 | Apache-2.0 | 2026-09-24 | v0.2.5, 2023-02-04 |
| TestableIO/System.IO.Abstractions | 1,626 | MIT | 2026-10-01 | v22.3.0, 2026-09-28 |
| webtorrent/webtorrent | 31,429 | MIT | 2026-10-04 | v3.0.21, 2026-07-27 |

Licenses the GitHub API reported as NOASSERTION were read from the repo's LICENSE file (rqbit Apache-2.0, libtorrent BSD-3-Clause) or the crate metadata (rustls, rustls-acme).
