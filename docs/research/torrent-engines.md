# Torrent engines at private-tracker grade

Research for [ticket #10](https://github.com/jatassi/SlipStream/issues/10) on the v2 map. It feeds charting decision Q13 ("the torrent client is private-tracker grade, Usenet stays external") from [issue #2](https://github.com/jatassi/SlipStream/issues/2). It lays out facts and options. It does not pick an engine.

Every source link is pinned to the commit read on 2026-10-04. Claims marked **inferred** are reasoning from the cited code, not something a source states.

## Answer in brief

- Only libtorrent-rasterbar documents seeding thousands of torrents at once and ships ratio and seed-time limits. Its limits only deprioritize a torrent in the queue. qBittorrent implements the stop or remove action itself on top. ([LT-tuning], [LT-limits], [QB-share])
- anacrolix/torrent and rain are mature Go libraries with no seeding-goal policy. The host must implement ratio and time goals. rain runs at put.io and keeps a separate identity for private torrents. ([AN-seed], [RN-readme-top], [RN-private-id])
- librqbit is not ready for private trackers today. Its own code says private trackers are "not fully implemented". Its HTTP announce loop never sends `completed` or `stopped`. ([RQ-private], [RQ-http-loop])
- Trackers gate clients by a **prefix match on the 20-byte `peer_id`**. Gazelle/Ocelot sites use an allow list. UNIT3D sites use a deny list plus User-Agent and header checks. So a new peer ID prefix fails every allow-list tracker until that site's staff add it (inferred from the mechanism). ([OC-whitelist], [GZ-seed], [U3-checkClient])
- A fake tracker and swarm can run fully offline on loopback. anacrolix ships an embeddable tracker server. chihaya has a client allow/deny middleware. libtorrent has a discrete-event network simulator, but only for code compiled against it. ([AN-trackerserver], [CH-clientapproval], [LS-readme])

## 1. What "private-tracker grade" means in protocol terms

### From the specifications

- **BEP 27 (private torrents).** A client with `private=1` "MUST ONLY announce itself to the private tracker" and must only connect to peers that tracker returns. DHT, PEX and LSD are out. A peer "MUST use only one tracker at a time and only switch between trackers when the current tracker fails". On a switch it must drop all peers. ([BEP-27])
- **BEP 3 (announce).** `uploaded` and `downloaded` are totals "so far". `event` is `started` when a download begins, `completed` when it finishes, and `stopped` when the client ceases. The response carries `interval` and may carry `failure reason`. ([BEP-3])
- **BEP 12 (multitracker).** Tiers are processed in order. URLs inside a tier are shuffled once, tried in order, and a working tracker moves to the front. ([BEP-12])
- **BEP 20 (peer ID conventions).** Documents the Azureus style `-XXnnnn-` prefix, for example `-qB` for qBittorrent and `-TR` for Transmission. It is a record of practice. It states no registry and no mandate. ([BEP-20])

### From tracker source code

Tracker software shows the rules a client meets in practice. Two open-source private-tracker codebases were read. Ocelot is the tracker that pairs with Gazelle sites. UNIT3D is a full site with a built-in announce endpoint.

| Rule | Ocelot (Gazelle) | UNIT3D |
|---|---|---|
| Client gate | Allow list of `peer_id` prefixes. Empty list disables the gate. ([OC-whitelist], [OC-loadwl]) | Deny list of `peer_id` prefixes. ([U3-checkClient]) |
| User-Agent | Stored, truncated to 51 chars, not enforced. ([OC-ua]) | Required. Over 64 chars rejected. Browser words (`Mozilla`, `Chrome`, `Bot` and others) rejected. Requests carrying `accept-language`, `referer`, `accept-charset` or `want-digest` rejected. ([U3-checkClient]) |
| Announce interval | `interval` 1800 s plus up to 240 s jitter. `min interval` 1800 s. ([OC-config], [OC-interval]) | 1800 to 3600 s. An announce inside 85 to 95 percent of the minimum is rejected. A duplicate announce for the same event inside 30 s is rejected; the code comment names qBittorrent as the cause. ([U3-interval], [U3-minInterval]) |
| `completed` event | A snatch is recorded only on `event=completed` with `left=0`. ([OC-completed]) | `completed` from a peer the tracker has no record of is rejected. ([U3-checkPeer]) |
| Port | Trusted as sent. ([OC-port]) | Ports below 1024 rejected except on `stopped`. 8080, 8081, 1214, 3389, 4662, 6346, 6347 and 6699 rejected. ([U3-ports]) |
| Peers per user per torrent | Not checked in the announce path read. | Capped by `announce.rate_limit`. ([U3-maxConn]) |
| Seeding obligation | Site policy, outside the tracker. | Hit-and-run defaults to 604800 s (7 days) of seed time, 3 day grace, 3 warnings. ([U3-hitrun]) |

**Inferred.** Private-tracker grade therefore means at least these engine behaviors. BEP 27 isolation. One announce per tracker per interval that honors `min interval`. `started`, `completed` once, and `stopped` on pause, removal and shutdown. Accurate cumulative `uploaded` and `downloaded`. A listen port above 1024 that is not on common deny lists. A per-tracker identity that the site accepts.

## 2. How trackers whitelist clients, and what that means for a new client

### The mechanism

- **Ocelot** loads `SELECT peer_id FROM xbt_client_whitelist` at start and on `SIGUSR1`. It rejects any announce whose `peer_id` does not start with one of the prefixes, with "Your client is not on the whitelist". Staff can add, remove and edit entries at runtime through the site API. ([OC-loadwl], [OC-whitelist], [OC-wl-api], [OC-readme])
- The whitelist table holds a `peer_id` prefix of up to 20 chars and a display `vstring`. ([OC-schema])
- **Gazelle's own seed whitelist** shows the granularity the software ships with. It lists `-qB46` (qBittorrent 4.6.x), `-qB50`, `-DE2` (Deluge 2.x), `-lt0F` (rTorrent's libtorrent 0.15.x), `-BI3700-` (BiglyBT 3.7) and one row per exact Transmission release, `-TR4010-` through `-TR4060-`. It also lists `-LT`, labelled "Libtorrent (Rasterbar)". ([GZ-seed])
- **UNIT3D** inverts the default. Any client passes unless staff add its `peer_id_prefix` to `blacklist_clients`. It adds the User-Agent and header heuristics in the table above. ([U3-checkClient], [U3-blacklist-migration])
- **chihaya** (a general tracker) offers the same choice as config. Its `clientapproval` middleware takes a `whitelist` or a `blacklist` of client IDs, never both. ([CH-clientapproval])

