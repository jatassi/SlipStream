# Torrent engine bakeoff: findings

Prototype for [#33](https://github.com/jatassi/SlipStream/issues/33), following the research in [#10](https://github.com/jatassi/SlipStream/issues/10) and [#8](https://github.com/jatassi/SlipStream/issues/8). Everything under this directory is throwaway measuring code. The fake tracker and swarm double as the first offline-mode fake.

**Decision (Jackson, 2026-10-11): SlipStream v2 embeds libtorrent-rasterbar.**

## Setup

- One box: 4 vCPU, 16 GB, Linux 6.18, Ubuntu 24.04.
- Engines:
  - libtorrent-rasterbar 2.0.10, the distro package, reached through a 281-line C shim (`lt-shim/`) from both Go/cgo and Rust.
  - anacrolix/torrent v1.61.0.
  - rain v1.13.0.
  - Go 1.24.7 and Rust 1.97.
- Fixtures (`fixtures/`): private single-file torrents of 4 GiB each, 1024 pieces of 4 MiB. The data files are sparse, so nothing touches the disk.
- Every engine adds every torrent as a complete seed without hashing the data:
  - libtorrent uses `seed_mode`.
  - anacrolix gets a piece-completion store that reports every piece complete.
  - rain gets a resume DB written by the harness.
- Fake tracker (`tracker/`): HTTP on loopback, with an in-memory swarm. Its rules copy UNIT3D's `checkMinInterval` (`AnnounceController.php` at `8b88f4c`):
  - An announce is rejected if it repeats the same event within 30 s.
  - An announce is rejected if it comes sooner than 0.9 x `min interval` after the announce that opened the window. `completed` and `stopped` are exempt.
- Each engine disables DHT, LSD, UPnP and NAT-PMP, lifts every limit on active torrents, and applies seedbox tuning. Each harness's `NOTES.md` lists the exact settings.

## 1. Idle seeding cost

`run/seed.sh` runs one engine at a time on a quiet box. Each run adds N torrents, waits 60 s, then samples `/proc` every 2 s for 180 s. The tracker interval is 1800 s, so the window holds no announces. Every run was checked to have reached the tracker: N distinct `started` infohashes. Each cell is the median of 3 runs, with the range in brackets. lt-rust ran once.

| Engine | Anonymous RSS at 1000 | Anonymous RSS at 3000 | Per 1000 torrents | CPU at 1000 (% of one core) | CPU at 3000 | Time to add all | Threads / fds at 3000 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| libtorrent via Go | 30.2 MiB [30.1–30.2] | 79.7 MiB [79.4–80.1] | **~25 MiB** | 0.14 [0.13–0.14] | 0.14 [0.12–0.14] | 0.9 s at 3000 | 10 / 12 |
| libtorrent via Rust | 27.4 MiB | not run | | 0.05 | | 0.4 s at 1000 | 3 / 12 |
| rain | 309 MiB [280–314] | 947 MiB [910–958] | ~320 MiB | 5.3 [4.7–5.7] | 14.6 [14.2–14.7] | 3.3 s at 3000 | 28–83 / 6012 |
| anacrolix | 619 MiB [582–639] | 1933 MiB [1744–2016] | ~660 MiB | 0.18 [0.16–0.19] | 0.68 [0.39–0.71] | 14.7 s at 3000 | 10 / 8 |

The table uses `RssAnon`, the process's anonymous memory. libtorrent does its disk I/O through mmap, so its `VmRSS` also counts page cache: about 2 GB for 20 seeds that had served data. `RssAnon` and PSS are the honest comparison. Raw runs are in `results/seed.results.jsonl`.

What limits each number:
- **anacrolix memory is per-piece state, doubled by Go's GC target (measured).**
  - The heap profile at 1000 torrents puts 184 MB in `(*Torrent).makePieces` and 52 MB in `request-strategy.NewPieceOrder`. That is about 300 bytes of live state per piece.
  - `HeapInuse` was 641 MB against roughly 300 MB live, which is the `GOGC=100` headroom.
  - `GOGC=25` brought it to 511 MiB. `GOMEMLIMIT=350MiB` brought it to 476 MiB, but GC thrash took 2.7% of a core.
  - The cost scales with piece count, so a library of large torrents is the worst case.
- **rain CPU and fds scale with torrent count (inferred, not profiled).**
  - rain opens a TCP listener per torrent, which accounts for 2 fds per torrent. Its default port range, 20000–30000, caps it near 10,000 torrents.
  - It runs per-torrent goroutines and timers. Idle CPU tripled from 1000 to 3000 torrents.
- **libtorrent is flat.**
  - Its seeds in `seed_mode` hold no piece picker. Memory grows after a peer requests pieces and libtorrent verifies them; that state is not measured here.
  - Most of the Go harness's CPU is the 200 ms alert poll plus the Go runtime. The Rust harness idles at 0.05%.

## 2. Announce compliance

`run/scenario.py` runs each engine against the enforcing tracker. The engines run in parallel on separate ports and trackers.

| Scenario | libtorrent | rain | anacrolix |
| --- | --- | --- | --- |
| **Pacing**: 1000 torrents, one URL, `interval`=`min interval`=120 s, 9 min | Clean. 0 rejections. It announced every 300 s because of its `min_announce_interval` floor (default 300 s). | Clean. Gaps 120.0 s. | Clean. Gaps 120.0 s. |
| **Tiers**: 50 torrents, 2 tiers x 2 URLs on one site (one passkey) | One URL at a time, per BEP 12 and BEP 27. 150 announces, **0 rejected**. | One URL per tier, so 2 at once. 451 of 751 rejected. | Every URL in every tier. It retried each rejected `started` every minute. **1506 of 1798 rejected**. |
| **Failure**: tracker answers `failure reason` for 20 torrents, 10 min | One `started` per torrent, then no retry in the 10 min window (cause not traced; libtorrent documents a growing tracker backoff). Quietest, but a transient error would leave a torrent unannounced for a while. | Exponential backoff: retries at about 6, 15, 44, 88, 173 and 349 s. 145 rejected announces. | A retry every 60 s, flat, all `started`. 220 rejected announces, 11 per torrent. |
| **Shutdown**: SIGTERM after seeding (from §1) | Every `stopped` sent in all 7 runs. | Sent, but only after the harness calls `StopAll` and waits. `Close()` aborts in-flight `stopped`. | `Close()` sends none. The harness drops every torrent and polls. That still lost `stopped` for 2817 of 3000 and 362 of 1000 torrents in 2 of 6 runs. |
| **Pause and resume** (harness proofs) | Pause sends `stopped`. Resume sends `started` with the counters reset. Remove sends `stopped`. | Pause sends `stopped`. Resume sends `started`, which UNIT3D's window rejects within ~27 min. Remove sends nothing. | There is no pause. The announcers keep running. |

anacrolix's announce pacing at scale, the open question from #10:
- It never reads `min interval`. Its HTTP response struct has no such field. It schedules the next announce at the last success plus `interval`, retries no sooner than 60 s after an error, and caps in-flight announces at 2 per tracker URL (`client-tracker-announcer.go:408`, `:618`, `:756` at v1.61.0).
- With one URL per torrent and an honest tracker interval it stays inside UNIT3D's rules.
- It ignores tiers and announces to every URL (`torrent.go:2244-2252`). On a multi-URL private torrent that breaks BEP 27, trips UNIT3D's duplicate rule on every cycle, and makes it hammer the site with retries.
- The 2-in-flight cap serialises a whole library's announces to one site. At 250 ms of tracker latency, 4000 torrents take about 4000 / 2 x 0.25 s = 500 s per cycle (inferred from the cap, not run).

Two behaviours apply to every engine:
- **No engine jitters its announces.** All 1000 torrents hit the tracker in the same second, every cycle. UNIT3D's random 1800–3600 s `interval` spreads them out in practice. SlipStream should not rely on that.
- **UNIT3D rejects any `started` that lands within ~27 min of the window-opening announce**, unless a `stopped` arrived more than 5 min into the window. So a quick pause and resume earns a rejection on every engine that sends `stopped` on pause, and libtorrent is one of those. SlipStream's pause has to respect that window, either by not sending `stopped` for short pauses or by delaying the resume announce.

## 3. The libtorrent shim, from Go and from Rust

Source: `lt-shim/NOTES.md`.

- **Small and easy.** 8 `extern "C"` functions over an opaque session and `int64` torrent ids, with two POD structs. It is 281 lines of C++ and header. The Go bindings are 111 lines and the Rust bindings 115. It compiled and linked on the first attempt. Every obstacle came at runtime:
  - `ssrf_mitigation` blocked the loopback tracker. That setting is test-only.
  - A wildcard listen address double-announced.
- **Go costs.**
  - It needs `CGO_ENABLED=1`, so pure-Go cross-compiling is gone. `GOOS=windows` fails at once without a mingw toolchain.
  - A cgo call costs 53 ns. The harness polls for alerts rather than taking C-to-Go callbacks.
  - libtorrent's memory is invisible to `GOMEMLIMIT` and `MemStats`.
  - The race detector does not see C++.
- **Rust costs.** A hand-written extern block was enough, and bindgen was not needed at this size. The Rust process ran 3 threads at idle against 10 for Go, and 0.05% CPU against 0.14%.
- **Static binary.** Proven for Go. libtorrent built from source as a static library in 179 s, and the binary is 13.7 MB with `ldd` reporting "not a dynamic executable". Static glibc warns about NSS, so production should build against musl. Rust static was not tried.
- **Cross-platform.** Plan on native CI jobs per OS, each with libtorrent, boost and OpenSSL built static. A cross-compile from one machine does not work for either language.
- **Production shim (guess).** About 40–60 functions: resume data, per-torrent control, bulk `post_torrent_updates`, per-tracker state, and about 20 typed alerts with error codes. That is 2–3 engineer-weeks, plus 1–2 weeks for static builds and CI on three OSes.

## Not done

- **Scenarios skipped.** The engine was locked in before two scenarios ran:
  - the lifecycle scenario with a fast download, which would show the resume `started` rejection live;
  - pacing at 4000 torrents behind 250 ms of tracker latency.
- **Not traced.** Why libtorrent did not retry a failed announce within 10 min.
- **Not measured.**
  - Memory after seeds serve real traffic, when libtorrent verifies pieces and grows its piece state.
  - CPU under upload load.
- **uTP.** It is about 50x slower than TCP over loopback in libtorrent (`lt-shim/NOTES.md`). It is irrelevant on loopback-only tests, but worth a look before the real build.

## Rerun

```bash
cd fixtures && go build -o fixtures . && cd ../tracker && go build -o tracker . && cd ..
(cd lt-shim && make) && (cd engine-lt-go && go build -o engine-lt-go .) && (cd engine-lt-rust && cargo build --release)
(cd engine-anacrolix && go build -o engine-anacrolix .) && (cd engine-rain && go build -o engine-rain .)
run/seed.sh lt-go 1000 60 180 /tmp/out          # one idle-seeding run
run/batch-seed.sh /tmp/seed                     # the full section 1 matrix
python3 run/scenario.py anacrolix tiers /tmp/sc # one compliance scenario
```

Needs `libtorrent-rasterbar-dev` from apt.
