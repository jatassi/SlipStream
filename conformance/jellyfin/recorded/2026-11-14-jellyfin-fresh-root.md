# Jellyfin fresh-root recorded run, 2026-11-14 (S35 (Media Servers), #160)

Recorded 2026-11-14 for S35 (Media Servers), issue #160, step 2. It is the "Jellyfin fix is unproven" gate under "The fresh folder rule".

**Result: pass on Jellyfin 12.2.0 (the floor) and 12.4.0 (the newest image).** A full scan that runs while the new file is on disk makes Jellyfin take later `Created` reports under that root. The rule holds after the root empties again.

## Setup

- Throwaway `jellyfin/jellyfin:12.2` and `:12.4` containers on Colima on Jackson's Mac. Each run had a fresh config, cache and media volume, and all of them were removed afterwards. Image digests are in `<version>/image.txt`.
- One movie library with two roots. `/media/seeded` held one movie and `/media/fresh` was empty.
- Realtime monitoring off, internet providers off, and the `RefreshLibrary` task's triggers cleared. `LibraryMonitorDelay` was set from its default of 60 to 5 so that each wait is short.
- Every report is one `POST /Library/Media/Updated` with the file's path, as the spec's sender sends it. Each returned 204 in under 40 ms.
- The probe polls `GET /Users/{id}/Items?Recursive=true&IncludeItemTypes=Movie&Fields=Path` every second. The full scan is `POST /Library/Refresh`, and it has ended when the `RefreshLibrary` task reads `Idle`.

## Cases

| Case | 12.2.0 | 12.4.0 |
|---|---|---|
| 1. The library was scanned while `/media/fresh` was empty. A file lands there and is reported `Created`. | Absent after 90 s (#101 reproduced) | Absent after 90 s |
| 2a. Full scan with that file on disk | Picks it up. The scan took 0.5 s | Same |
| 2b. A second file lands and is reported `Created` | Appears after 5.1 s, that is the monitor delay | Same |
| 3a. A file is deleted, its folder kept, and reported `Deleted` | Gone after 5.1 s | Same |
| 3b. A movie folder is removed and its file reported `Deleted` | Gone after 5.2 s | Same |
| 3c. The root is empty again in Jellyfin, then a file is reported `Created` | Appears after 5.2 s | Same |
| 4a. Two movie folders removed, two `Deleted` calls 0.1 s apart | Both gone within 5.2 s | Same |
| 4b. Two movie folders removed, one call with both `Deleted` paths | Both gone within 5.2 s | Same |

`pass` in `results.json` needs case 1 to fail and cases 2a and 2b to succeed.

## What this means for the build

- **The rule as the spec words it works.** Check for an item under the mapping's `server_folder`. When there is none, run a full scan and send after it ends. The bytes are already on disk when the change is queued, so the scan itself finds the first file.
- **A full scan of an empty root does not fix it.** Case 1's root had been in a full scan, while empty, before the report. `scanned_at` must be set only after a scan that ran with content under the folder, or after the sender's check found an item there. This is what the spec says, and the fake needs to copy it: a scan that finds nothing under a root leaves that root fresh.
- **Once fixed, the root stays fixed** for the server's life in these runs, even with no items left under it (3c). A one-time `scanned_at` per mapping is enough. A config wipe or a library recreated on the server is not covered here.
- **The scan-end signal needs care on a small library.** `RefreshLibrary` went from `Running` to `Idle` in 0.5 s. A sender polling every few seconds can miss `Running` entirely. `LastExecutionResult.EndTimeUtc` moves forward on every scan, so "`Idle` and `EndTimeUtc` later than the request" is a signal that can't miss. The recorded task bodies are in `exchanges.jsonl`.

## One observation that did not reproduce

`12.2-unreproduced-delete/` is the first full 12.2 run. Its fresh-root cases match the table. In its delete step, both movie folders were removed and two `Deleted` reports went 0.1 s apart. Neither item left Jellyfin within 90 s each. Cases 3b, 4a and 4b repeat that shape and passed on both versions in later runs, so the cause is unknown. It is kept in case a conformance case for removals fails the same way.

## Files

`2026-11-14-jellyfin-fresh-root/<version>/` holds, for 12.2, 12.4 and the first 12.2 run (`12.2-unreproduced-delete`):

- `exchanges.jsonl`, one line per call: step, method, path, query, request body, status, latency and response body. The access token and every password are redacted, and the `Authorization` header is not recorded. These are the bodies `fake-jellyfin` copies: `/System/Info/Public`, `/Library/VirtualFolders`, the `RefreshLibrary` task with `State` and `LastExecutionResult`, `/Users/AuthenticateByName` and the item listing.
- `results.json`, the outcome per case, and `image.txt`, the image digest.

The probe (`probe/run.sh`, `probe/fresh_root.py`), `run.log` and `server.log` stay with the run in the project folder.

## In the suite

`FreshRootNeedsScanWithContent` in `conformance/jellyfin` replays cases 1 to 3c against fake-jellyfin on every PR, and `ScanEndSignal` and `ReportDroppedByScanIsSentAgain` use the "`Idle` and `EndTimeUtc` later than before the request" rule. Against a real server the case skips, since SlipStream adds no library folder, and this run stands for it.