The live allow lists of real private sites sit behind logins. This research could not read them. The Gazelle seed list is the closest primary source.

### What the engines send by default

| Engine | Default peer ID prefix | Default HTTP User-Agent | Configurable |
|---|---|---|---|
| libtorrent-rasterbar | `-LT2120-` unless the host sets `peer_fingerprint` | `libtorrent/<version>` unless the host sets `user_agent`; the docs recommend `client/version libtorrent/version` | Yes, both. ([LT-defaults], [LT-ua], [LT-fingerprint]) |
| qBittorrent (on libtorrent) | `-qB` + version | `qBittorrent/<version>` | Hard-coded. ([QB-id]) |
| anacrolix/torrent | `-GT0003-` | `anacrolix-torrent/<module version>` | Yes, `Bep20` and `HTTPUserAgent`. ([AN-version], [AN-config-id]) |
| rain | `-RN<version>-` | `Rain/<version>` | Yes, separately for private torrents. ([RN-private-id]) |
| librqbit | Azureus style with code `rQ` | `client_name_and_version` option | Yes, `peer_id` and name. ([RQ-peerid], [RQ-ua]) |
| Transmission | `-TR` plus version digits | `Transmission/<version>` | Build time only. ([TR-cmake], [TR-ua]) |

anacrolix also ships an `AnonymousMode`. It swaps in a libtorrent-style `-LT20C0-` prefix and a `curl/7.81.0` User-Agent, with code comments pointing at libtorrent as the source of those values. ([AN-anon], [AN-version])

### Implications for shipping a new client (inferred unless cited)

- A new honest prefix such as `-SS0100-` fails every allow-list tracker until that site's staff add it. Gazelle-style lists key on version prefixes, and some on exact releases ([GZ-seed]). Each SlipStream release that changes the version digits could need a fresh entry on those sites.
- Deny-list trackers (UNIT3D family) accept a new prefix by default. They still reject on User-Agent shape, extra headers, min-interval breaches and duplicate announces ([U3-checkClient], [U3-minInterval]).
- An embedded libtorrent can present the library's own `-LT` fingerprint. Gazelle's seed list allows `-LT` ([GZ-seed]). Whether a given site keeps that row, and whether its rules treat a libtorrent-based app as acceptable, is site policy this research cannot read.
- Sending another client's prefix, such as `-qB46`, while running different code is spoofing. Trackers see the `peer_id` and the User-Agent side by side ([OC-ua], [U3-checkClient]), so a mismatch is visible to staff. Site rules on spoofing are behind logins and were not read.
- A managed sidecar of an already whitelisted client (for example qBittorrent-nox on libtorrent) keeps an honest, accepted identity. It moves the engine out of process and puts its license (GPLv2 or v3, [QB-copying]) on a separate binary rather than linked code.

