# Media Server conformance results (issue #71)

Run 2026-10-10 against containers on Colima (virtiofs), library mounted read-only at `/media`.

| Server | Version | Auth used by probes |
|---|---|---|
| Plex | 1.43.4.10903, unclaimed, `myPlexSubscription=false` (no Plex Pass), `ALLOWED_NETWORKS=0.0.0.0/0` | none (unauthenticated API answers 200) |
| Jellyfin | 12.2.0 | `Authorization: MediaBrowser Client=…, Device=…, DeviceId=…, Version=…, Token=…` |
| Emby | 4.10.1.0 | same header (also sent as `X-Emby-Authorization` and `X-Emby-Token`) |

## Rerunning

Rerun on each server release. From this directory: `./make_seed.sh`, `docker compose up -d`, then `probes/run.sh`. The runner generates a throwaway admin password into `creds.txt`, finishes each wizard, resets the library and runs every scenario. Logs and websocket captures land in `evidence/`, which is not committed. Plex is left unclaimed with `ALLOWED_NETWORKS=0.0.0.0/0`, so its API answers without a token. Set `PLEX_CLAIM` to claim it with an account instead.

Two full runs from a reset library gave the same results. The figures below are from the first run. The second measured 60.3 s and 60.2 s on Jellyfin and 90.3 s twice on Emby for scenario 1, 5.6 s for every Jellyfin and Emby removal, and again bound a 1080p Sidecar to every Heat version on Plex.

## Setup

| | Plex | Jellyfin | Emby |
|---|---|---|---|
| Wizard | none (unclaimed) | `POST /Startup/Configuration`, `GET /Startup/User`, `POST /Startup/User {Name,Password}`, `POST /Startup/RemoteAccess`, `POST /Startup/Complete`, all 204 | same five calls; `POST /Startup/User` answers 200 `{}` |
| Libraries | `POST /library/sections?name=Movies&type=movie&agent=tv.plex.agents.movie&scanner=Plex Movie&language=en-US&location=/media/movies` → 201 (Shows: `type=show`, `tv.plex.agents.series`, `Plex TV Series`) | `POST /Library/VirtualFolders?name=Movies&collectionType=movies&refreshLibrary=false&paths=/media/movies` body `{"LibraryOptions":{…}}` → 204 | same as Jellyfin → 204 |
| Realtime monitoring off | `PUT /:/prefs?FSEventLibraryUpdatesEnabled=0&FSEventLibraryPartialScanEnabled=0` → 200 (both were already `false` by default in this image) | `LibraryOptions.EnableRealtimeMonitor=false` at creation; read back `false` | same; read back `false` |
| Scheduled scans off | `ScheduledLibraryUpdatesEnabled=0` (already false), also `ButlerTaskRefreshLocalMedia=0`, `ButlerTaskRefreshPeriodicMetadata=0` | "Scan Media Library" task had an `IntervalTrigger` of 12 h; `POST /ScheduledTasks/{id}/Triggers []` → 204, read back `[]` | same, task "Scan media library" |
| Metadata | modern agents (online; titles were matched, e.g. episodes renamed "The Target") | `EnableInternetProviders=false`, empty fetchers; names come from the file/folder | same; episode names stay as the file name |
| Empty trash | `autoEmptyTrash=true` (default, left on) | n/a | n/a |

All disabling was possible over the API. No realtime pickup was observed in any server during the run (files only appeared after an explicit route).

## Scenario 0. Initial full scan and version grouping (also covers 5)

Library: `Heat (1995) - 1080p.mkv`, `Heat (1995) - 4K HDR.mkv`, `Heat (1995) {edition-Director's Cut} - 4K HDR.mkv`; `The Wire - S01E01 - 1080p.mkv`, `The Wire - S01E01 - Extended 1080p.mkv`, `The Wire - S01E02 - 720p.mkv`. A control folder `Collateral (2004)` with only `- 1080p` and `- 4K HDR` was added to separate the edition effect.

