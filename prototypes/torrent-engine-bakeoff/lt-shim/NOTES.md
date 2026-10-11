# libtorrent shim, Go harness, Rust harness

Covers `lt-shim/`, `engine-lt-go/` and `engine-lt-rust/`. Throwaway prototype code for issue #33. All numbers below come from this box (4 vCPU, 16 GB, Linux 6.18, libtorrent-rasterbar 2.0.10 from the distro, Go 1.24.7, Rust 1.97).

## Build and run

```bash
cd lt-shim && make                                   # liblts.a + liblts.so
cd ../engine-lt-go && go build -o engine-lt-go .     # links liblts.a statically, libtorrent dynamically
cd ../engine-lt-rust && cargo build --release        # target/release/engine-lt-rust
```

Build the shim first. Both harnesses link `../lt-shim/liblts.a`.

```bash
engine-lt-go/engine-lt-go --mode seed --fixtures DIR --listen-port 17402 --peer-id-prefix -SG0001- --status st.jsonl
engine-lt-rust/target/release/engine-lt-rust --mode lifecycle --fixtures DIR --index 0 --data-dir EMPTY --listen-port 17421 --status st.jsonl
```

Both take the contract flags plus `-v`, which prints every alert to stderr. `error` status lines are written for failed adds, tracker errors and any alert in libtorrent's error category.

`lt-shim/prove/prove-seed.sh` and `lt-shim/prove/prove-life.sh` rerun the proofs below. Each starts its own tracker (`TRACKER=path` overrides `../../tracker/tracker`) and prints the tracker-log summary.

## Shim surface

`lts.h` holds 8 functions, 2 plain structs, an opaque `lts_session*` and an `int64` torrent id.

| Function | What |
| --- | --- |
| `lts_session_new(preset, keys, values, n, err, errlen)` | Session from string key/value pairs, resolved with `setting_by_name` and typed by the setting index. `preset` picks libtorrent defaults or the seedbox set below. |
| `lts_add(s, torrent_file, save_path, seed_mode, err, errlen)` | Synchronous `add_torrent`. Returns an id > 0, or -1 with the exception text in `err`. |
| `lts_pause`, `lts_resume`, `lts_remove` | By id. Remove keeps data. |
| `lts_status_get(s, id, *lts_status)` | state, paused, seeding, finished, peers, seeds, rates, progress, done/wanted, all-time up/down. |
| `lts_poll_alerts(s, buf, max, timeout_ms)` | `wait_for_alert` + `pop_alerts`, copied into `lts_alert {type, lt_type, torrent, what[48], message[456]}`. |
| `lts_close(s)` | `session::abort()`, then the `session_proxy` destructor blocks until libtorrent has sent `stopped` announces. |

Torrent ids are opaque `int64`s. `add_torrent_params::userdata` carries the id so `torrent_removed_alert` can still be mapped after the handle dies. Other torrent alerts map through an `unordered_map<torrent_handle, id>`. Every entry point catches every exception. Alert types that the harnesses act on map to a small `LTS_ALERT_*` enum, and the raw libtorrent type id rides along in `lt_type`.

Two contract rules are on the caller. `lts_poll_alerts` has a single consumer, and its alert pointers stay valid only until the next `pop_alerts`. `lts_close` must not race a poll that is still running. Both harnesses stop and join the poller before they close.

Line counts (`wc -l`) are below.

| File | Lines |
| --- | --- |
| `lt-shim/lts.h` | 68 |
| `lt-shim/lts.cpp` | 213 |
| `lt-shim/Makefile` | 21 |
| `engine-lt-go/lt.go` (cgo bindings) | 111 |
| `engine-lt-go/main.go` | 200 |
| `engine-lt-rust/src/lt.rs` (FFI bindings) | 115 |
| `engine-lt-rust/src/main.rs` | 174 |
| `engine-lt-rust/build.rs` + `Cargo.toml` | 21 |

The Rust harness uses a hand-written `extern "C"` block, not bindgen. The surface is 8 functions and 2 POD structs, so writing it by hand took less time than wiring bindgen up. bindgen would add libclang as a build dependency, plus a target sysroot when cross-compiling, for no gain at this size. The cost is that struct layout is checked by eye only. A production shim should either use bindgen or `static_assert` the `sizeof`/`offsetof` values on both sides.

## Settings applied