## 3. Engine survey

Pinned commits are listed under Sources. "Seeding goals" means ratio and time targets with a stop or remove action.

| | libtorrent-rasterbar 2.1.2 | anacrolix/torrent | rain | librqbit | libtransmission |
|---|---|---|---|---|---|
| Language | C++ | Go | Go | Rust | C++ |
| License | BSD-3-Clause ([LT-license]) | MPL-2.0 ([AN-license]) | MIT ([RN-license]) | Apache-2.0 ([RQ-license]) | GPLv2 or GPLv3 ([TR-copying]) |
| Scale evidence | Docs cover "servers seeding thousands of torrents" and name the knobs ([LT-tuning], [LT-tuning-limits]) | "used 24/7 in production by downstream services since late 2014" ([AN-readme]) | put.io since 2019, "thousands of torrents daily", "optimized for hundreds of concurrent torrents" ([RN-readme-top], [RN-readme]) | None found in README | Not assessed |
| BEP 27 isolation | DHT, LSD and PEX skipped for private torrents ([LT-priv-dht], [LT-priv-lsd], [LT-priv-pex]) | DHT, LSD and PEX skipped ([AN-priv], [AN-priv-lsd], [AN-priv-pex]) | DHT skipped for private torrents ([RN-priv-dht]) | DHT and LSD skipped ([RQ-private]) | Not assessed |
| Multi-tracker | BEP 12 by default; `announce_to_all_tiers` and `announce_to_all_trackers` opt into uTorrent behavior ([LT-tiers]) | Starts a scraper for every URL in every tier ([AN-alltiers]) | BEP 12 tiers ([RN-tiers]) | Private torrents use only the first tracker ([RQ-private]) | Not assessed |
| `min interval` | `min_announce_interval` floor setting ([LT-mininterval]) | HTTP response struct has no `min interval` field; uses `interval` ([AN-httpresp]) | `tracker-min-announce-interval` setting ([RN-config]) | Announces at `min interval` when present, else `interval` ([RQ-http-loop]) | Not assessed |
| Events | `started`, `completed` once, `stopped`, plus `paused` ([LT-events], [LT-event-strings]) | `started`, `completed` once, `stopped` ([AN-events]) | `stopped` sent on removal and on close, tested ([RN-stopped-test]) | Over HTTP, `started` once and then no event ([RQ-http-loop]). Over UDP, `completed` on every announce once complete ([RQ-udp-event]) | Not assessed |
| Seeding goals | `share_ratio_limit`, `seed_time_ratio_limit`, `seed_time_limit`. Reaching one only drops queue priority ([LT-limits], [LT-seedtime]) | None. `Seed` bool only ([AN-seed]) | None found. Persists `BytesUploaded` and `SeededFor` per torrent ([RN-stats]) | None found | Ratio limit and idle limit per torrent ([TR-limits]) |
| Persisted counters | `total_uploaded`, `seeding_time`, `finished_time`, `active_time` in resume data ([LT-atp], [LT-status]) | Session stats only (`Torrent.Stats`) ([AN-stats]). **Inferred.** The host must persist totals | Bolt resume store ([RN-stats]) | Not assessed | Not assessed |
| Embedding from Go | cgo over C++, or the small C binding in `bindings/c` (session, add/remove, alerts, settings, status) ([LT-cbind]) | Native | Native | FFI or sidecar | GPL linking |

### Notes per engine

**libtorrent-rasterbar.** The engine behind qBittorrent and Deluge. qBittorrent's current master requires libtorrent 1.2.19 or 2.0.10 at minimum ([QB-cmake]). To seed thousands of torrents the docs say to raise `active_limit` and `active_seeds`. They also say to use `get_torrent_status()` with `post_torrent_updates()` for large sessions ([LT-tuning-limits]). The ratio and time limits make a torrent "done" so it "will leave room for other torrents". Done torrents "will still be allowed to be seeded" ([LT-limits]). qBittorrent adds the policy layer itself. It checks ratio, seeding time and inactive seeding time per torrent, in "match any" or "match all" mode, and then stops, removes, removes with content, or enables super seeding ([QB-share]). The official bindings are Python and a thin C binding with a 2020 to 2021 copyright ([LT-cbind], [LT-bindings]).