| | Plex | Jellyfin | Emby |
|---|---|---|---|
| Heat (3 files incl. `{edition-…}`) | **One movie, 3 Media.** `editionTitle` absent. The Director's Cut is not a separate item (no Plex Pass on this server; see deltas). | **Not grouped. Three separate Movie items**, each MediaSource `Name` = full file name, e.g. `"Heat (1995) - 1080p"`. | **Not grouped on the first scan.** Three items, three distinct `PresentationUniqueKey`s, names = full file name. |
| Heat without the edition file (moved aside, refresh) | n/a | **Grouped**: one item, MediaSources `Name` `"1080p"`, `"4K HDR"`. Moving the edition file back splits it into three again. | Grouped (shared `PresentationUniqueKey`, labels `"4K HDR"`, `"1080p"`). After moving the edition file back, Emby kept 1080p+4K grouped and made the edition its own item. A second fresh scan split Heat into three items again, so Emby's result depends on scan history. |
| Collateral control (2 files, no edition) | one movie, 2 Media | one item, labels `"1080p"`, `"4K HDR"` | grouped, labels `"4K HDR"`, `"1080p"` |
| The Wire S01E01 (1080p + Extended 1080p) | one episode, 2 Media | one Episode, MediaSources `"1080p"`, `"Extended 1080p"` | grouped, labels `"1080p"`, `"Extended 1080p"` |
| How versions appear in the API | `GET /library/metadata/{rk}` → `Metadata[0].Media[]`, each with `id, videoResolution, width, height, bitrate, container, videoCodec…` and `Part[]{id, file, key, size}`. **No label field**: `Media.title` is absent, so "1080p" vs "4K HDR" text is not exposed. | `GET /Users/{uid}/Items/{id}` → `MediaSources[]{Id, Name, Path, Container, Size, Bitrate, MediaStreams, …}`. Alternate versions have **no entry** in `/Items` listings; only the primary item is listed. Order: 1080p first. | **Every file is its own item** (ids 17 and 18 for Collateral) sharing `PresentationUniqueKey`. Listing endpoints (`/Items`, `/Users/{uid}/Items`) return only that item's own MediaSource with the full-file-name `Name`; the merged list with labels appears only on item detail `GET /Users/{uid}/Items/{id}` and `POST /Items/{id}/PlaybackInfo`. The user listing collapses grouped items to one. MediaSource ids are `mediasource_<itemId>`. Order: 4K first. |

Evidence (trimmed):

```
# Jellyfin, with edition file present (probes/dump_mb.py jellyfin)
Movie Heat 2f03… path: …/Heat (1995) - 4K HDR.mkv   MS name='Heat (1995) - 4K HDR'
Movie Heat 3cf3… path: …/Heat (1995) - 1080p.mkv    MS name='Heat (1995) - 1080p'
Movie Heat 73ac… path: …/Heat (1995) {edition-Director's Cut} - 4K HDR.mkv
Movie Collateral (2004) 5a20…  MS name='1080p' | MS name='4K HDR'
Episode The Wire - S01E01 - bc02…  MS name='1080p' | MS name='Extended 1080p'
# Plex (probes/dump_plex.py)
movie 'Heat' ratingKey=1 editionTitle=None
   Media id=2 4k  Part file=…/Heat (1995) - 4K HDR.mkv
   Media id=3 4k  Part file=…/Heat (1995) {edition-Director's Cut} - 4K HDR.mkv
   Media id=1 1080 Part file=…/Heat (1995) - 1080p.mkv
# Emby detail vs listing
GET /Users/{uid}/Items/17 MediaSources: [('mediasource_18','4K HDR'), ('mediasource_17','1080p')]
GET /Users/{uid}/Items (list): ('17', ['Collateral (2004) - 1080p'])
```

## Scenario 1. New video file

Route sent right after copying the file into place; latency is time from the route to the item being visible over the API (polled every 0.5 s).

| | Plex | Jellyfin | Emby |
|---|---|---|---|
| Route | `GET /library/sections/{id}/refresh?path=/media/movies/Ronin (1998)` → 200, 1–2 ms | `POST /Library/Media/Updated {"Updates":[{"Path":"/media/movies/Ronin (1998)/Ronin (1998) - 1080p.mkv","UpdateType":"Created"}]}` → 204 | same body → 204 |
| Movie visible | 0.6 s | 60.5 s | 90.6 s |
| Episode visible (S01E04, folder `Season 01`) | 0.5 s | 60.5 s | 14.9 s (an earlier pending timer fired; see below) |
| With debounce set to 5 s (`Thief (1981)`) | 0.5 s | 5.4 s | 5.4 s |
| Why | scan is synchronous-ish and immediate | `Media/Updated` is debounced by `ServerConfiguration.LibraryMonitorDelay` = **60 s** default | debounced by `LibraryMonitorDelaySeconds` = **90 s** default |

