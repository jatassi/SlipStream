# Plex playback recorded run, 2026-11-26 (S36 (Playback report), #161)

Recorded 2026-11-26 for S36 (Playback report), issue #161, step 1 of its delivery order. It records what a real Plex Media Server reports in `GET /status/sessions` while Plex Web plays, and answers the recorded-run questions for Plex.

**Result: every Plex capture rule in the spec holds on PMS 1.43.5.11079, with three findings the build must take in.** `TranscodeSession` has no `height` when the video is transcoded. During a remux or transcode, the session's `Media` and `Part` describe the delivered stream, not the file. A direct play can carry a `TranscodeSession` left over from an earlier play.

## Setup

- A throwaway `plexinc/pms-docker:latest` container (PMS 1.43.5.11079) on Colima on Jackson's Mac. It was claimed by throwaway account A through a plex.tv PIN that Jackson approved in a private window, because A's stored legacy token had been revoked. The container, its volumes and the server's device on the account were removed afterwards. The image digest is in `1.43.5.11079/image.txt`.
- One movie library with the `tv.plex.agents.none` agent over a read-only bind mount. It holds the media of the Jellyfin run of the same date: `Twoversion (2020)` in 2160p HEVC and 1080p H.264, `Remux (2020)` (H.264 and AAC in MKV), `Audiotranscode (2020)` (two DTS 5.1 tracks, English then French), `Burnin (2020)` (a PGS track) and `Direct (2020)` (H.264 and AAC in MP4), 180 s each. With no agent match, Plex kept the two Twoversion files apart, so they were merged into one item with `PUT /library/metadata/6/merge?ids=5`, as Plex Web's Merge does.
- The player is Plex Web 4.162.1, served by the PMS, in Playwright WebKit on macOS. Its sign-in token was put in Plex Web's `myPlexAccessToken` local storage. It used the movie page's Play Version dialog, audio and subtitle pickers and Play button, and the player's Settings > Quality > Show all list. Plex Web's "Sync your watch state" prompt was declined.
- `GET /status/sessions` every 2 s while each play runs, and `GET /library/metadata/{ratingKey}` before each play. The PMS restarted before each case, because Plex Web otherwise restores a paused player on load and its sessions linger. The player's `/video/:/transcode/universal/*` and `/library/parts/*` requests are logged with `"source":"player"`.

## Cases

| Case | Result |
|---|---|
| A. Two versions, 1080p picked | The session lists both Media. Only Media 5, the 1080p file, has `"selected":true`. `videoDecision` and `audioDecision` are `copy` and `Part.decision` is `transcode`: a remux of the MKV to DASH `mp4` |
| B. Direct at 2 Mbps | `videoDecision` and `audioDecision` `transcode`, `Part.decision` `transcode`. `TranscodeSession`: `mp4`, `h264`, `aac`, 2 channels, **no `height`**. The session's Media reads `height` 404 and `bitrate` 1459, which is the output. `Session.bandwidth` is 1532 |
| B. 2160p HEVC at full quality | `copy` and `copy`, `Part.decision` `transcode`. Safari takes HEVC, so only the container changed |
| C. Remux | `copy` and `copy`, `Part.decision` `transcode`, `TranscodeSession.container` `mp4`, `height` 1080 |
| D. Audio only, French track | `videoDecision` `copy`, `audioDecision` `transcode` (DTS, named `dca`, to AAC). The session's audio `Stream` is id 3 with `decision` `transcode`, which is the French stream in the item's metadata |
| E. Burn-in | `videoDecision` `transcode`, `audioDecision` `copy`, `subtitleDecision` `burn`. The session's PGS `Stream` id 6 reads `"decision":"burn"` and `"burn":"1"` |
| F. Identity | Direct play: `Part.decision` `directplay` and no `TranscodeSession` in a clean play. One `Player.machineIdentifier` over every case and across a rename of the browser's device on plex.tv. `Player.title` stayed `Safari` |

## Answers

**1. Decision words per play type.** All of these are from `TranscodeSession` and `Part.decision`.

| Play | `videoDecision` | `audioDecision` | `subtitleDecision` | `Part.decision` |
|---|---|---|---|---|
| Direct play | no `TranscodeSession` | | | `directplay` |
| Remux (container change, streams copied) | `copy` | `copy` | absent | `transcode` |
| Audio-only transcode | `copy` | `transcode` | absent | `transcode` |
| Video transcode (2 Mbps) | `transcode` | `transcode` | absent | `transcode` |
| Burned subtitles | `transcode` | `copy` | `burn` | `transcode` |