**anacrolix/torrent.** Library-first Go package with DHT, PEX, uTP, v2 and pluggable storage ([AN-readme]). Identity is fully configurable ([AN-config-id]). It honors BEP 27 isolation. It does not follow BEP 27's "one tracker at a time" for multi-URL private torrents, because it announces to every URL ([AN-alltiers]). **Inferred.** Most private torrents carry one announce URL, so this matters only for multi-URL private torrents. The announce dispatcher computes the next announce from the last response's `interval` ([AN-events]). Seeding goals need a host-side policy.

**rain.** Server-first Go client and library from put.io. It allocates **one listen port per torrent**, from 20000 to 30000 by default ([RN-readme], [RN-ports]). The README cites "accurate ratio tracking per account" on private trackers as the reason. **Inferred.** A Node seeding thousands of torrents needs that many ports open and forwarded, which the Q10 forwarder would have to carry. rain applies a separate `peer_id` prefix, handshake version and User-Agent to private torrents only ([RN-private-id], [RN-private-ua]).

**librqbit.** Fast Rust client with an HTTP API. Its session code truncates a private torrent's tracker list to one with the warning "private trackers are not fully implemented, so using only the first tracker" ([RQ-private]). The HTTP announce loop sends `started` once and then no event, so `completed` and `stopped` never reach an HTTP tracker ([RQ-http-loop]). **Inferred.** On Ocelot this means no snatch is recorded ([OC-completed]).

**libtransmission.** Has per-torrent ratio and idle limits but no seed-time limit in its public header ([TR-limits]). It is licensed GPLv2 or GPLv3 ([TR-copying]). **Inferred.** Linking it would put SlipStream under GPL terms.

**Not researched.** rakshasa's libtorrent (rTorrent's engine) and Java or JavaScript engines were out of scope for this pass.

## 4. Where seeding-goal policy lives

**Inferred** from section 3. No surveyed Go or Rust engine owns ratio or time goals. libtorrent's built-in limits only reorder the queue. In every option the Node's downloading Role decides when a torrent has met its goal, and acts by stopping or removing it. qBittorrent's `processTorrentShareLimits` is a working model of that policy ([QB-share]). Tracker obligations such as UNIT3D's 7-day hit-and-run window ([U3-hitrun]) are per site, so the goal is naturally a per-tracker rule rather than a global setting.

A host-side policy can take an injected clock. That lets the offline suite test a 7-day seed goal in seconds. libtorrent's internal `seed_time_limit` reads the session's clock, which only libsimulator can fake (section 5).

## 5. Driving an engine offline with a fake swarm and tracker

Charting decision Q17 requires the end-to-end suite to run with network egress blocked. Every piece below runs on loopback.

| Piece | Option | Notes |
|---|---|---|
| Fake tracker, in process (Go) | anacrolix `tracker/server` and `tracker/http/server` | The `AnnounceTracker` interface (`TrackAnnounce`, `Scrape`, `GetPeers`) lets a test own every announce and response ([AN-trackerserver]). |
| Recording tracker | `httptest` handler that logs each query | rain's own tests do exactly this to assert `started` then `stopped` with `uploaded` and `downloaded` present ([RN-stopped-test]). |
| Fake tracker, standalone | chihaya | HTTP and UDP, memory or Redis storage, client allow or deny middleware ([CH-readme], [CH-clientapproval]). **Inferred.** It has no per-user passkey or ratio ledger, so ratio checks need a custom handler. |
| Production-shaped private tracker | Ocelot with MySQL | Ships a Dockerfile and `gazelle.sql` ([OC-readme]). Enforces the real allow list and `completed` semantics. Its license is a custom What.CD EULA, not an open-source license ([OC-license]). |
| Fake swarm | Two or more engine sessions on loopback seeding generated content | rqbit's e2e tests and anacrolix's `test/` package do this ([RQ-e2e], [AN-test]). |
| Simulated network and time | libtorrent `simulation/` on libsimulator | A discrete-event simulator implementing the `boost.asio` API with simulated time, bandwidth and latency. Code must be compiled against it ([LS-readme], [LT-gitmodules], [LT-sim]). **Inferred.** It suits testing libtorrent itself, not a host app that links a normal libtorrent build. |

**Inferred.** A compliance suite can assert the section 1 rules directly against a recording tracker. It can check one `started`, one `completed`, a `stopped` on pause and removal, intervals at or above `min interval`, the configured `peer_id` prefix and User-Agent, no DHT or LSD traffic for private torrents, and monotonic `uploaded`. Real whitelist acceptance cannot be tested offline. Only a live site decides that.

## 6. Options for the decision

These are the shapes the facts allow. Picking one belongs to a later decision ticket.

