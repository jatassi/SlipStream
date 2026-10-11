# jatassi/SlipStream #1 (v2 map): the 53 sub-issues, with an eye to library choices

Sources: the 53 sub-issue bodies from the issue_read dump, every comment on the 48 closed sub-issues (fetched with `gh api .../comments`, 68 comments, about 190 KB), and the eight research findings docs the research issues link to (`docs/research/*.md` on the `research/*` branches). Quotes are verbatim. "Findings doc" means the linked research doc, not the issue text. Raw dumps sit next to this file: `raw.md` holds the bodies, `comments.md` the comments, and `docs/*.md` the findings docs.

Glossary used throughout: Controller and Worker are Nodes, a Role is a subsystem a Node runs, a Transfer is a Node-to-Node file copy, a Conversion is an A/V pipeline job, a Work is a movie or series, and the Fake world is the offline-mode harness.

---

## 1. Every sub-issue

Labels are `wayfinder:<type>`. 48 are closed (46 completed, 2 not_planned) and 5 are open.

| # | Title | State | Type | Gist |
|---|---|---|---|---|
| 2 | Chart SlipStream v2 | closed | grilling | 27 charting answers: Go/Rust/.NET open, Controller and Workers, built-in torrent client, A/V pipeline, subtitles, relay, offline mode, parity bar |
| 3 | Build the parity matrix from the v1 inventories | closed | task | `docs/parity-matrix.md` has 359 rows with P-### IDs. Exceptions need the chef's yes |
| 4 | Which v1 integrations stay beside the built-ins | closed | grilling | Keeps qBit, Transmission, Deluge and rTorrent. Drops external Prowlarr mode for a generic Torznab indexer, drops usenet entirely, keeps Cardigann as a fresh runner |
| 5 | The Content Type contract | closed | grilling | Compiled-in modules (no plugins). The core owns the tree, files, token engine and quality. Music is a test-only proof type (ADR 0001) |
| 6 | The Edition and Target model | closed | grilling | Edition × Target. Targets have quality order, cutoff and sourcing policy. Global blocklist by info hash. Propers and repacks (ADR 0002) |
| 7 | Custom formats engine and TRaSH Guides import | closed | grilling | .NET-syntax regex with a per-match timeout (regexp2). Tracked TRaSH import with diffs (ADR 0006) |
| 8 | Stack candidates against the built-ins' needs | closed | research | Go is the only language with an in-process library for all 9 needs. The torrent engine is the deciding constraint |
| 9 | Prowlarr catalog: vendor or reimplement | closed | research | 577 Cardigann YAML files plus 69 C# indexers. GPL-3.0 implications. v1's Go runner has gaps |
| 10 | Torrent engines at private-tracker grade | closed | research | Only libtorrent is whitelisted and proven at thousands of torrents. librqbit is not private-ready |
| 11 | Choose the stack | closed | grilling | **Go 1.27, cgo libtorrent, SQLite (modernc) + sqlc + goose, React 19/Vite/TanStack, Huma, Orval, SSE, yamux + ConnectRPC** (ADR 0013) |
| 12 | Node topology | closed | grilling | Join Code pairing, pinned keys, Roles and Role groups, lockstep versions, Handover (ADR 0003) |
| 13 | Fast transfer between Nodes | closed | research | Multi-stream TCP. Segmenting is a pull. BBR needs privileges. QUIC needs several connections |
| 14 | File sync design | closed | grilling | Transfer: the receiver pulls verified chunks over extra pinned-TLS connections, with BLAKE3 per chunk (ADR 0004) |
| 15 | CGNAT, remote access and relay techniques | closed | research | UPnP/NAT-PMP/PCP compared with STUN for CGNAT detection. Relay options: wireguard-go netstack, tsnet, frp, rathole, hosted tunnels |
| 16 | Relay Role and reachability diagnostic | closed | grilling | The relay forwards over the Node link with **no WireGuard**. SNI passthrough. ACME TLS-ALPN-01 on the home Node (ADR 0012) |
| 17 | ffmpeg hardware acceleration and HDR handling | closed | research | NVENC, QSV, VAAPI, VT, AMF. Stacked capability probes. Only libplacebo or jellyfin-ffmpeg handle DV P5 |
| 18 | A/V pipeline design | closed | grilling | "Conversion" runs on a bundled, pinned jellyfin-ffmpeg plus dovi_tool. Trial-run capability proof. The Controller schedules (ADR 0007) |
| 19 | Subtitle providers and timing sync | closed | research | Bazarr catalog, OpenSubtitles terms, ffsubsync (MIT) compared with alass (GPL) |
| 20 | Subtitle pipeline design | closed | grilling | SubDL, SubSource and Gestdown. Native Timing sync = libfvad + an FFT port from ffsubsync (ADR 0008) |
| 21 | Plex, Jellyfin and Emby APIs | closed | research | Path-scoped refresh on all three. Only Jellyfin exposes device profiles. Auth differences |
| 22 | Media Server contract | closed | grilling | Typed change refresh, path mappings, session tracking, a Playback report, wire-level fakes (ADR 0009) |
| 23 | Identity, PIN and passkey, admin permission, API key | closed | grilling | One Account, admin as a flag, PIN + passkey, opaque session cookie, one global API key (ADR 0005) |
| 24 | Portal onboarding flow and emails | closed | prototype | Variant B, guided steps, with Sign in with Plex and an auto-accepted share |
| 25 | UI redesign direction | closed | prototype | Variant C "Lit console": Geist, artwork-tinted materials, Apple-like controls |
| 26 | Offline mode boundary | closed | grilling | The Fake world harness. Wire-level fakes for 47 boundaries. Embedded reference implementations (ADR 0014) |
| 27 | verify-slipstream skill design | closed | grilling | One verify skill over the Fake world. Playwright CLI, `fakeworld` commands, an evidence dir |
| 28 | v1 importer scope | closed | grilling | One Import feature with v1, Radarr and Sonarr readers. Credentials are not carried over |
| 29 | **The feature graph** | **open** | grilling | Spec order as sub-issues with blockedBy edges, each written with /to-spec. All 42 blockers are closed |
| 31 | SlipStream v2's license | closed | grilling | GPL-3.0. Cardigann YAML is pulled at runtime and never bundled. Third-party notices and a CI license allow-list (ADR 0010) |
| 32 | How Media Servers classify viewers through a relay | closed | prototype | Relayed viewers look local, accepted. wireguard-go netstack measured at 645–966 Mbit/s |
| 33 | Torrent engine bakeoff | closed | prototype | **libtorrent chosen**: about 25 MiB per 1000 seeds, against 320 for rain and 660 for anacrolix. The C shim is 8 functions |
| 42 | Import pipeline under Editions and Targets | closed | grilling | "Fills" semantics, one folder per Work, `{Target}` and `{Edition Tag}` tokens (ADR 0011) |
| 61 | Work page at scale | closed | prototype | A · Grid plus C's "Needs you" list. Home tint comes from the most important job |
| 62 | Nodes, Roles, Pairing and Handover screens | closed | prototype | F: a topology canvas with traffic, plus a Roster tab |
| 63 | Conversion queue and backfill dialog | closed | prototype | B · Ledger |
| 64 | Built-in torrent client screens | closed | prototype | B · By state. Tracker routing and seeding goals |
| 65 | Relay diagnostics screens | closed | prototype | "Path": per-hop route checks |
| 68 | OpenSubtitles.com without ads | closed (not_planned) | task | Could not test VIP or contact OpenSubtitles, so OpenSubtitles stays out |
| 69 | Subtitle Provider setup and subtitle screens | closed | prototype | B · Ranked list with a guided key sheet |
| 71 | Media Server conformance suite | closed | prototype | Real Plex 1.43.4, Jellyfin 12.2 and Emby 4.10 in containers. `{edition-…}` breaks grouping. Refresh delays of 60 and 90 s |
| 74 | Prove Plex sign-in, share and auto-accept against real Plex | closed | task | PIN sign-up works and share auto-accept works, but Plex still emails an invite |
| 85 | Edition file names every Media Server groups | closed | grilling | `Heat (1995) - {edition-Director's Cut} 4K HDR.mkv`, with the tag after the dash |
| 87 | Search, RSS sync and grab across Nodes | closed | grilling | Tracker and Indexer are separate. One home Node per Indexer. A faithful Go port of the Cardigann runtime. FlareSolverr-API solver URL (ADR 0017) |
| 88 | Requests portal under Editions and Targets | closed | grilling | 32 answers on requests, limits, status, watchers and subtitle reports (ADR 0020) |
| 89 | Where Editions are wanted: Work, level or Item | closed | grilling | Work-wide. A listed non-Standard cut counts only once it has been seen |
| 90 | Moving torrents and library files between Nodes | closed | grilling | A Move is per Work or torrent. Torrent handoff respects tracker windows (ADR 0016) |
| 91 | Config backup and restore | closed | grilling | Backup = `VACUUM INTO` db → `.tar.zst.age`. A restored Controller takes a new key (ADR 0018, 0019) |
| 92 | More than one Relay Node | closed (not_planned) | grilling | Out of scope: one Relay Node only |
| 93 | Packaging, the updater and the update dashboard | closed | grilling | musl-static/MinGW-static, minisign-signed manifest, self-replace, no paid signing (ADR 0015) |
| 94 | **Design system and DESIGN.md** | **open** | grilling | Distil the prototypes into tokens, type and components in Impeccable's DESIGN.md format |
| 101 | **How Media Servers handle a file moving to a new path** | **open** | prototype | Conformance run for the "moved" change. The Move spec and the Media Server fakes wait on it |
| 106 | **Tracker detail page** | **open** | prototype | UI for a Tracker as its own entity (from #87) |
| 107 | **Manual search sheet under Editions and Targets** | **open** | prototype | Layout of verdicts per Target per Release (from #87 Q15) |

---

## 2. Libraries, tools, protocols and OSS projects named or evaluated

Verdicts: **CHOSEN** means adopted for v2. **REJECTED** means explicitly ruled out. **DEFERRED** means it waits on maturity. **PRECEDENT** means cited as prior art only. **CONSIDERED** means evaluated with no verdict, or superseded by another choice.

### 2.1 Core stack (#11, ADR 0013), which supersedes the #8 comparison

#11 resolution: "**SlipStream v2 is one Go binary with an embedded React app, linking libtorrent through cgo.**"

| Item | Purpose | Verdict | Reason / quote | Issue |
|---|---|---|---|---|
| Go 1.27 | Language for the Controller and every Worker | CHOSEN | #8: "Go is the only candidate with a maintained, in-process library for all nine needs, and its cgo-free build cross-compiles from one machine." | #8, #11 |
| Rust | Language | REJECTED | "Rust is out. No regex engine runs .NET syntax, which ADR 0006 needs." #8 had noted Rust "has the strongest multi-Node simulation tool (turmoil)" | #8, #11 |
| .NET | Language | REJECTED (came second) | ".NET came second. It would have kept Prowlarr's C# and native regex. It loses on CertMagic's TLS-ALPN-01, the Worker footprint, and v1 being Go." #8: "no maintained userspace WireGuard, an archived Kestrel ACME integration, a system `libmsquic` dependency on Linux, and no cross-OS Native AOT." | #8, #11 |
| C++ around libtorrent, JVM + libtorrent4j, TypeScript + WebTorrent | Language | CONSIDERED, not pursued | Findings doc #8, "Other candidates considered": "Not investigated further." | #8 |
| SQLite in WAL mode, one database on the Controller | Database | CHOSEN | "One SQLite database in WAL mode on the Controller. There is no Postgres." Workers keep "a disposable local SQLite file for their outbox and resume data." | #11 |
| modernc.org/sqlite | SQLite driver | CHOSEN | "modernc's pure-Go driver, with sqlc generating typed Go from SQL." Findings doc #8: "Only Go has a mature SQLite that needs no C toolchain" | #8, #11 |
| mattn/go-sqlite3, ncruces/go-sqlite3, bbolt | Database | CONSIDERED, not picked | Findings doc #8 table | #8 |
| sqlc | Typed SQL codegen | CHOSEN | as above | #11 |
| goose | Migrations | CHOSEN | "goose migrations kept per module, so the core and each Content Type own their tables." #91: a restore migrates "forward with goose" | #11, #91 |
| React 19.3, TypeScript 7, Vite 8, React Compiler | UI | CHOSEN | "React 19.3, TypeScript 7 and Vite 8, with React Compiler on." | #11 |
| TanStack Router, Query, Virtual; Tailwind 4; Motion | UI | CHOSEN | "TanStack Router, Query and Virtual, Tailwind 4 and Motion." | #11 |
| Oxlint, Oxfmt, Vitest | Frontend tooling | CHOSEN | "Oxlint, Oxfmt and Vitest as tooling." | #11 |
| TanStack DB 0.13, Vite+ | UI | DEFERRED | "TanStack DB (0.13) and Vite+ wait until they mature." | #11 |
| Server-rendered frameworks | UI | REJECTED | "Server-rendered frameworks are out, because there's no JavaScript server." | #11 |
| `<ViewTransition>` | UI transitions | TO VERIFY | "The walking skeleton checks that `<ViewTransition>` works with TanStack Router's navigation." | #11 |
| Component layer | UI components | OPEN, deferred to #94 | "The component layer stays with [Design system and DESIGN.md](#94)." | #11, #94 |
| Huma v2 on `net/http` | REST API and OpenAPI 3.1 | CHOSEN | "JSON REST, code-first with Huma v2 on `net/http`, which generates OpenAPI 3.1. CI fails when the committed document drifts." (v1 used Echo) | #11 |
| Orval 8 | Generates TanStack Query hooks, Zod and MSW mocks | CHOSEN | "Orval 8 generates TanStack Query hooks, Zod schemas and MSW mocks." | #11 |
| Zod, MSW | Schemas and UI mocks | CHOSEN (generated) | Orval output. #26: "P-261 is superseded by generated MSW mocks." | #11, #26 |
| Server-Sent Events | Live updates | CHOSEN | "one Server-Sent Events stream per session. Its typed events are declared in the OpenAPI document." It replaces v1's WebSocket (P-241, P-345) | #11 |
| TLS 1.3 with pinned Ed25519 keys + yamux | Node link transport | CHOSEN | "Transport is TLS 1.3 with each Node's pinned Ed25519 key, multiplexed by yamux." | #11, #12 |
| ConnectRPC (connect-go 2.x), protobuf, `buf breaking` | Node-to-Node RPC | CHOSEN | "ConnectRPC protobuf services run on both sides, so calls go both ways. connect-go is locked to 2.x." "`buf breaking` in CI guards them." | #11 |
| `slog` | Logging | CHOSEN | "`slog`, and plain constructors with no DI framework." | #11 |
| DI framework (v1 uses Wire) | DI | REJECTED | "plain constructors with no DI framework" | #11 |
| dlclark/regexp2 | .NET-syntax regex for Custom Formats | CHOSEN | #7: "71 of 1,342 unique TRaSH regexes fail under Go's `regexp`... Go's `regexp2` compiles all 1,342." #11: "`regexp2` for Custom Formats" | #7, #11 |
| Go `regexp` (RE2) | Regex | REJECTED for Custom Formats | Same reason: 71 TRaSH regexes fail under it | #7 |
| CertMagic | ACME / TLS | CHOSEN | "CertMagic for ACME." #16: the home Node "gets its certificate from ACME with TLS-ALPN-01" | #8, #11, #15, #16 |
| lego, `x/crypto/acme/autocert` | ACME | CONSIDERED | Findings docs #8 and #15. autocert "has no DNS-01" | #8, #15 |
| gonum FFT | Timing sync FFT | CHOSEN | "gonum's FFT plus vendored libfvad through cgo for Timing sync." | #11 |
| libfvad (BSD, C) | Voice activity detection | CHOSEN (vendored, cgo) | #20: "libfvad (BSD) for speech detection" | #11, #19, #20, #31 |
| `golang.org/x/sys` | Suspend and resume ffmpeg and dovi_tool | CHOSEN | "`golang.org/x/sys` to suspend and resume ffmpeg and `dovi_tool`." | #11, #18 |
| `testing/synctest`, `net.Pipe` | Fake time and in-memory I/O for simulation tests | CHOSEN | "Node link simulation runs yamux over `net.Pipe` behind a fault wrapper, and `testing/synctest` fakes time." #26 Q18: "There is no product Clock interface." | #11, #26 |
| turmoil (Rust) | Multi-node simulation | REJECTED with Rust | Findings doc #8: "strongest multi-Node simulation tool found" | #8 |
| GoReleaser | Packaging | CONSIDERED (findings doc #8) | Not mentioned in the #93 resolution | #8 |
| Go `embed` | Embedding the SPA | Implied | "The app is embedded in the binary." | #8, #11 |

### 2.2 Torrent engine (#10, #33, #4, #64, #90)

| Item | Verdict | Reason / quote | Issue |
|---|---|---|---|
| **libtorrent-rasterbar** (C++, BSD-3) via a C shim and cgo | **CHOSEN** | #33: "**SlipStream v2 embeds libtorrent-rasterbar**, picked by Jackson on the measurements below." "libtorrent: about 25 MiB, with CPU flat at 0.14% of a core." Shim: "8 functions in 281 lines... A production shim is about 40 to 60 functions. That is 2 to 3 weeks, plus 1 to 2 for builds and CI (a guess)." Cost: "Go loses its pure cross-compile, and each OS needs a native CI job with libtorrent, boost and OpenSSL." #8: "libtorrent-rasterbar is [on the whitelist], and it needs a hand-owned FFI shim in any of the three languages." | #8, #10, #33, #11 |
| anacrolix/torrent (Go, MPL-2.0) | REJECTED | "about 660 MiB... On a 2-tier x 2-URL private torrent, the tracker rejected 1506 of its 1798 announces." "no clean way to flush `stopped` on shutdown." Not on the whitelist | #10, #33 |
| cenkalti/rain (Go, MIT) | REJECTED | "about 320 MiB. CPU rises to 14.6% of a core at 3000 torrents, and it opens a listening port per torrent." | #10, #33 |
| librqbit (Rust) | REJECTED | #10: "librqbit is not private-tracker ready. Its code says private trackers are 'not fully implemented', and its HTTP announces never send `completed` or `stopped`." | #10 |
| MonoTorrent (C#), libtransmission (GPL), WebTorrent | CONSIDERED, not pursued | Findings docs #8 and #10 | #8, #10 |
| Existing libtorrent bindings: libtorrent4j, lt-rs, libtorrent-sys, i96751414/libtorrent-go, libtorrent-net | REJECTED | Findings doc #8: "none of these is a maintained base for production use." | #8 |
| qBittorrent-nox as a supervised sidecar | CONSIDERED (option 3 in #10) | Superseded by embedding. qBittorrent stays as an external client (below) | #10 |
| Seeding-goal policy | BUILT BY SLIPSTREAM (see §3) | #10: libtorrent's "ratio and seed-time limits only reorder the queue, so qBittorrent adds its own stop or remove policy on top." | #10, #64 |
| External clients: qBittorrent, Transmission, Deluge, rTorrent | KEPT as integrations | #4: "They cover nearly every seedbox and tracker allow-list. They are the fallback while a new peer_id prefix waits on Gazelle allow-lists." | #4 |
| Flood, Aria2, Synology DS, Freebox, rqbit, Tribler, Vuze, uTorrent, Hadouken | DROPPED | #4 and #3 exceptions | #3, #4 |
| SABnzbd, NZBGet, Newznab (usenet) | DROPPED | #4: "Usenet leaves v2: no Newznab indexers, no NZB grabs, no usenet clients." | #4 |
| Peer ID `-SS2000-` | v2's own client identity | #64 and #90 reference "a tracker whose allow list lacks `-SS2000-`" | #64, #90 |
| Fake tracker: anacrolix `tracker/server`, chihaya, Ocelot, libtorrent libsimulator | CONSIDERED (findings doc #10). The actual fake is a loopback tracker built in-repo with UNIT3D rules | #33: "the loopback tracker (UNIT3D rules plus an in-memory swarm) and the fixture generator are the first offline-mode fake." #26 Q28: "Real libtorrent runs end to end with DHT, LSD, UPnP and NAT-PMP off" | #10, #26, #33 |

### 2.3 Indexers, Prowlarr and Cardigann (#9, #4, #31, #11, #87)

| Item | Verdict | Reason / quote | Issue |
|---|---|---|---|
| Prowlarr code (C#, GPL-3.0) linked or embedded | REJECTED (wrong stack) | #9: "Linking it into SlipStream, or porting it, binds SlipStream to GPL-3.0 when shipped." | #9 |
| Prowlarr as an external process (v1 "external Prowlarr mode") | DROPPED | #4: "External Prowlarr mode (P-118 to P-122) is dropped. A generic Torznab indexer replaces it" | #4 |
| Prowlarr's site-specific C# indexers | PORTED to Go | #11: "Every site-specific C# indexer is ported to a native Go indexer behind one indexer contract, with none deferred. That's about 99 files and 1.15 MB of C#." Allowed because v2 is GPL-3.0 (#31) | #11, #31 |
| Cardigann YAML catalog (`indexers.prowlarr.com`, schema v11) | Consumed at runtime, never bundled | #31: "never bundled (the files carry no license). The Controller pulls the catalog at runtime from a URL the admin can override" | #9, #31, #11 |
| Cardigann runner | BUILT (Go port) | #87 Q8: "The runner is a faithful Go port of Prowlarr's Cardigann runtime with all 22 filters, refusing definitions it can't fully run." (Earlier #4: "built as a fresh runner that copies nothing from v1's code.") | #4, #87 |
| Jackett definitions (GPL-2.0) | NOT used | Findings doc #9, option D3 | #9 |
| Torznab (generic) | CHOSEN as the generic indexer | #87 Q11: "Saved only after a `t=caps` probe." Reaches any Prowlarr or Jackett indexer | #4, #87 |
| FlareSolverr (MIT, headless Chrome) | EXTERNAL, user-run, not bundled | #87 Q10: "A Challenge solver URL per Searching Node accepts anything speaking FlareSolverr's API, such as Byparr. No solver is bundled." | #9, #87 |
| Byparr | Named as a compatible solver | as above | #87 |
| reCAPTCHA, hCaptcha, Turnstile | NOT supported | #87 Q10 | #87 |
| TRaSH Guides (MIT) | IMPORTED, tracked | #7 Q7: "Tracked import. Each item keeps its `trash_id`." Q8: "Releases bundle a TRaSH snapshot" | #7 |

### 2.4 Node link, transfer, relay and NAT (#12, #13, #14, #15, #16, #32, #92)

| Item | Verdict | Reason / quote | Issue |
|---|---|---|---|
| rclone, lftp (`pget`), aria2 | REJECTED as the transport | Findings doc #13, option A ("Drive an external tool"): "A second binary to ship... Segmenting works only as a pull." #14 chose its own protocol | #13, #14 |
| Syncthing (MPL-2.0) | PRECEDENT for the chunk protocol design | Findings doc #13: "closest open source design to Node-to-Node sync" | #13 |
| quic-go | NOT chosen (Transfer runs over TLS/TCP) | Findings doc #13: "quic-go ships Cubic only." #14: "The Transfer opens extra TLS connections" | #8, #13, #14 |
| BitTorrent between Nodes | REJECTED | #14: "BitTorrent between Nodes is out under BEP 27" | #13, #14 |
| BLAKE3 | CHOSEN for chunk hashing | #14: "The sender hashes each chunk with BLAKE3 as it reads it" | #14, #90 |
| tus, the IETF resumable-upload draft | CONSIDERED (findings doc #13) | — | #13 |
| wireguard-go + `tun/netstack` (gVisor) | REJECTED for the relay | #16: "There is no WireGuard and no second tunnel stack." #32 measured "645 to 966 Mbit/s on Apple M4 loopback" | #8, #15, #16, #32 |
| tsnet, libtailscale, Headscale | CONSIDERED, not chosen | Findings doc #8: "Using it ties the relaying Role to a Tailscale-compatible coordination server." | #8, #15 |
| BoringTun, GotaTun, smoltcp (Rust) | REJECTED with Rust | Findings doc #8 | #8 |
| frp, rathole, `ssh -R` | CONSIDERED, not chosen | Findings doc #15, relay design B | #15 |
| Cloudflare Tunnel, Tailscale Funnel, Plex Relay | REJECTED | Findings doc #15: Cloudflare "terms restrict video". Funnel allows only 3 ports and `ts.net` names. Plex Relay is capped at 2 Mbps | #15 |
| Pangolin/Newt (AGPL) | PRECEDENT only | "Copying its code would bring those terms into SlipStream." | #15 |
| UPnP IGD, NAT-PMP, PCP, STUN | Protocols used in the reachability diagnostic | #16: "gateway WAN address from UPnP, NAT-PMP or PCP against STUN, giving open, port-mappable, double NAT or CGNAT." No client library is named | #15, #16 |
| Tailscale `portmapper` and `netcheck` (BSD-3) | PRECEDENT. Tailscale's `TestIGD` is the model for the gateway fake | #26 Q30: "The gateway fake is built in-repo after Tailscale's `TestIGD`, and STUN on pion/stun" | #15, #26 |
| pion/stun | CHOSEN for the STUN fake | #26 Q5 and Q30 | #26 |
| Caddy, kernel WireGuard | The chef's current setup, observed only | #32 | #32 |
| DNS provider APIs | REJECTED | #92: "SlipStream never calls a DNS provider's API." | #92 |
| PROXY protocol, `X-Forwarded-For`, `X-Real-IP` | Considered. The decision is to forward as-is | #32: "The relay makes relayed viewers look local... SlipStream does not try to keep them classed as remote." | #15, #32 |

### 2.5 A/V, ffmpeg and Conversion (#17, #18, #26, #93)

| Item | Verdict | Reason / quote | Issue |
|---|---|---|---|
| **jellyfin-ffmpeg** (GPL-3.0), pinned and bundled per OS and arch | CHOSEN | #18: "The Controller sends each Conversion only to a Node that has proven those settings by a trial run, on a bundled, pinned jellyfin-ffmpeg." "It is GPL-3.0 and runs as a separate process." It also serves the import probe and subtitle Speech Maps | #17, #18, #20, #93 |
| Upstream FFmpeg, system ffmpeg | NOT chosen | Findings doc #17, option A. An admin override path per Node is allowed but "The UI marks it unsupported" | #17, #18 |
| libplacebo, zscale + tonemap, tonemap_opencl/vaapi, vpp_qsv, scale_vt | Tone mappers; which one runs comes from the Node's proven capability | #18: "The tone mapper comes from the Node's proven capabilities, with the operator as an advanced setting." | #17, #18 |
| **dovi_tool** (MIT) | CHOSEN | #18: "converts Dolby Vision Profile 7 to 8.1 with `dovi_tool` (MIT)" | #18, #31 |
| libx265, libsvtav1 | Used when DV must be kept (software only) | "Keeping Dolby Vision on an HDR output is opt-in and forces a software encode with libx265 or libsvtav1." | #17, #18 |
| mkvmerge | Allowed fallback | #26 Q27: "mkvmerge is allowed if ffmpeg can't write the Dolby Vision configuration record." | #26 |
| ffmpeg-go, ffmpeg-sidecar, rust-ffmpeg, FFMpegCore | NOT adopted | Findings doc #8: "`os/exec`" is the Go option. "this need does not separate the candidates" | #8 |
| Tdarr, Unmanic | PRECEDENT only. "Neither probes hardware or tone maps" | The scheduler is built in-house (§3) | #17 |
| Jellyfin `EncodingHelper.cs` (GPL-2.0) | PRECEDENT, "not a source to copy" | Findings doc #8 | #8, #17 |

### 2.6 Subtitles (#19, #20, #68, #69)

| Item | Verdict | Reason / quote | Issue |
|---|---|---|---|
| SubDL, SubSource, Gestdown | CHOSEN providers, implemented natively | #20: "SubDL, SubSource and Gestdown ship at launch, behind one provider contract." | #20 |
| OpenSubtitles.com | REJECTED for now | "stays out until it can ship without ads." #68 closed not_planned | #19, #20, #68 |
| Bazarr / subliminal / subliminal_patch (GPL-3.0) as a sidecar or port | REJECTED | Findings doc #19, option 3: "brings in a Python runtime and a second database." The `subliminal_patch` code is GPL | #19 |
| ffsubsync (MIT) | PORTED (algorithm), not run | #20: "split-penalty alignment ported from ffsubsync's MIT code. There is no Python and no GPL" | #19, #20, #31 |
| alass (GPL-3.0, unmaintained since 2019) | REJECTED | GPL and unmaintained. Its algorithm already exists in ffsubsync under MIT | #19 |
| auditok (GPLv3 pin in ffsubsync) | Avoided | Findings doc #19 | #19 |
| libfvad | CHOSEN | see 2.1 | #20 |
| OCR for PGS subtitles | NOT done | #20 Q16: "there is no OCR" | #20 |

### 2.7 Media Servers (#21, #22, #71, #74, #85, #101)

| Item | Verdict / use | Issue |
|---|---|---|
| Plex, Jellyfin, Emby APIs | First-class adapters behind one contract (ADR 0009). Wire-level fakes, plus a fake plex.tv | #21, #22 |
| Plex webhooks | NOT used: "Plex webhooks need Plex Pass and are not used." | #22 |
| Plex JWT auth (Ed25519 key, 7-day refresh) | CHOSEN | #22 |
| Wizarr (`_post_join_setup`) | PRECEDENT for Plex share auto-accept | #24 |
| Containers (Colima/Docker) running real Plex, Jellyfin and Emby | The conformance suite at `conformance/media-servers/` (PR #83) | #71 |

### 2.8 Identity, mail and portal (#23, #24, #26, #27)

| Item | Verdict / use | Issue |
|---|---|---|
| Passkeys / WebAuthn | CHOSEN feature. **No library named**. A "virtual WebAuthn authenticator" (Playwright, Chromium CDP) is the fake | #23, #26, #27 |
| Opaque session tokens in an HttpOnly cookie | CHOSEN, custom. v1's non-revocable JWTs are replaced | #23 |
| Generic SMTP | CHOSEN. The fake is go-smtp | #2, #26 |
| go-smtp | CHOSEN as a reference implementation in the Fake world | #26 Q5 |
| Notifiers: Discord, Telegram, Slack, Pushover, ntfy, Gotify, Apprise, Webhook, Custom Script | Kept as integrations. Gotify, ntfy, Pushbullet, Apprise and Custom Script are made to actually send | #3, #88 |

### 2.9 Offline mode, testing and verify (#26, #27, ADR 0014)

#26 Q5: "'Built from scratch' means not ported from v1. Standard protocols embed reference implementations: Pebble and challtestsrv, go-smtp, pion/stun, real libtorrent peers, and miniupnpd in the namespace tier."

| Item | Use | Issue |
|---|---|---|
| Pebble + challtestsrv (Let's Encrypt) | Embedded ACME CA fake: "Pebble and challtestsrv are embedded, with the randomness fixed." | #8, #26 |
| miniupnpd, nftables, Linux namespaces | Tier N NAT tests | #26 |
| SOCKS5 outside dialer, fake gateway, fake STUN | Tier L loopback | #26 |
| Playwright (CLI, virtual authenticator), Chromium host-resolver rules | e2e and verify | #26, #27 |
| ChaCha8 (Go `math/rand/v2`) | "one seed per process with a ChaCha8 stream per subsystem" | #26 |
| JSON Schema for Scenario YAML | "one strict YAML file, with a generated JSON Schema" | #26 |
| Geist, Geist Mono, Lucide | Bundled fonts and icons. "The browser contacts no third party." | #25, #26 |
| CI egress blocking (namespace block, sinkhole, canary) | DROPPED: "Nothing blocks egress, on any OS." | #26 |

### 2.10 Packaging, updater and backup (#91, #93)

| Item | Verdict / quote | Issue |
|---|---|---|
| musl-static (Linux), MinGW static (Windows), libSystem only (macOS) | "every third-party library is linked in." Enforced by a CI linkage allow-list | #93 |
| MSI, ZIP, DMG, deb, rpm, tarball, AppImage | Package set | #93 |
| Docker `debian:trixie-slim` + the jellyfin-ffmpeg deb, at `ghcr.io/jatassi/slipstream` | CHOSEN | #93 |
| minisign | Release manifest signing: "A minisign-signed `manifest.json` on GitHub Releases, verified against keys compiled into the binary." | #93 |
| SignPath Foundation | Windows code signing, if accepted. "Nothing paid." | #93 |
| Paid Apple notarization | REJECTED: "macOS ships ad-hoc signed and unnotarized" | #93 |
| age (passphrase), zstd, SQLite `VACUUM INTO`, `integrity_check` | Backup format `slipstream-<UTC>-<version>-<trigger>.tar.zst.age` | #91 |
| Built-in cloud backup destinations | REJECTED: "there are no built-in cloud destinations" | #91 |
| Windows service mode | REJECTED (Jackson's choice): "Windows has no service option" | #93 |
| Impeccable DESIGN.md format | CHOSEN for the design-system doc | #94 |

---

## 3. Where an issue says SlipStream builds something itself

| What | Quote / decision | Reason given | Issue |
|---|---|---|---|
| **Cardigann definition runner** | #4: "built as a fresh runner that copies nothing from v1's code." Refined in #87 Q8: "a faithful Go port of Prowlarr's Cardigann runtime with all 22 filters" | v1's runner is missing 4 filters (71 definition uses), full captcha and FlareSolverr (#9). GPL-3.0 (#31) allows a port | #4, #9, #87 |
| **About 99 Prowlarr C# indexers ported to native Go** | #11: "Every site-specific C# indexer is ported to a native Go indexer behind one indexer contract, with none deferred. That's about 99 files and 1.15 MB of C#" | The stack is Go, and the GPL-3.0 license allows porting (#31) | #11, #31 |
| **Indexer contract, Torznab client, rate limits, backoff, health** | #87 Q8–Q11 | Replaces external Prowlarr mode (#4) | #87 |
| **libtorrent C shim** (about 40–60 functions in production) | #33: "Embedding it means owning a C ABI shim" (findings doc #8). "A production shim is about 40 to 60 functions." | No maintained binding exists in any language | #8, #33 |
| **Seeding-goal and hit-and-run policy, tracker routing, announce-window handling** | #10: "anacrolix/torrent and rain are production Go libraries with no seeding-goal policy, so the host writes it either way." libtorrent's limits "only reorder the queue." #33: "UNIT3D rejects a resume `started` within about 27 minutes... SlipStream's pause and resume has to respect that window." #90 Q9: the torrent handoff protocol | No engine owns per-tracker goals | #10, #33, #64, #90 |
| **Transfer protocol** (chunked pull over extra pinned-TLS connections, BLAKE3, resume, adaptive connection count) | #14: "the receiver pulls in verified chunks." Matches findings doc #13 option C, the "Custom chunk protocol over N TCP connections": "Most code to own." | It works in either dial direction, needs no second binary, and avoids BEP 27 problems | #13, #14 |
| **Relay** (forwarding over the Node link, SNI passthrough, no WireGuard) | #16: "the relay forwards each public TCP or UDP stream down it. There is no WireGuard and no second tunnel stack." | Reuses the Node link. One tunnel stack | #16 |
| **Node link pairing and handshake** (Join Code, pinned keys, frozen handshake and update path) | #12, #11 Q9. Built on yamux and ConnectRPC | Lockstep updates need a frozen handshake | #12, #11 |
| **Reachability diagnostic** (UPnP/NAT-PMP/PCP vs STUN, connect-back) | #16. No client library named. Tailscale portmapper is prior art | — | #15, #16 |
| **Timing sync** | #20: "It is SlipStream's own code: libfvad (BSD) for speech detection, an FFT offset search, a framerate ratio search, and split-penalty alignment ported from ffsubsync's MIT code. There is no Python and no GPL, and it runs in the offline suite" | No Python and no GPL. Runs offline | #19, #20 |
| **Subtitle providers** (native SubDL, SubSource and Gestdown clients) | #20 | Bazarr or subliminal would bring a Python runtime and GPL | #19, #20 |
| **Custom formats engine + TRaSH tracked import** | #7. Uses regexp2 | Parity with Sonarr and Radarr. TRaSH regexes need .NET syntax | #7 |
| **Conversion scheduler and capability probing** (trial runs, earliest-finish placement) | #18 Q8–Q9 | Findings doc #17: Tdarr and Unmanic "Neither probes hardware or tone maps" | #17, #18 |
| **Offline-mode fakes / Fake world** (wire-level fakes for 47 boundaries, a fault proxy, `fakeworld` CLI, Scenario YAML) | #26 Q2: "Each boundary gets one wire-level fake speaking the real protocol." Q30: "The gateway fake is built in-repo after Tailscale's `TestIGD`" | Determinism and CI. Reference implementations are embedded only where they exist | #26, #27 |
| **Loopback private tracker + fake swarm** (UNIT3D rules) | #33 | First offline fake | #33, #90 |
| **Media Server adapters and fakes**, plus a fake plex.tv | #22: "Offline, one fake per kind speaks the subset of the real HTTP and socket API the contract uses" | Behavior copied from the conformance suite (#71) | #22, #71 |
| **Updater** (minisign manifest, self-replace, phased rollout, rollback) | #93 Q7–Q12 | No library named | #93 |
| **Backup format and restore/reconcile** | #91 | Built from age, zstd and `VACUUM INTO` | #91 |
| **Identity** (Accounts, PIN lockout, sessions, reset links, API key) | #23 | v1's lockout lived in memory, its JWTs couldn't be revoked, and its API key authenticated nothing | #23 |
| **Content Type module system** (compiled-in, token engine, build-time boundary check) | #5: "a compiled-in module behind one interface; no third-party plugins" | v1 had 149 type-branch sites outside modules | #5 |
| **Naming token engine** (`{Target}`, `{Edition Tag}`) | #42, #85 | Media Server grouping rules | #42, #85 |
| **Design system / component layer** | #94 (open), #11 | Prototype-led redesign | #11, #25, #94 |
| **v1, Radarr and Sonarr importer** | #28 | — | #28 |
| No DI framework | #11: "plain constructors with no DI framework" | — | #11 |

Note: #26 Q5 defines "built from scratch" as "not ported from v1", not as "no third-party libraries".

---

## 4. Open sub-issues and what they wait on

| # | Title | Waits on / status |
|---|---|---|
| **29** | **The feature graph** | Body: "What specs does v2 build in, and in what order? The first is the walking skeleton, and offline mode with pstack verification skill(s?) comes right after it. Deliverable is a graph of SlipStream v2 features as sub-issues under a new parent GitHub issue using blockedBy edges for dependencies. The body of each feature sub-issue must be populated using the /to-spec skill in the repo." It has **42 blockedBy edges (#3–#28, #31–#33, #42, #61–#65, #87–#93) and all are closed**, so it is unblocked, with no comments yet. Inputs handed to it: #27 says "#29: which fakes the skeleton carries". #27 also says "The walking skeleton and #93 need four things from the binary: a data-dir flag, a headless mode with no tray on every OS, a single-instance lock per data dir, and a system endpoint reporting the data dir and build stamp", plus "SSE by API key, a musl-static build in the cloud, and libtorrent's outbound bind address". #11 says "The walking skeleton checks that `<ViewTransition>` works with TanStack Router's navigation." No issue has a "foundations" or library-adoption step. Library choices were settled in #11 (closed) |
| **94** | **Design system and DESIGN.md** | No blockers. It owns the UI **component layer** that #11 left open ("The component layer stays with [Design system and DESIGN.md]"). It distils prototypes #24, #25, #61, #63, #64, #65 and #69, and says "no prototype code lands". It uses "the impeccable skill for the DESIGN.md format". Its body says #62 is still open, but #62 is now closed |
| **101** | **How Media Servers handle a file moving to a new path** | No blockers. "The Move spec and the Media Server fakes wait on this result." Adds a path-change scenario to the #71 conformance suite |
| **106** | **Tracker detail page** | No blockers. A UI prototype made possible by #87. It starts from #64's "By tracker" card |
| **107** | **Manual search sheet under Editions and Targets** | No blockers. A UI prototype. Content settled by #87 Q15 |

None of the open issues is about library choices. The only explicit library deferrals in the map are TanStack DB, Vite+ and the UI component layer (→ #94). The map leaves these unnamed, so they are still open for a library survey:

- a WebAuthn library;
- UPnP, NAT-PMP and PCP client libraries;
- a SOCKS5 client;
- the zstd and age Go packages (age is named as a format only);
- a BLAKE3 implementation;
- a minisign verifier;
- a YAML parser for Cardigann definitions and Scenarios;
- a cron or scheduler;
- an HTML templating or email library for the table-layout emails.