The spec's table holds: `copy` is direct, `burn` is burned, and absent is none. This client never produced `subtitleDecision` `copy` or `transcode`.

**2. `Part.decision` on a container change.** Holds. It is `transcode` on a remux (MKV to `mp4`) and on every transcode, and `directplay` on a direct play.

**3. `selected` and which Media the session lists.** For a two-version item, `/status/sessions` lists **all** of the item's Media. Only the playing one has `selected`: `[["6",null],["5",true]]`. JSON gives `true`. XML would give `selected="1"`. A one-version item lists its one Media, selected. The session names the version by Media id, so 5 maps to the 1080p file through `/library/metadata/6`.

**4. Received fields.** `TranscodeSession` carries `container`, `videoCodec`, `audioCodec` and `audioChannels` on every play that has one, and `sourceVideoCodec` and `sourceAudioCodec` too. It has **no `height` or `width` when the video is transcoded**, at 2 Mbps and in the burn. It has them only when the video is copied. The session's selected Media then describes the output: `height` 404, `bitrate` 1459, `videoResolution` `SD`. `Media.bitrate` is in kbps: the library reads 4135 for the 4.1 Mbps Direct file. `Session.bandwidth` (1532 at 2 Mbps, 10000000 on some copies) is a reservation, as the spec says, not the received bitrate. The spec's "`TranscodeSession` height" therefore fails on a video transcode. The selected session Media's `height` and `bitrate` give the received height and bitrate.

**5. Stream ids.** Every `Stream` id in a session is in `/library/metadata/{ratingKey}` for the same Media. The session gives them as strings (`"11"`) and the metadata as numbers (`11`), so compare them as strings. The session lists only the streams in use: video, the chosen audio and the chosen subtitle.

**6. Player identity.** `machineIdentifier` is the same over every play and across a rename of the browser's plex.tv device (`PUT /devices/{id}.xml?device[name]=`, status 200). Plex Web has no device name setting of its own, and `Player.title` stayed `Safari` after the rename. The player reads `platform` `Safari`, `product` `Plex Web`, `device` `OSX` and `model` `bundled`.

**7. Names.** Containers: `mkv` and `mp4` in the library, and `mp4` for every Plex Web transcode (DASH). Video codecs: `h264` and `hevc`. Audio codecs: `aac` and `dca` for DTS. Subtitles: `pgs`. Plex's per-kind table needs `dca` to DTS and `pgs` to PGS.

## Findings for the build

- **Read the received height and bitrate from the selected session Media during a transcode.** `TranscodeSession` drops `height` when it re-encodes video.
- **The session's Media and Part describe the delivered stream.** During a remux or transcode, the selected Media reads `container` `mp4` with no `Part.file`. The version lookup must map the Media id through `/library/metadata/{ratingKey}` to get the file, as the spec's version lookup does.
- **A direct play can carry a stale `TranscodeSession`.** In four invocations a direct play (`Part.decision` `directplay`) came with a `TranscodeSession` from an earlier play by the same player, once with `sourceVideoCodec` `hevc` and height 2160 while the player was playing the H.264 file. This was before the PMS restarts, or right after Plex Web restored an earlier queue on load. Trust `TranscodeSession` only when `Part.decision` is `transcode`.
- **Hardware** reads `transcodeHwRequested` false, with no `transcodeHwDecoding` or `transcodeHwEncoding`. No run had a GPU.

## Files

`1.43.5.11079/` beside this file holds:

- `exchanges.jsonl`, one line per call: invocation, t, case, step, source, method, path, query, request, status and response. Tokens are redacted. Account ids and names, avatar URLs, addresses, and the server's and players' machine identifiers are replaced by stable placeholders such as `<account-id-1>` and `<machine-id-1>`, so a repeated id still reads as one. Each run of the recorder is one invocation. Cases were run more than once while the player script was fixed, and `results.json` reads the last invocation of each case. Repeated range requests for one URL are kept once.
- `results.json`, per case what the spec predicts, what happened and pass or contradiction, and each answer above with its evidence.
- `image.txt`, the image digest.

The recorder (`plex-up.mjs`, `plex-merge.mjs`, `plex-play.mjs`, `plex-cases/cases.mjs`, `check-plex.mjs`, `export.mjs`) and the raw logs stay in the project folder.