1. **Embed libtorrent-rasterbar.** Proven at thousands of torrents. Built-in BEP 27, BEP 12 and a `min interval` floor. The host still writes the goal policy. The cost is C++ in the build, reached by cgo or the thin C binding, or a small sidecar in C++ or Python. The identity stays open, either an own prefix or the library's `-LT` default.
2. **Embed a Go engine (anacrolix or rain).** No cross-language build if the stack stays Go, as v1 is ([v1-backend]). The host writes the goal policy. For anacrolix it likely also patches multi-tracker and `min interval` handling (inferred). rain's port-per-torrent design shapes the forwarder. Scale to thousands needs a prototype to measure.
3. **Supervise a whitelisted external engine as a sidecar.** For example qBittorrent-nox under SlipStream's control on each downloading Node. It has an accepted identity today and a mature goal policy. The cost is a second process and API to manage, and a GPL binary to ship in packages and Docker images (Q24).
4. **librqbit.** Not viable for private trackers without upstream work on tiers and announce events ([RQ-private], [RQ-http-loop]).

Questions a prototype can settle by running:

- Memory and CPU per thousand idle seeding torrents for options 1 to 3.
- Whether anacrolix holds a 1800 s interval across a thousand torrents without bursts.

One question no prototype settles. How long a whitelist request takes on the chef's target sites, which only the chef can ask.

## Sources

Specifications are BEP 3, 12, 20 and 27 on bittorrent.org. Repositories were read at these commits.

- libtorrent-rasterbar `arvidn/libtorrent` at `c1698d585415d00cc5f8feb935cadcbc63e6361b` (branch RC_2_1, version 2.1.2.0)
- anacrolix/torrent at `d913b30f520e10e85a94c7dc4b7b3e828b22c4b2`
- rain `cenkalti/rain` at `33efae40f7a6822087ff36309cb7acf20b9c73ce`
- rqbit `ikatson/rqbit` at `176484e2e3bfcfea1ec6c66c9f20ab755377f63b`
- Transmission at `48835c6660a7a3730b5a122bb7b88909997addbe`
- qBittorrent at `9cc9aeade3b87b659a4ec22b9033fe3878207a06`
- Ocelot `OPSnet/Ocelot` at `d254fcad6e11a0d7cb60c781403ab8b0daa3526d`
- Gazelle `OPSnet/Gazelle` at `a80542f2cd6072ad5833c785f6db50813b52f04d`
- UNIT3D `HDInnovations/UNIT3D` at `8b88f4c8182eb3d3912ffef425c3224dcfd596f4`
- chihaya at `878b42edf8e8702dee3c7f21a5eeb3a08cbdd4f2`

[BEP-3]: https://www.bittorrent.org/beps/bep_0003.html
[BEP-12]: https://www.bittorrent.org/beps/bep_0012.html
[BEP-20]: https://www.bittorrent.org/beps/bep_0020.html
[BEP-27]: https://www.bittorrent.org/beps/bep_0027.html