Both debounce values are settable via `GET/POST /System/Configuration`. The rest of the run uses 5 s. Jellyfin's timer appears to restart on each report (E04 took a full 60.5 s). Emby's E04 appeared after 14.9 s because the E03 report about 75 s earlier still had a pending 90 s timer, so Emby seems to batch reports under one timer without restarting it (inferred from one sample).

Push events seen: Plex `timeline` entries `{"type":1,"itemID":"18","state":0,"metadataState":"created"}` plus `activity` `library.update.section`; Jellyfin and Emby `LibraryChanged {"ItemsAdded":[…],"FoldersAddedTo":[…],"CollectionFolders":[…]}`.

## Scenario 2. New Sidecar beside an existing video

| | Plex | Jellyfin | Emby |
|---|---|---|---|
| Route A / video path | **A** `PUT /library/metadata/1/refresh` → 200. `Heat (1995) - 1080p.en.srt` appeared after 1.0 s. Same for Collateral (1.0 s). | **video path** `{"Path":"…/Collateral (2004) - 1080p.mkv","UpdateType":"Modified"}` → subtitle after 5.3 s (= debounce). | **video path** `Modified` → subtitle after 6.1 s. |
| Route B / sidecar path | **B** `GET /library/sections/1/refresh?path=/media/movies/Heat (1995)` → 200. `Heat (1995) - 4K HDR.fr.srt` appeared after 0.1 s. | **sidecar path** `{"Path":"…/Heat (1995) - 1080p.de.srt","UpdateType":"Created"}` → subtitle after 5.2 s. | **sidecar path** `Created` → subtitle after 6.1 s. |
| Which works | both | both | both |
| Binds to the right version only? | **No.** Every sidecar is attached to **every** Media/Part in the folder. `Heat (1995) - 1080p.en.srt` shows as an `eng` stream on the 1080p, the 4K and the Director's Cut parts (stream ids 81, 82, 83), and `Collateral (2004) - 1080p.en.srt` on both Collateral parts. | Yes. `Heat (1995) - 1080p.*.srt` only on the 1080p source, `Collateral (2004) - 1080p.*.srt` only on the 1080p source, nothing on 4K. | Yes. Same result as Jellyfin. |
| Side effects | | A report for one path refreshes the whole folder: `Heat (1995) - 4K HDR.fr.srt` (never reported to Jellyfin) was picked up by the `de.srt` report. | same folder-wide pickup |
| Fields | `Media[].Part[].Stream[]` with `streamType=3, id, key=/library/streams/{id}, codec=srt, languageCode=eng` | `MediaSources[].MediaStreams[]` with `Type=Subtitle, Index, IsExternal=true, Path, Language=eng, Codec=subrip` | same with `Language=en`, `Codec=srt` |

Evidence (Plex, after A and B; `probes/s2_sidecar.py plex`):

```
Heat (1995) - 4K HDR.mkv [(82,'eng'), (87,'fra')]
Heat (1995) {edition-Director's Cut} - 4K HDR.mkv [(83,'eng'), (88,'fra')]
Heat (1995) - 1080p.mkv [(81,'eng'), (86,'fra')]
```

Jellyfin note: the first Jellyfin attempt returned 401 because a parallel probe logged in with the same `DeviceId`, which revokes the earlier token. Probes now use one DeviceId per process.

## Scenario 3. Sidecar replaced under the same name

`en-v2.srt` (cue "Replaced cue") copied over `Heat (1995) - 1080p.en.srt` (size and mtime change).

| | Plex | Jellyfin | Emby |
|---|---|---|---|
| Download route | `GET /library/streams/81` | `GET /Videos/{itemId}/{mediaSourceId}/Subtitles/{index}/Stream.srt` (and `.vtt`) | same |
| Served content right after the replace, **before any route** | new cue | new cue (srt and converted vtt) | new cue (srt and converted vtt) |
| After the route (Plex A, Jellyfin/Emby video path `Modified`) | new cue, 0 s | new cue | new cue |
| Stream identity | stream id 81 unchanged across refreshes | `Index` changes when sidecars are added (en went 0 → 1 after `de.srt` arrived) | `Index` changes (en went 2 → 3) |

