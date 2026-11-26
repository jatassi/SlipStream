# Emby playback recorded run, 2026-11-26 (S36 (Playback report), #161)

Recorded 2026-11-26 for S36 (Playback report), issue #161, step 1 of its delivery order. It checks the rules in "What each kind captures" and "Conformance and recorded runs" against what a real Emby reports while its own web client plays, and answers the spec's two Emby questions.

**Result: the version, stream, reason and track rules hold on Emby 4.10.1.0 and 4.10.2.0 (the newest image), with four findings the build must take in.** Emby's `MediaSourceId` always differs from its item id. No session field names subtitle delivery. A direct play of the file reads `DirectStream`, and a remux reads `Transcode`. Sessions have no `DeviceType`. Both versions answered the same in every case.

## Setup

- Throwaway `emby/embyserver:4.10.1.0` and `:latest` (4.10.2.0) containers on Colima on Jackson's Mac, each with a fresh config volume, all removed afterwards. Image digests are in `<version>/image.txt`.
- Setup by API: the startup wizard with an admin user and a random password made for the run, then one movie library over a read-only bind mount, with realtime monitoring and internet providers off.
- The media are those of the Jellyfin run of the same date: `Twoversion (2020)` in 2160p HEVC and 1080p H.264, `Remux (2020)` (H.264 and AAC in MKV), `Audiotranscode (2020)` (two DTS 5.1 tracks, English then French), `Burnin (2020)` (a PGS image subtitle track) and `Direct (2020)` (H.264 and AAC in MP4), 180 s each.
- The player is Emby Web, served by the server, in Playwright WebKit on macOS. It signs in as the admin user, opens each movie, picks the version, audio or subtitle track in the details page's pickers, and presses Play. Quality is the app's "In network" video quality under Settings > Playback.
- `GET /Sessions` every 2 s while each play runs, about 12 polls per play. The item's `GET /Users/{u}/Items/{id}` and `POST /Items/{id}/PlaybackInfo` come before each play. The player's own `PlaybackInfo` answers and stream requests are logged with `"source":"player"`.

## Cases

| Case | 4.10.1.0 and 4.10.2.0 |
|---|---|
| A. Two versions, 1080p picked | The movie's card opens item 15 (the 2160p file), while `GET /Users/{u}/Items` lists item 14 (the 1080p file). `NowPlayingItem.Id` is 15, and `PlayState.MediaSourceId` is `mediasource_14`, whose path is the 1080p file. `Transcode`: video and audio direct into `ts`, `ContainerNotSupported` |
| B. Direct at 1.5 Mbps | `Transcode`: video transcoded to H.264 405p at 1,500,000 b/s, audio direct, `ContainerBitrateExceedsLimit` |
| B. 2160p HEVC at full quality | `Transcode`: video and audio direct, `ContainerNotSupported` |
| C. Remux | `Transcode`: video and audio direct, container `ts`, `ContainerNotSupported` |
| D. Audio only, French track | `Transcode`: video direct, audio DTS 5.1 to AAC 6 channels, but `TranscodeReasons` is only `ContainerNotSupported`. `AudioStreamIndex` 2 is the French track |
| E. Burn-in | `Transcode`: video transcoded, `SubtitleCodecNotSupported`, `SubtitleStreamIndex` 2. `VideoPipelineInfo` has a `SubtitleOverlay` step. The player's `PlaybackInfo` gives the stream `"DeliveryMethod":"Encode"`, and its request has `SubtitleMethod=Encode` |
| F. Identity | One `DeviceId` over every play and across a rename through `POST /Devices/Options`. `DeviceName` follows the rename. No `DeviceType`. Direct play reads `DirectStream` with no `TranscodingInfo`, and the player fetches `/emby/videos/11/original.mp4` |

## Predictions