[LT-tuning]: https://github.com/arvidn/libtorrent/blob/c1698d585415d00cc5f8feb935cadcbc63e6361b/docs/tuning.rst#L10-L16
[LT-tuning-limits]: https://github.com/arvidn/libtorrent/blob/c1698d585415d00cc5f8feb935cadcbc63e6361b/docs/tuning.rst#L254-L275
[LT-defaults]: https://github.com/arvidn/libtorrent/blob/c1698d585415d00cc5f8feb935cadcbc63e6361b/src/settings_pack.cpp#L117-L127
[LT-limits]: https://github.com/arvidn/libtorrent/blob/c1698d585415d00cc5f8feb935cadcbc63e6361b/include/libtorrent/settings_pack.hpp#L1884-L1892
[LT-seedtime]: https://github.com/arvidn/libtorrent/blob/c1698d585415d00cc5f8feb935cadcbc63e6361b/include/libtorrent/settings_pack.hpp#L1456-L1459
[LT-ua]: https://github.com/arvidn/libtorrent/blob/c1698d585415d00cc5f8feb935cadcbc63e6361b/include/libtorrent/settings_pack.hpp#L199-L204
[LT-fingerprint]: https://github.com/arvidn/libtorrent/blob/c1698d585415d00cc5f8feb935cadcbc63e6361b/include/libtorrent/settings_pack.hpp#L370-L376
[LT-tiers]: https://github.com/arvidn/libtorrent/blob/c1698d585415d00cc5f8feb935cadcbc63e6361b/include/libtorrent/settings_pack.hpp#L518-L529
[LT-mininterval]: https://github.com/arvidn/libtorrent/blob/c1698d585415d00cc5f8feb935cadcbc63e6361b/include/libtorrent/settings_pack.hpp#L1489-L1493
[LT-priv-dht]: https://github.com/arvidn/libtorrent/blob/c1698d585415d00cc5f8feb935cadcbc63e6361b/src/torrent.cpp#L766-L767
[LT-priv-lsd]: https://github.com/arvidn/libtorrent/blob/c1698d585415d00cc5f8feb935cadcbc63e6361b/src/torrent.cpp#L2932-L2933
[LT-priv-pex]: https://github.com/arvidn/libtorrent/blob/c1698d585415d00cc5f8feb935cadcbc63e6361b/src/ut_pex.cpp#L605
[LT-events]: https://github.com/arvidn/libtorrent/blob/c1698d585415d00cc5f8feb935cadcbc63e6361b/src/torrent.cpp#L3510-L3520
[LT-event-strings]: https://github.com/arvidn/libtorrent/blob/c1698d585415d00cc5f8feb935cadcbc63e6361b/src/http_tracker_connection.cpp#L115
[LT-atp]: https://github.com/arvidn/libtorrent/blob/c1698d585415d00cc5f8feb935cadcbc63e6361b/include/libtorrent/add_torrent_params.hpp#L252-L259
[LT-status]: https://github.com/arvidn/libtorrent/blob/c1698d585415d00cc5f8feb935cadcbc63e6361b/include/libtorrent/torrent_status.hpp#L571-L573
[LT-cbind]: https://github.com/arvidn/libtorrent/blob/c1698d585415d00cc5f8feb935cadcbc63e6361b/bindings/c/include/libtorrent.h#L219-L287
[LT-bindings]: https://github.com/arvidn/libtorrent/tree/c1698d585415d00cc5f8feb935cadcbc63e6361b/bindings
[LT-license]: https://github.com/arvidn/libtorrent/blob/c1698d585415d00cc5f8feb935cadcbc63e6361b/LICENSE
[LT-gitmodules]: https://github.com/arvidn/libtorrent/blob/c1698d585415d00cc5f8feb935cadcbc63e6361b/.gitmodules#L1-L3
[LT-sim]: https://github.com/arvidn/libtorrent/blob/c1698d585415d00cc5f8feb935cadcbc63e6361b/simulation/setup_swarm.hpp#L32-L73
[LS-readme]: https://github.com/arvidn/libsimulator

[QB-id]: https://github.com/qbittorrent/qBittorrent/blob/9cc9aeade3b87b659a4ec22b9033fe3878207a06/src/base/bittorrent/sessionimpl.cpp#L133-L134
[QB-share]: https://github.com/qbittorrent/qBittorrent/blob/9cc9aeade3b87b659a4ec22b9033fe3878207a06/src/base/bittorrent/sessionimpl.cpp#L2494-L2580
[QB-cmake]: https://github.com/qbittorrent/qBittorrent/blob/9cc9aeade3b87b659a4ec22b9033fe3878207a06/CMakeLists.txt#L13-L14
[QB-copying]: https://github.com/qbittorrent/qBittorrent/blob/9cc9aeade3b87b659a4ec22b9033fe3878207a06/COPYING

[AN-readme]: https://github.com/anacrolix/torrent/blob/d913b30f520e10e85a94c7dc4b7b3e828b22c4b2/README.md#L5-L7
[AN-license]: https://github.com/anacrolix/torrent/blob/d913b30f520e10e85a94c7dc4b7b3e828b22c4b2/LICENSE
[AN-seed]: https://github.com/anacrolix/torrent/blob/d913b30f520e10e85a94c7dc4b7b3e828b22c4b2/config.go#L118-L123
[AN-config-id]: https://github.com/anacrolix/torrent/blob/d913b30f520e10e85a94c7dc4b7b3e828b22c4b2/config.go#L180-L217
[AN-version]: https://github.com/anacrolix/torrent/blob/d913b30f520e10e85a94c7dc4b7b3e828b22c4b2/version/version.go#L11-L79
[AN-anon]: https://github.com/anacrolix/torrent/blob/d913b30f520e10e85a94c7dc4b7b3e828b22c4b2/client.go#L326-L331
[AN-priv]: https://github.com/anacrolix/torrent/blob/d913b30f520e10e85a94c7dc4b7b3e828b22c4b2/torrent.go#L2401-L2403
[AN-priv-lsd]: https://github.com/anacrolix/torrent/blob/d913b30f520e10e85a94c7dc4b7b3e828b22c4b2/client.go#L1623-L1624
[AN-priv-pex]: https://github.com/anacrolix/torrent/blob/d913b30f520e10e85a94c7dc4b7b3e828b22c4b2/pexconn.go#L45-L46
[AN-alltiers]: https://github.com/anacrolix/torrent/blob/d913b30f520e10e85a94c7dc4b7b3e828b22c4b2/torrent.go#L2251-L2260
[AN-httpresp]: https://github.com/anacrolix/torrent/blob/d913b30f520e10e85a94c7dc4b7b3e828b22c4b2/tracker/http/protocol.go#L11-L20
[AN-events]: https://github.com/anacrolix/torrent/blob/d913b30f520e10e85a94c7dc4b7b3e828b22c4b2/client-tracker-announcer.go#L824-L870
[AN-stats]: https://github.com/anacrolix/torrent/blob/d913b30f520e10e85a94c7dc4b7b3e828b22c4b2/torrent.go#L2455-L2476
[AN-trackerserver]: https://github.com/anacrolix/torrent/blob/d913b30f520e10e85a94c7dc4b7b3e828b22c4b2/tracker/server/server.go#L38-L68
[AN-test]: https://github.com/anacrolix/torrent/tree/d913b30f520e10e85a94c7dc4b7b3e828b22c4b2/test