All three read the sidecar from disk on each request, so the route isn't needed for a same-name replace to be served. It is still needed for anything the server stores, such as language, codec or the stream list. Stream indexes in Jellyfin and Emby are not stable identifiers.

## Scenario 4. Removed file

Deleted `Heat (1995) - 4K HDR.mkv` and its `fr.srt`, and the whole `Ronin (1998)` folder.

| | Plex | Jellyfin | Emby |
|---|---|---|---|
| Route | `GET /library/sections/1/refresh?path=/media/movies/Heat (1995)` and `…?path=/media/movies/Ronin (1998)` (folder no longer exists) → 200 | `{"Path":"…/Heat (1995) - 4K HDR.mkv","UpdateType":"Deleted"}` and the Ronin file path → 204 | same → 204 |
| Version gone | 0.0 s (by the first poll). Heat keeps 2 Media. | 5.6 s | 5.6 s |
| Whole movie gone | 0.6 s. `GET /library/metadata/18` → 404, so it was purged rather than trashed (`autoEmptyTrash=true`). | 5.6 s | 5.6 s |
| Notes | Scanning a `path` that no longer exists is accepted and removes its items. With `autoEmptyTrash=false` items would be left as unavailable (not tested). | Reporting the path of a folder that no longer exists works. | same |

## Scenario 5. Two Targets' files as versions

Covered in scenario 0. Summary: Plex exposes versions as `Media[]` with no label. Jellyfin uses one item with `MediaSources[].Name` = label (`"1080p"`, `"4K HDR"`, `"Extended 1080p"`). Emby uses one item per file plus a merged `MediaSources[]` on detail and PlaybackInfo only. A `{edition-…}` file in the folder stops Jellyfin, and Emby on a fresh scan, from grouping any of the folder's files.

## Scenario 6. Video transcode, session list and push events

The client was emulated by starting HLS with a low bitrate/resolution, fetching the master playlist, the variant and the first segment. The playback report is a Plex `/:/timeline` or a Jellyfin/Emby `/Sessions/Playing` + `/Progress`.

| | Plex | Jellyfin | Emby |
|---|---|---|---|
| Start transcode | `GET /video/:/transcode/universal/start.m3u8?path=/library/metadata/15&mediaIndex=1&partIndex=0&protocol=hls&directPlay=0&directStream=0&videoResolution=640x360&maxVideoBitrate=400&session=conf-tc-1` with `X-Plex-Client-Identifier`, `X-Plex-Session-Identifier`, `X-Plex-Platform`, `X-Plex-Product` | `GET /Videos/{id}/master.m3u8?MediaSourceId=…&PlaySessionId=…&DeviceId=…&VideoCodec=h264&AudioCodec=aac&VideoBitrate=300000&MaxWidth=640&…` or server-decided: `POST /Items/{id}/PlaybackInfo` with a DeviceProfile (`MaxStreamingBitrate=400000`) → `MediaSources[].TranscodingUrl` | same two ways |
| Playback report needed to appear in the session list? | **Yes for `/status/sessions`** (size 0 without `/:/timeline`). `/transcode/sessions` lists it either way. | **No.** `TranscodingInfo` is on the device's session even without `/Sessions/Playing` (`NowPlayingItem` null). | **Yes.** Without `/Sessions/Playing` the session has no `TranscodingInfo`. |
| Field that says video is transcoded | `TranscodeSession.videoDecision = "transcode"` (also on each `Media[].Part[].Stream[].decision`). Direct stream has `videoDecision = "copy"`, **but `Part.decision` is still `"transcode"`**. Direct play has no `TranscodeSession` and `Part.decision = "directplay"`. The played version is the `Media` with `selected: true`. | `TranscodingInfo.IsVideoDirect = false` (null `TranscodingInfo` for direct play); `PlayState.PlayMethod = "Transcode"`; `TranscodingInfo.TranscodeReasons` filled only on the PlaybackInfo path (`["AudioCodecNotSupported","ContainerBitrateExceedsLimit"]`), `[]` when the client forces parameters; `HardwareAccelerationType` | `TranscodingInfo.IsVideoDirect = false`, `PlayMethod = "Transcode"`, `TranscodeReasons` (`["ContainerBitrateExceedsLimit"]` on the PlaybackInfo path, also in the `TranscodingUrl` query), plus `VideoEncoder` (`"x264"`), `VideoEncoderIsHardware`, `VideoDecoderIsHardware`, `VideoPipelineInfo[]` |
| After stop | `/status/sessions` size 0 after `state=stopped` + `/video/:/transcode/universal/stop` | **Stale**: when no Playing/Stopped report was sent, `TranscodingInfo` stayed on the session after `DELETE /Videos/ActiveEncodings`. With reports, it clears. | clears |
| Push channel | `ws://localhost:32400/:/websockets/notifications`, no auth, no subscription (unclaimed server) | `ws://localhost:8096/socket` with the `Authorization: MediaBrowser … Token=…` header. **`?api_key=` query gives 403.** Then send `{"MessageType":"SessionsStart","Data":"0,1500"}`. The server sends `ForceKeepAlive {"Data":60}` and expects `KeepAlive`. | `ws://localhost:8097/embywebsocket?api_key=…&deviceId=…` → 101, then `SessionsStart` |
| Push payloads | `transcodeSession.start` / `.update` / `.end` with `TranscodeSession[]{key:"/transcode/sessions/conf-tc-1", videoDecision, audioDecision, progress, speed, transcodeHwRequested, …}`; `playing` with `PlaySessionStateNotification[]{sessionKey, clientIdentifier, ratingKey, state, transcodeSession, viewOffset}`. The transcode events fire without any timeline. | `Sessions` with `Data[]` = full session objects (same shape as `GET /Sessions`, including `TranscodingInfo`), also `UserDataChanged`, `LibraryChanged`, `RefreshProgress` | same `Sessions` shape (keys include `InternalDeviceId`, `Protocol`), no `MessageId` |

