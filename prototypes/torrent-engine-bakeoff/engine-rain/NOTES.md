# engine-rain (github.com/cenkalti/rain v1.13.0, Go 1.24.7)

Build: `cd engine-rain && /usr/local/go/bin/go build -o engine-rain .` -> 17,444,014 bytes (+ a few KB after the NOFILE edit; `ls -l`).
Run: same flags as the contract (`--mode seed|lifecycle`, `--fixtures`, `--listen-port`, `--peer-id-prefix`, `--status`, `--index`, `--data-dir`). Engine logs go to stderr (INFO per torrent; large at 1000 torrents).

## Settings applied
`DHTEnabled=false`, `PEXEnabled=false`, `RPCEnabled=false`, `Host=127.0.0.1`, blocklists off, `DataDirIncludesTorrentID=false`, `ResumeOnStartup=true`, `MaxOpenFiles` = hard RLIMIT_NOFILE (rain calls setrlimit(cur=max=value) and errors if value > hard limit; box hard limit is 20000), `PrivatePeerIDPrefix` only with `--peer-id-prefix` (default `-RN0.0.0-`: `Version` is "0.0.0" in a plain `go build`; rain's own build injects it by ldflags). User agent default `Rain/0.0.0` (seen in tracker log). Everything else default (TrackerNumWant 200, TrackerMinAnnounceInterval 1 min, TrackerStopTimeout 5 s). Rain has no cap on active torrents. No UPnP/NAT-PMP code exists in rain.

## Skip-hash seeding: resume DB with a full bitfield
Rain has no "add as complete" API, but a torrent loaded from its bolt resume DB with a `bitfield` and existing files of the right size skips verification (torrent_allocation.go: `if t.bitfield != nil && !al.HasMissing` -> start announcing/uploading directly; session_load.go:`loadExistingTorrent`). `rain/internal/resumer/boltdbresumer` is `internal`, so the harness writes the same bbolt schema itself (bucket `torrents`/<id>/keys `info_hash port name trackers url_list fixed_peers info bitfield added_at bytes_* seeded_for started stop_* complete_cmd_run version=3`), with an all-ones bitfield, `started=true`, torrent id = infohash hex, then `NewSession` starts everything (`ResumeOnStartup`). The DB lives in a temp dir created per run. Nothing is hashed: 1000 x 4 GiB torrents -> `all_added` in 1.44 s (0.34 s writing the DB, 0.53 s `NewSession` load, rest waiting for `Seeding`). The schema copy is a version-3 snapshot and breaks if rain changes the format.

## Port per torrent, fds
Every started torrent opens its own TCP listener (torrent_start.go `startAcceptor`: `ListenTCP` on `spec.Port`), ports drawn from `PortBegin..PortEnd` (default 20000-30000, so ~10000 torrents max). The harness sets the range to `listen-port .. listen-port+N` and assigns port P+i to torrent i, so the engine announces 1000 different ports. Measured with 1000 torrents seeding: 2012 fds, 95 threads, RSS 329 MB. 20 torrents: 52 fds, 24 MB RSS. Needs `ulimit -n` of a few thousand, and 1000 free ports.

## Shutdown
`Session.Close()` closes torrents and aborts any in-flight stopped announcer (torrent_close.go: `stoppedEventAnnouncer.Close()`), so a bare Close sends no `stopped`. The harness calls `StopAll()`, waits until no torrent is `Stopping` (each sends `stopped` with a 5 s timeout), then Close. Result: 20 started / 20 stopped; 1000 started / 1000 stopped, `shutdown_done` 0.96 s.

## Lifecycle
pause = `Torrent.Stop()` (sends `stopped`), resume = `Torrent.Start()` (announces `started` again). Remove: rain's `RemoveTorrent` deletes the data (`RemoveAll`, session.go:stopAndRemoveData) and has no keep-data option, so the harness hard-links the file to `<name>.keep` first. `RemoveTorrent` also goes through `torrent.Close()`, so no `stopped` is sent at remove (observed: tracker log for the run is `started`, `completed`, `stopped` (pause), `started` (resume), nothing after). Downloaded 4 GiB from the anacrolix seeder: added -> completed 33 s. Walked all steps; shutdown +10 s.

## Announce behaviour (internal/announcer/periodic.go, internal/tracker/tier.go)
- `started` on start, `completed` once (not if complete at start), regular `""` afterwards; `stopped` only to trackers that answered at least once (`HasAnnounced`).
- Next announce = tracker `interval`; but while the torrent "needs more peers" it uses `minInterval` instead (periodic.go `getNextInterval`), which is the tracker's `min interval` if it sent one (>0) else the config `TrackerMinAnnounceInterval` (1 min). So rain reads `min interval` (httptracker/announceresponse.go:12) and never goes below it, but a leecher with few peers re-announces at the min interval, not the interval.
- Tiers: `NewTier` shuffles the URLs in a tier and uses one at a time, moving to the next only after a failure (tier.go:27-35). One announcer per tier, not per URL.
- Errors: exponential backoff 5 s start, x2, 30 min cap (periodic.go:82-88), or the tracker's `retry in` if given.
- Each announcer is its own goroutine/timer per torrent per tier; no global scheduling or concurrency cap, so 1000 torrents start announcing at once (1000 `started` within ~1.4 s).
- Pause/resume announces `stopped` then `started` again. Against the newer tracker rules `started` 45 s after the earlier `started` was flagged `min_interval_violation`.

## Proof
Seed 20: 20 `started`, SIGTERM -> 20 `stopped`. Cross-engine: rain lifecycle downloaded from the anacrolix seed-mode instance and anacrolix lifecycle downloaded from the rain seed-mode instance, both walked added/completed/paused/resumed/removed/shutdown.