The shim's `LTS_PRESET_SEEDBOX`, used by both harnesses, starts from `lt::high_performance_seed()`. It then applies the table in `lts.cpp` and the runtime keys. This keeps one source of truth, so Go and Rust run with identical settings.

Contract-required settings.

- `enable_dht=false`, `enable_lsd=false`, `enable_upnp=false`, `enable_natpmp=false`.
- `active_downloads=-1`, `active_seeds=-1`, `active_limit=-1`, `active_tracker_limit=-1`, `dont_count_slow_torrents=true` (the default).
- `announce_to_all_tiers=false`, `announce_to_all_trackers=false` (the defaults, set explicitly for BEP 12 behaviour).
- `peer_fingerprint=<--peer-id-prefix>` when given. The tracker log shows it, for example `peer_ids ['-SG0001-']`.
- PEX is off for private torrents by libtorrent itself (BEP 27), so nothing is set for it.

Torrents are added with `auto_managed` and `paused` cleared. libtorrent's queue only governs auto-managed torrents ("Torrents that are not auto managed are not counted against these limits", `settings_pack.hpp`). So the `active_*` limits are inert here and are set only because the contract asks for them. Clearing `auto_managed` is also what makes `pause()` stick. On an auto-managed torrent, the queue may resume it at any time.

Seedbox tuning. These are the settings `high_performance_seed()` changes from defaults, dumped from the library at runtime (`/tmp/lt-scratch/presets.cpp`) rather than copied from docs.

| Setting | Value | Default |
| --- | --- | --- |
| `connections_limit` | 8000 | 200 |
| `unchoke_slots_limit` | -1 | 8 |
| `file_pool_size` | 500 | 40 |
| `alert_queue_size` | 10000 | 2000 |
| `connection_speed` | 500 | 30 |
| `listen_queue_size` | 3000 | 5 |
| `request_timeout` | 10 | 60 |
| `peer_timeout` | 20 | 120 |
| `inactivity_timeout` | 20 | 600 |
| `max_failcount` | 1 | 3 |
| `max_rejects` | 10 | 50 |
| `max_out_request_queue` | 1500 | 500 |
| `max_queued_disk_bytes` | 7340032 | 1048576 |
| `send_buffer_low_watermark` | 1048576 | 10240 |
| `send_buffer_watermark` | 3145728 | 512000 |
| `send_buffer_watermark_factor` | 150 | 50 |
| `send_not_sent_low_watermark` | 524288 | 16384 |
| `max_peer_recv_buffer_size` | 5242880 | 2097152 |
| `checking_mem_usage` | 2048 | 256 |
| `aio_threads` | 8 | 10 |
| `suggest_mode` | 1 (suggest read cache) | 0 |
| `allowed_fast_set_size` | 0 | 5 |
| `mixed_mode_algorithm` | 0 (prefer_tcp) | 1 |
| `active_seeds`, `active_limit`, `active_tracker_limit` | 2000, 20000, 2000 | overridden to -1 by the table below |
| `active_dht_limit` | 600 | 88 (inert, DHT is off) |

Beyond the contract-required keys, the seedbox table overrides two more settings, taken from `min_memory_usage()` and the tuning guide ("500 is probably enough"). Both are memory costs paid per torrent, which dominates when thousands of torrents sit idle.

- `max_peerlist_size=500` (default 3000).
- `max_paused_peerlist_size=50` (default 1000).

`alert_mask` is `error | status | tracker | storage`.

Runtime keys set by each harness.

- `listen_interfaces=127.0.0.1:<port>`. With `0.0.0.0` libtorrent 2.0 announces once per local listen address. Each torrent sent 2 `started` announces, one from `127.0.0.1:17400` and one from `192.0.2.2:17400` (`-v` alert output). A production box should bind the one interface it wants to announce from.
- `ssrf_mitigation=false`. **Test-environment only.** With it on, every announce to the loopback tracker failed with `blocked by SSRF mitigation`. Production should leave it on.

Seed mode uses `torrent_flags::seed_mode`. Nothing is hashed at add time. libtorrent verifies each piece the first time a peer requests it. `all_added` is written after every torrent's status reports `is_seeding`.

## Proof

All final runs use the contract fixtures generator (20 torrents, 4 MiB x 1024 pieces, private), the contract tracker (rebuilt from `tracker/main.go` at 02:50 into `/tmp/lt-scratch/ltbk-trk`), tracker ports 7450/7451 and engine ports 174xx.

Seed mode, Go (`proof-seed-go.txt`).