Evidence (trimmed):

```
# Plex /transcode/sessions
{"key":"conf-tc-1","videoDecision":"transcode","audioDecision":"transcode","protocol":"hls","container":"mpegts",
 "sourceVideoCodec":"h264","videoCodec":"h264","transcodeHwRequested":false,"progress":98.3,"speed":13.0}
# Plex direct stream /status/sessions
TranscodeSession {"videoDecision":"copy","audioDecision":"copy",…}  Media 25 selected True Part.decision transcode
# Plex direct play
TranscodeSession None  Media 25 selected True Part.decision directplay
# Jellyfin GET /Sessions (PlaybackInfo path)
PlayState {"PlayMethod":"Transcode","MediaSourceId":"5a20…"}
TranscodingInfo {"IsVideoDirect":false,"IsAudioDirect":false,"VideoCodec":"h264","Width":960,"Height":540,
 "HardwareAccelerationType":"none","TranscodeReasons":["AudioCodecNotSupported","ContainerBitrateExceedsLimit"]}
# Emby GET /Sessions (PlaybackInfo path)
TranscodingInfo {"IsVideoDirect":false,"VideoEncoder":"x264","VideoEncoderIsHardware":false,
 "TranscodeReasons":["ContainerBitrateExceedsLimit"],"Width":426,"Height":240,…}
# Jellyfin ws
{"MessageType":"Sessions","MessageId":"…","Data":[{…,"DeviceId":"conformance-probe-76644","PlayState":{"PlayMethod":"Transcode"},
 "TranscodingInfo":{"IsVideoDirect":false,"TranscodeReasons":[],"Width":640,"Height":360}}]}
```

## Contract deltas