| Prediction | Verdict | Body |
|---|---|---|
| `PlayState.MediaSourceId` names the version played | Holds | `"MediaSourceId":"mediasource_14"`, the source named `1080p` at `Twoversion (2020) - 1080p.mkv` |
| An alternate's `MediaSourceId` differs from the displayed item | Holds | Displayed and `NowPlayingItem.Id` 15, `MediaSourceId` `mediasource_14` |
| Whether Emby's `MediaSourceId` ever differs from its item id | **Always differs** | Every source id is `"mediasource_"` plus its item id, and each source carries `ItemId`: `mediasource_15` (`ItemId` 15), `mediasource_14` (`ItemId` 14). Even a one-version movie plays `mediasource_11` on item 11 |
| Only the primary item's `MediaSources` lists the alternate | **Contradicted** | Items 14 and 15 both list `["mediasource_15","mediasource_14"]` |
| `DirectStream` is a remux | **Contradicted** | The MKV remux reads `"PlayMethod":"Transcode"` with `"IsVideoDirect":true,"IsAudioDirect":true`. The MP4 played as the original file reads `"PlayMethod":"DirectStream"` with no `TranscodingInfo` |
| Container changed when `PlayMethod` is `DirectStream` or `Transcode` | Holds for `Transcode` only | With both streams direct, `Transcode` gives `remux`. On Emby `DirectStream` with no `TranscodingInfo` is a direct play, so the rule would call it a remux wrongly |
| `IsVideoDirect` and `IsAudioDirect` tell an audio-only transcode | Holds | `"IsVideoDirect":true,"IsAudioDirect":false`, though `TranscodeReasons` names no audio reason |
| Burned image subtitles give `SubtitleCodecNotSupported` | Holds | `"TranscodeReasons":["SubtitleCodecNotSupported"]`, `"IsVideoDirect":false` |
| The subtitle delivery field Emby.SDK's OpenAPI names | **No session field** | The server's own `/emby/openapi.json` (4.10.1.0) has no delivery field in `TranscodingInfo`, `Session.SessionInfo` or `PlayerStateInfo`. `SubtitleDeliveryMethod` (`Encode`, `Embed`, `External`, `Hls`, `VideoSideData`) is only `MediaStream.DeliveryMethod`, which the player's `PlaybackInfo` answer carries. The burn shows in the session as a `SubtitleOverlay` step in `TranscodingInfo.VideoPipelineInfo` |
| `TranscodingInfo` carries `Container`, `VideoCodec`, `Height`, `Bitrate`, `AudioCodec`, `AudioChannels` | Holds | `{"Container":"ts","VideoCodec":"h264","Height":405,"Bitrate":1500000,"AudioCodec":"aac","AudioChannels":2}` |
| `AudioStreamIndex` names the chosen track | Holds | 2, the `fre` DTS stream at `MediaStreams` index 2 |
| `DeviceId` is a stable player key | Holds | One id over every play and across a rename |
| `DeviceType` names the platform | **Contradicted** | The OpenAPI lists `DeviceType`, but Emby Web's sessions don't carry it. They carry `"DeviceName":"Safari macOS"` and `"Client":"Emby Web"` |

## What this means for the build

- **Jellyfin's version rule works on Emby.** Read the item's `MediaSources` and take the path of the source whose `Id` is `MediaSourceId`. Never compare `MediaSourceId` with an item id, because on Emby they never match. Any listed item of the group lists every version, so `NowPlayingItem.Id` is enough.
- **`DirectStream` doesn't mean a container change on Emby.** A direct play of the original file reads `DirectStream` with no `TranscodingInfo`. Decide from `TranscodingInfo`: none means `direct_play`, and both streams direct means `remux`.
- **Burned subtitles:** `SubtitleCodecNotSupported` with video not direct, Jellyfin's rule, is the session signal. The `SubtitleOverlay` pipeline step confirms it. No session field names delivery, so "the field Emby.SDK's OpenAPI names" has nothing to read.
- **Reasons can be incomplete.** An audio-only transcode named only `ContainerNotSupported`. The stream flags are right, so the report should not lean on the reasons alone.
- **Platform needs another field.** `DeviceType` is absent. `Client` or `DeviceName` are what the session gives.
- **Hardware** reads `VideoDecoderIsHardware` and `VideoEncoderIsHardware` false on every transcode. No run had a GPU.

## Files

`<version>/` beside this file holds, for 4.10.1.0 and 4.10.2.0:

- `exchanges.jsonl`, one line per call: invocation, t, case, step, source, method, path, query, request, status and response. The access token, `api_key` and password are redacted. User, device and server ids and the server name are replaced by stable placeholders such as `<user-id-1>` and `<device-id-1>`, so a repeated id still reads as one. Each run of the recorder is one invocation, and E and F were recorded a second time to capture the player's requests. Repeated range requests for one URL are kept once. Two `500 Unrecognized Guid format` lines in case A come from asking for an item by its source id, which Emby refuses.
- `results.json`, per case what the spec predicts, what happened and pass or contradiction, and every prediction with its verdict and evidence.
- `image.txt`, the image digest.

The recorder (`media.sh`, `pgs.py`, `mb-setup.mjs`, `mb-play.mjs`, `check-mb.mjs`, `export.mjs`) and the raw logs stay in the project folder.