```
engine exit=0
{"count":20,"elapsed":0.011657478,"event":"all_added",...}
{"elapsed":0.18627159,"event":"shutdown_done",...}
announces {'started': 20, 'stopped': 20}
started unique infohashes 20
stopped unique infohashes 20
peer_ids ['-SG0001-'] ua ['libtorrent/2.0.10.0'] rejected 0 violations 0
```

Seed mode, Rust (`proof-seed-rs.txt`).

```
engine exit=0
{"count":20,"elapsed":0.007809338,"event":"all_added",...}
{"elapsed":0.146349223,"event":"shutdown_done",...}
announces {'started': 20, 'stopped': 20}
started unique infohashes 20
stopped unique infohashes 20
peer_ids ['-SR0001-'] ua ['libtorrent/2.0.10.0'] rejected 0 violations 0
```

Lifecycle mode. Each run downloads one 4 GiB torrent from a seed-mode instance of the same harness on another port. Below is the first Go run (tracker on 17370, before the port assignment). The tracker events for that infohash follow the status lines.

```
lifecycle exit=0 wall=725s
{"step":"added","elapsed":0.0005}
{"step":"completed","elapsed":579.5,"total_done":4294967296}
{"step":"paused","elapsed":624.6}
{"step":"resumed","elapsed":669.6}
{"step":"removed","elapsed":714.6}
{"step":"shutdown","elapsed":724.6}
0.0   -SEED01- started   left 0
2.0   -LIFE01- started   left 4294967296
581.5 -LIFE01- completed left 0 down 4294967296
626.6 -LIFE01- stopped   (pause)
671.6 -LIFE01- started   (resume, down/up counters reset to 0)
716.6 -LIFE01- stopped   (remove)
726.7 -SEED01- stopped   up 4294967296
```

The Rust run had the same shape (`completed` at 595.3 s, wall 740 s).

Final reruns used the 02:50 tracker with the UNIT3D `checkMinInterval` rules, on ports 7450/7451, with both harnesses running in parallel (`/tmp/lt-scratch/proof-life-{go,rs}.txt`).

```
Go    lifecycle exit=0 wall=517s  completed 372.2  paused 417.2  resumed 462.2  removed 507.2  shutdown 517.3
Rust  lifecycle exit=0 wall=496s  completed 351.3  paused 396.3  resumed 441.3  removed 486.3  shutdown 496.3
tracker (both): started, completed, stopped, started, stopped for -LIFE01-, then -SEED01- stopped with up 4294967296
violations: [] for both
```

The peer id stayed the same across pause and resume (`peer_id_hex` matched on all 5 announces). The resume `started` escaped `min_interval_violation` only because the pause `stopped` came more than 5 minutes after the window opened, and the tracker resets the window in that case (`tracker/main.go`). That is inferred from the tracker code, not observed. With a fast download (TCP, about 13 s for 4 GiB, extrapolated from the 1 GiB bench), the pause `stopped` would land about 60 s in. The resume `started` would then be flagged.

What libtorrent does to the tracker during the lifecycle is worth knowing for UNIT3D-style rules. `pause()` sends `stopped`. `resume()` sends a fresh `started` with uploaded/downloaded reset to 0. `remove_torrent` sends `stopped`. `session::abort()` sends nothing for a torrent that was already removed.

## Surprises

**uTP over loopback is about 50x slower than TCP.** The lifecycle download ran at about 7 MB/s with CPU idle and every libtorrent thread parked in `futex_do_wait`/`ep_poll`. A scratch bench (`/tmp/lt-scratch/dlbench`, 1 GiB torrent, seed-mode seeder) shows the cause.

```
default settings (uTP on)                         did not finish in 40 s  (< 27 MB/s)
enable_outgoing_utp=false enable_incoming_utp=false, seedbox preset   3.3 s  327 MB/s
same, libtorrent default preset                                       3.4 s  312 MB/s
same, seeder fully checked (pages cached) instead of seed_mode        1.4 s  768 MB/s
```

The harnesses keep uTP on, because the contract says to use engine defaults. Turning it off is one key in the seedbox table and would cut a lifecycle run from about 12 minutes to about 2.5. That decision belongs to the coordinator.

**RSS is mostly page cache.** libtorrent 2.0 does disk I/O through mmap. VmRSS of a seed with 20 x 4 GiB torrents was about 2.0 GB, but `RssAnon` was 8 MB and `RssFile` was 2.0 GB (`/proc/PID/status`). Compare engines on `RssAnon` or PSS. `measure/` now records these fields.