1. **Plex Sidecar pickup works both ways.** Item refresh `PUT /library/metadata/{rk}/refresh` (≈1 s) and a section scan with `path` set to the folder (≈0.1 s) both add the subtitle stream. The contract's item-refresh plus folder-scan fallback holds.
2. **Plex binds every Sidecar to every version in the folder.** `Heat (1995) - 1080p.en.srt` appears on the 1080p, 4K and Director's Cut parts. Version-specific Sidecars cannot be expressed on Plex. The Import pipeline's Sidecar-naming constraint (Jellyfin's full-file-name prefix) still holds for Jellyfin and Emby, but on Plex a 1080p-synced subtitle will be offered on the 4K file too. This matters if Targets differ in cut or timing.
3. **Emby Sidecar handling (previously unverified) matches Jellyfin.** Reporting either the video path (`Modified`) or the Sidecar's own path (`Created`) works. Binding is per version by full-file-name prefix. Jellyfin also accepts the Sidecar's own path, so "report the video's path" is sufficient but not required on either.
4. **`/Library/Media/Updated` is debounced.** Jellyfin waits `LibraryMonitorDelay` (60 s default) and Emby `LibraryMonitorDelaySeconds` (90 s default) before anything appears. "Fire and confirm acceptance" is right (204 in under 20 ms). But any refresh SLA or fake latency should model a 60/90 s delay, not seconds. The setting is server-wide and admin-tunable. A report refreshes the whole containing folder.
5. **`{edition-…}` in a file name breaks Jellyfin version grouping for the whole folder.** With the ADR 0011 layout, Jellyfin showed three separate Heat movies. It grouped as soon as the edition file was gone. Emby also did not group on a fresh scan, and later grouped the non-edition files only. This refines "Jellyfin groups movie versions only when every file starts with the folder name": the remainder after the folder name must also start with a separator such as ` - `, and `{` doesn't count (inferred from the observation plus the rule; the exact Jellyfin source check was not re-read). Issue #42 / ADR 0011 needs to know this.
6. **Plex editions not observed.** With no Plex Pass (`myPlexSubscription=false`), the `{edition-Director's Cut}` file became a third Media of the same Heat item, with no `editionTitle`. Whether Pass splits it was not observable without signing in to plex.tv.
7. **Plex exposes no version label.** `Media` has no `title` for `1080p` / `4K HDR`. Versions are told apart only by `videoResolution`/`width`/`height`/`bitrate` and `Part.file`. Jellyfin and Emby expose the label as `MediaSource.Name`.
8. **Emby models versions as separate items.** Each file has its own item id, grouped by `PresentationUniqueKey`. Only item detail and PlaybackInfo return the merged `MediaSources`, while listings return one source per item. "Open in Emby" lookups by path find the file's own item, which is fine. A fake must reproduce the split between listing and detail.
9. **Jellyfin alternate versions are absent from `/Items` listings.** A lookup by path must also scan sibling items' `MediaSources`.
10. **Sidecar replace needs no refresh for serving.** All three re-read the file on each subtitle request, srt and converted vtt alike. Jellyfin and Emby subtitle `Index` values shift when Sidecars are added. Don't persist them.
11. **Session detection specifics.**
   - Plex. Use `TranscodeSession.videoDecision == "transcode"`, not `Part.decision`, which is `"transcode"` even for a remux (`copy`).
   - Plex. `/status/sessions` needs a client timeline, while `/transcode/sessions` and the `transcodeSession.*` events don't. For "suspend Conversions while transcoding", `transcodeSession.start/end` plus a `/transcode/sessions` poll are the more direct signal.
   - Jellyfin. `TranscodingInfo` can go stale after the encode is killed when the client sent no stop. Check `PlayState`/`NowPlayingItem` too.
   - Emby. It only shows `TranscodingInfo` once the client reports playback.
   - Jellyfin and Emby. `TranscodeReasons` is empty when the client forced the parameters rather than letting PlaybackInfo decide, so the Playback report will sometimes have no reason.
12. **Jellyfin 12 websocket auth.** The `api_key` query param is refused with 403, and the `Authorization` header works. Emby still accepts `?api_key=`. Jellyfin sends `ForceKeepAlive` (60 s) and needs `KeepAlive` replies. The `Sessions` push payload is the full `GET /Sessions` object array on both.
13. **Jellyfin token revocation.** A new `AuthenticateByName` with the same `DeviceId` revokes the earlier token (401 on the old one). SlipStream must use one stable DeviceId per Media Server connection and never log in twice concurrently with it.

## Not observed

- Plex edition split under Plex Pass (server is unclaimed; signing in to plex.tv was out of scope).
- Plex behavior with `autoEmptyTrash=false` (left at default `true`).
- Hardware transcodes (no GPU in the containers). The `transcodeHwRequested` / `HardwareAccelerationType` / `VideoEncoderIsHardware` fields were observed as false/none only.
- Realtime monitoring on virtiofs (it was turned off by design, so whether inotify reaches the containers was not tested).
- Plex webhooks (need Pass; not used by the contract).