[RN-readme-top]: https://github.com/cenkalti/rain/blob/33efae40f7a6822087ff36309cb7acf20b9c73ce/README.md#L8
[RN-readme]: https://github.com/cenkalti/rain/blob/33efae40f7a6822087ff36309cb7acf20b9c73ce/README.md#L160-L168
[RN-license]: https://github.com/cenkalti/rain/blob/33efae40f7a6822087ff36309cb7acf20b9c73ce/LICENSE
[RN-private-id]: https://github.com/cenkalti/rain/blob/33efae40f7a6822087ff36309cb7acf20b9c73ce/torrent/config.go#L44-L49
[RN-private-ua]: https://github.com/cenkalti/rain/blob/33efae40f7a6822087ff36309cb7acf20b9c73ce/torrent/session.go#L265-L270
[RN-config]: https://github.com/cenkalti/rain/blob/33efae40f7a6822087ff36309cb7acf20b9c73ce/torrent/config.go#L114-L123
[RN-ports]: https://github.com/cenkalti/rain/blob/33efae40f7a6822087ff36309cb7acf20b9c73ce/torrent/config.go#L219-L220
[RN-tiers]: https://github.com/cenkalti/rain/blob/33efae40f7a6822087ff36309cb7acf20b9c73ce/torrent/session.go#L246-L263
[RN-priv-dht]: https://github.com/cenkalti/rain/blob/33efae40f7a6822087ff36309cb7acf20b9c73ce/torrent/torrent_start.go#L90
[RN-stats]: https://github.com/cenkalti/rain/blob/33efae40f7a6822087ff36309cb7acf20b9c73ce/torrent/session_stats.go#L138-L140
[RN-stopped-test]: https://github.com/cenkalti/rain/blob/33efae40f7a6822087ff36309cb7acf20b9c73ce/torrent/session_remove_test.go#L18-L78

[RQ-license]: https://github.com/ikatson/rqbit/blob/176484e2e3bfcfea1ec6c66c9f20ab755377f63b/LICENSE
[RQ-private]: https://github.com/ikatson/rqbit/blob/176484e2e3bfcfea1ec6c66c9f20ab755377f63b/crates/librqbit/src/session.rs#L1551-L1579
[RQ-peerid]: https://github.com/ikatson/rqbit/blob/176484e2e3bfcfea1ec6c66c9f20ab755377f63b/crates/librqbit/src/session.rs#L580-L582
[RQ-ua]: https://github.com/ikatson/rqbit/blob/176484e2e3bfcfea1ec6c66c9f20ab755377f63b/crates/librqbit/src/session.rs#L728-L731
[RQ-http-loop]: https://github.com/ikatson/rqbit/blob/176484e2e3bfcfea1ec6c66c9f20ab755377f63b/crates/tracker_comms/src/tracker_comms.rs#L238-L318
[RQ-udp-event]: https://github.com/ikatson/rqbit/blob/176484e2e3bfcfea1ec6c66c9f20ab755377f63b/crates/tracker_comms/src/tracker_comms.rs#L406-L417
[RQ-e2e]: https://github.com/ikatson/rqbit/blob/176484e2e3bfcfea1ec6c66c9f20ab755377f63b/crates/librqbit/src/tests/e2e.rs

[TR-limits]: https://github.com/transmission/transmission/blob/48835c6660a7a3730b5a122bb7b88909997addbe/libtransmission/transmission.h#L765-L775
[TR-copying]: https://github.com/transmission/transmission/blob/48835c6660a7a3730b5a122bb7b88909997addbe/COPYING
[TR-cmake]: https://github.com/transmission/transmission/blob/48835c6660a7a3730b5a122bb7b88909997addbe/CMakeLists.txt#L158-L175
[TR-ua]: https://github.com/transmission/transmission/blob/48835c6660a7a3730b5a122bb7b88909997addbe/libtransmission/session.cc#L324-L327