**Tracker log buffering.** The contract tracker buffers its log, so a count taken while it runs comes up short. Truncating the log under a running tracker corrupted it with NUL bytes. The proof scripts start a fresh tracker per run and count after its SIGTERM.

**Another process ran `pkill -x tracker`** and killed my tracker mid-run. The proofs use a renamed copy (`ltbk-trk`).

## Build obstacles

The shim compiled and linked on the first attempt. The obstacles were all at runtime or in packaging.

1. SSRF mitigation blocked the loopback tracker. Fixed with `ssrf_mitigation=false` (above).
2. Double announces with a wildcard listen address. Fixed by binding `127.0.0.1:<port>` (above).
3. cgo needs `${SRCDIR}` in `#cgo` lines to find `../lt-shim`, and needs an explicit `-lstdc++` because the archive is C++ while the final link is driven by the Go toolchain's C link. Rust needs the same libraries in `build.rs` (`static=lts`, then `torrent-rasterbar ssl crypto stdc++`).
4. The distro ships no `libtorrent-rasterbar.a`, so a static binary needs libtorrent built from source (below).
5. `go build` from outside the module root fails (`cannot find main module`). The harness must be built from `engine-lt-go/`.

## Build times

| Step | Time |
| --- | --- |
| `lt-shim` clean `make` (one TU, but heavy libtorrent/boost headers) | 4.6 s |
| Go clean build, fresh `GOCACHE` (includes runtime/cgo and stdlib) | 7.4 s |
| Go incremental | 0.7 s |
| Rust clean `--release` (includes serde, serde_json, signal-hook) | 8.3 s |
| Rust incremental `--release` | 1.0 s |
| libtorrent 2.0.10 from source, static, `make -j3` | 179 s |

## Binary sizes and dynamic links

| Artifact | Size |
| --- | --- |
| `liblts.a` / `liblts.so` | 99.6 KB / 75.0 KB |
| `engine-lt-go` (default) | 3.13 MB |
| `engine-lt-go` (`-ldflags "-s -w"`) | 2.11 MB |
| `engine-lt-rust` (release, `strip = true`) | 0.65 MB |
| system `libtorrent-rasterbar.so.2.0.10` (needed at runtime by both) | 4.73 MB |
| fully static Go harness, stripped | 13.7 MB |

`ldd` lists the same set for both harnesses: `libtorrent-rasterbar.so.2.0`, `libssl.so.3`, `libcrypto.so.3`, `libstdc++.so.6`, `libgcc_s.so.1`, `libm.so.6`, `libc.so.6`, `ld-linux-x86-64.so.2`. boost is header-only here (asio, system).

## Fully static binary

It is feasible, and it was done for Go. `libssl.a`, `libcrypto.a`, `libstdc++.a` and `libboost_system.a` are installed. libtorrent was built from the release tarball.

```bash
cmake -S libtorrent-rasterbar-2.0.10 -B build -DCMAKE_BUILD_TYPE=Release -DBUILD_SHARED_LIBS=OFF \
  -Ddeprecated-functions=OFF -DCMAKE_CXX_STANDARD=17 -DCMAKE_INSTALL_PREFIX=/tmp/lt-scratch/lt-static
# shim recompiled against that prefix's pkg-config cflags (ABI defines differ: TORRENT_NO_DEPRECATE, no TORRENT_LINKING_SHARED)
# cgo LDFLAGS: liblts.a .../libtorrent-rasterbar.a -lssl -lcrypto -lstdc++ -lm -lpthread -ldl
go build -ldflags '-s -w -linkmode external -extldflags "-static"'
```

The result is `statically linked`, `ldd` reports `not a dynamic executable`, and it passed the 20/20 seed proof. The linker warns that `getaddrinfo`, `gethostbyname` and `dlopen` in static glibc need the matching glibc's NSS libraries at runtime. A tracker given by IP works. A tracker given by hostname depends on the host's NSS setup, so production should build against musl (Alpine, or `zig cc`) instead of static glibc. Rust would take the same libraries with `-C target-feature=+crt-static` or a musl target. That was not tried.

## Cross-compiling (Windows, macOS from Linux)

Neither harness cross-compiles out of the box. Both need libtorrent, OpenSSL and boost headers built for the target, plus a C++ cross toolchain.

