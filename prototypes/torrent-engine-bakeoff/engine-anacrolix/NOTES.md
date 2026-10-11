# engine-anacrolix (github.com/anacrolix/torrent v1.61.0, Go 1.24.7)

Build: `cd engine-anacrolix && /usr/local/go/bin/go build -o engine-anacrolix .` -> 32,320,144 bytes (unstripped, `ls -l`).

Run: `engine-anacrolix --mode seed --fixtures DIR --listen-port P [--peer-id-prefix -XX0000-] [--user-agent S] --status F`
and `--mode lifecycle --fixtures DIR --index I --data-dir EMPTY --listen-port P --status F`.
Extra: env `HOLD_AFTER_DROP=4s` sleeps between dropping torrents and closing the client (used to look at post-stop announce behaviour).

## Settings applied
`NoDHT`, `DisablePEX`, `NoDefaultPortForwarding` (UPnP/NAT-PMP), `DisableIPv6`, `Seed=true`, `ListenPort=P`, `EstablishedConnsPerTorrent=8` (default 50), `TotalHalfOpenConns=1000` (default 100), `HalfOpenConnsPerTorrent=8`, `DialRateLimiter=rate.NewLimiter(rate.Inf,1)` (default 10/s; nil panics in `dialAndCompleteHandshake`, client.go:837). `Bep20` set only with `--peer-id-prefix`; HTTP user agent only with `--user-agent`. Defaults otherwise: peer id `-GT0003-`+12 random, UA `anacrolix-torrent/v1.61.0` (seen in tracker log). No limit on active torrents exists in anacrolix. Local service discovery: this version has no LSD in `ClientConfig`; none ran. Private torrents skip PEX anyway.

## Skip-hash seeding
File storage (`storage.NewFileOpts`, `UsePartFiles=false`, base dir = `FIXTURES/data`) gets `PieceCompletion` = `allComplete`: `Get` returns `{Ok:true, Complete:true}`, `Set` no-op, `Persistent()` false. Torrents are added with `AddTorrentSpec` and `DisableInitialPieceCheck=true`. Nothing is hashed: `queueInitialPieceCheck` returns early when `storageCompletionOk` (torrent.go:2905-2921). Proof: the store saw exactly 1024 `Get` calls per torrent (20480 for 20, 1,024,000 for 1000), one per piece, and 1000 4 GiB torrents were "added and seeding" in 5.7 s (about 5.7 ms each, all CPU in setup, no disk reads).

## Lifecycle
Storage is file storage with the default map completion in `--data-dir`. pause = `DisallowDataDownload`+`DisallowDataUpload`+`SetMaxEstablishedConns(0)`; resume reverses it; remove = `Torrent.Drop()` (keeps data on disk). anacrolix has no real pause: a "paused" torrent keeps its tracker announcers, so no `stopped` goes out on pause (tracker log of one run: `started`, `completed`, then `stopped` only at remove). Lifecycle SIGTERM: drop, wait for stopped, close, exit 0.

## Shutdown sends `stopped`, but only if you wait
`Client.Close()` (client.go:537) drops torrents and returns; it does not flush `stopped`. The announce dispatcher sends `stopped` asynchronously after a drop (client-tracker-announcer.go:745-748), so a bare Close exits with zero `stopped` sent (observed: 20 `started`, 0 `stopped`). The harness drops every torrent, then polls `Client.WriteStatus` until no row contains `stopped` (hack, 120 s cap), then closes. Result: 20 started + 20 stopped; 1000 started + 1000 stopped, shutdown 0.82 s.

## Announce behaviour (v1.61.0, client-tracker-announcer.go unless noted)
- New dispatcher, one per client (`regularTrackerAnnounceDispatcher`, :23). Announces are priority-ordered across all torrents per tracker URL and capped at `maxConcurrentAnnouncesPerTracker = 2` in-flight requests per tracker URL (:408, enforced :618). With 1000 torrents on one URL, starts and stops are therefore serialised two at a time (still 5.7 s total on loopback; on a real tracker it is RTT-bound).
- Next regular announce = last OK time + the tracker's `interval` (:769, interval stored :577). `min interval` is never read: no `MinInterval` symbol anywhere outside tests, and the HTTP response struct has no such field (tracker/http/protocol.go:13).
- On error, retry no sooner than 1 minute after the last attempt (:756). Announce timeout 15 s (tracker/tracker.go:58).
- Event sequence: `started` first (:766), `completed` only if the torrent was seen with missing data and is now complete (:761), else regular. `stopped` after drop (:745-748) only if an announce ever succeeded.
- Tiers are not honoured. `startMissingTrackerScrapers` loops over every tier and every URL and starts an independent announcer for each (torrent.go:2244-2252). Observed: announce-list `A1,A2;B1` with 3 torrents -> 3 `started` on each of the three URLs (9 total). No tier fallback, no promotion of the working URL, no shuffle within tier.
- Misbehaviour seen with several URLs per torrent: after dropping, each URL got a burst of `stopped`, plain (no-event) and `stopped` again in a few ms (3 torrents x 3 URLs: 23-33 `stopped` and 15-25 empty-event announces within ~10 ms, many tripping `min_interval_violation`/`duplicate_event_30s`). With one URL per torrent it was exactly one `stopped` each. Cause not traced; looks like the post-`stopped` entry (zero `When`, treated as overdue) being re-announced. Caveat: the tracker was run with one passkey on three listeners, so the `duplicate_event_30s` flags on the initial `started` are a test artifact, the burst is not.
- `numwant`: seeder logged 50 (tracker default for 0); code sends 200 if the torrent wants peers else 0 (torrent.go:2265-2273).
- Idle RSS with 1000 torrents seeding: 661 MB (`VmRSS`), 10 threads, 8 fds (one shared listener, no port per torrent).

## Proof (real tracker/fixtures, 20 torrents)
Seed: `started` 20, SIGTERM -> `stopped` 20, `shutdown_done` written. Lifecycle (anacrolix leecher, anacrolix seeder, 4 GiB): added 0 s, completed +4.7 s, paused +45, resumed +45, removed +45, shutdown +10. Cross-engine with rain: see end of this file for the final run.

Cross-engine run (tracker :7300/:7301, engines :16000+): anacrolix lifecycle downloaded `r-000001` from the rain seed-mode instance (added +0, completed +22 s, paused, resumed, removed, shutdown, all steps written; tracker saw `started`, `completed`, `stopped`). anacrolix seed with 20 torrents on this run: 20 `started`, 20 `stopped` on SIGTERM.