[OC-whitelist]: https://github.com/OPSnet/Ocelot/blob/d254fcad6e11a0d7cb60c781403ab8b0daa3526d/src/worker.cpp#L459-L472
[OC-loadwl]: https://github.com/OPSnet/Ocelot/blob/d254fcad6e11a0d7cb60c781403ab8b0daa3526d/src/db.cpp#L242-L262
[OC-wl-api]: https://github.com/OPSnet/Ocelot/blob/d254fcad6e11a0d7cb60c781403ab8b0daa3526d/src/worker.cpp#L1085-L1111
[OC-schema]: https://github.com/OPSnet/Ocelot/blob/d254fcad6e11a0d7cb60c781403ab8b0daa3526d/gazelle.sql#L132-L138
[OC-ua]: https://github.com/OPSnet/Ocelot/blob/d254fcad6e11a0d7cb60c781403ab8b0daa3526d/src/worker.cpp#L423-L436
[OC-port]: https://github.com/OPSnet/Ocelot/blob/d254fcad6e11a0d7cb60c781403ab8b0daa3526d/src/worker.cpp#L394-L407
[OC-completed]: https://github.com/OPSnet/Ocelot/blob/d254fcad6e11a0d7cb60c781403ab8b0daa3526d/src/worker.cpp#L505-L507
[OC-interval]: https://github.com/OPSnet/Ocelot/blob/d254fcad6e11a0d7cb60c781403ab8b0daa3526d/src/worker.cpp#L815-L822
[OC-config]: https://github.com/OPSnet/Ocelot/blob/d254fcad6e11a0d7cb60c781403ab8b0daa3526d/src/config.cpp#L69-L72
[OC-readme]: https://github.com/OPSnet/Ocelot/blob/d254fcad6e11a0d7cb60c781403ab8b0daa3526d/README.md
[OC-license]: https://github.com/OPSnet/Ocelot/blob/d254fcad6e11a0d7cb60c781403ab8b0daa3526d/LICENSE

[GZ-seed]: https://github.com/OPSnet/Gazelle/blob/a80542f2cd6072ad5833c785f6db50813b52f04d/misc/my-migrations/20250503000000_remove_phinx_seeds.php#L11-L45

[U3-checkClient]: https://github.com/HDInnovations/UNIT3D/blob/8b88f4c8182eb3d3912ffef425c3224dcfd596f4/app/Http/Controllers/AnnounceController.php#L144-L195
[U3-interval]: https://github.com/HDInnovations/UNIT3D/blob/8b88f4c8182eb3d3912ffef425c3224dcfd596f4/app/Http/Controllers/AnnounceController.php#L46-L48
[U3-ports]: https://github.com/HDInnovations/UNIT3D/blob/8b88f4c8182eb3d3912ffef425c3224dcfd596f4/app/Http/Controllers/AnnounceController.php#L50-L66
[U3-minInterval]: https://github.com/HDInnovations/UNIT3D/blob/8b88f4c8182eb3d3912ffef425c3224dcfd596f4/app/Http/Controllers/AnnounceController.php#L463-L500
[U3-checkPeer]: https://github.com/HDInnovations/UNIT3D/blob/8b88f4c8182eb3d3912ffef425c3224dcfd596f4/app/Http/Controllers/AnnounceController.php#L443-L454
[U3-maxConn]: https://github.com/HDInnovations/UNIT3D/blob/8b88f4c8182eb3d3912ffef425c3224dcfd596f4/app/Http/Controllers/AnnounceController.php#L508-L520
[U3-hitrun]: https://github.com/HDInnovations/UNIT3D/blob/8b88f4c8182eb3d3912ffef425c3224dcfd596f4/config/hitrun.php#L26-L60
[U3-blacklist-migration]: https://github.com/HDInnovations/UNIT3D/blob/8b88f4c8182eb3d3912ffef425c3224dcfd596f4/database/migrations/2023_12_30_092415_add_peer_id_prefix_to_blacklist_client.php

[CH-clientapproval]: https://github.com/chihaya/chihaya/blob/878b42edf8e8702dee3c7f21a5eeb3a08cbdd4f2/middleware/clientapproval/clientapproval.go#L43-L96
[CH-readme]: https://github.com/chihaya/chihaya/blob/878b42edf8e8702dee3c7f21a5eeb3a08cbdd4f2/README.md

[v1-backend]: https://github.com/jatassi/SlipStream/blob/research/v1-inventory/docs/research/v1-inventory/backend.md