- Go. `CGO_ENABLED=1` is mandatory. `GOOS=windows CGO_ENABLED=1 go build` fails at once with `gcc: error: unrecognized command-line option '-mthreads'`, because the host gcc is not a Windows compiler. With `CGO_ENABLED=0` the cgo files drop out and the build fails with `undefined: newSession`. It needs `CC`/`CXX` set to mingw-w64 or `zig cc -target x86_64-windows-gnu` for Windows, and osxcross or zig plus a macOS SDK for macOS. libtorrent on macOS links system frameworks, so the SDK is required. Go's usual "set GOOS and build" story is gone.
- Rust. `cargo build --target x86_64-pc-windows-gnu` fails here only because that target's std is not installed. `rustup target add` fixes that part. The shim and libtorrent still need a target C++ toolchain, wired in through `build.rs` (cc crate) or `cargo-zigbuild`. The hand-written extern block needs no libclang. bindgen would need libclang plus target headers.
- In practice, both languages are easier to build natively on each OS's CI runner, with libtorrent from vcpkg (it has a libtorrent port, from memory and not checked here) or from source via CMake. Plan on three native CI jobs instead of one cross-compile.

## cgo-specific costs

- Call overhead is 53 ns per call into a no-op C function (`/tmp/lt-scratch/cgobench`, 20M calls, box under load from two downloads). That is negligible for this API. A Rust FFI call is a plain function call with no runtime transition (not measured).
- Threads. The Go seed process ran 17 threads, against 11 for the Rust one (`/proc/PID/status`). libtorrent owns about 9 of them (network, disk I/O, hashing). The Go runtime adds its Ms, plus one locked while the alert goroutine sits in `wait_for_alert` for up to 200 ms per call.
- No C-to-Go callbacks. The shim polls (`wait_for_alert`) instead of using `set_alert_notify`. That callback fires on libtorrent's network thread, which would mean a cgo callback from a foreign thread (it needs an M bound to it and must not block). Polling keeps every crossing Go-to-C.
- What `CGO_ENABLED=1` costs. Every developer and CI machine needs g++, libtorrent and pkg-config. The pure-Go cross-compile story is gone. Builds are no longer reproducible from `go.mod` alone. `go vet`/`go test` pay cgo compile time. libtorrent's memory is invisible to the Go runtime (`GOMEMLIMIT`, `runtime.MemStats`). The race detector does not see C++ data races.
- Signals. Go installs its own handlers. libtorrent installs none. SIGTERM handling worked as expected in both harnesses. SIGPIPE raised on a non-Go thread would kill a Go program. asio sends with `MSG_NOSIGNAL`, so it should not arise. That is a guess, not tested.

## Effort estimate for a production-grade shim

This is a guess, sized against the 213-line shim.

What it would need beyond this one.

- Resume data. `save_resume_data` driven by `save_resume_data_alert`, serialized with `write_resume_data_buf`, and passed out as a shim-owned byte buffer with a free function. Load it back through `read_resume_data` on add. Periodic and shutdown flush.
- Per-torrent control. Rate limits, max connections and uploads, super-seed, `force_recheck`, `force_reannounce`, `move_storage`, `replace_trackers`, file priorities, rename. Magnet links and the metadata-received flow.
- Bulk status and stats. `post_torrent_updates` returning arrays of status structs. Each `lts_status_get` today is one synchronous round trip to the network thread, which does not scale to thousands of torrents. Session counters come from `post_session_stats` plus `session_stats_metrics` names. Per-tracker state (`announce_entry`: last error, next announce, per-endpoint status) is needed to show tracker health.
- Typed alerts. libtorrent has about 100 alert types. SlipStream would need maybe 20 with typed payloads (tracker reply peers, tracker error code and failure reason, file error path and `error_code`, storage moved, hash failed, metadata received, state changes). `message()` strings are not a stable interface. Errors should cross as category plus value, not text.
- ABI hygiene. A version or size field in each struct, ownership rules for every buffer, and layout asserts or bindgen.
- Build and packaging. CMake for the shim on 3 OSes, libtorrent, boost and OpenSSL built static per target, and CI for all three.

Estimate: 2 to 3 engineer-weeks for a shim of about 40 to 60 functions (roughly 1.5k to 2.5k lines of C++) plus Go bindings. Another 1 to 2 weeks for cross-platform static builds and CI. After that, ongoing upkeep on each libtorrent upgrade, since the shim is where libtorrent API churn lands. Rust bindings on top would add about 1 week.
