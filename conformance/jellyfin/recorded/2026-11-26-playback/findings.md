# Jellyfin playback recorded run, 2026-11-26 (S36 (Playback report), #161)

Recorded 2026-11-26 for S36 (Playback report), issue #161, step 1 of its delivery order. It checks the rules in "What each kind captures" and "Conformance and recorded runs" against what a real Jellyfin reports while its own web client plays.

**Result: the version, stream, reason and track rules hold on Jellyfin 12.2.0 and 12.4.0, and three predictions are contradicted.** A remux never reads `DirectStream`, the alternate version's own item lists both versions, and sessions have no `DeviceType`. Both versions answered the same in every case.

## Setup

- Throwaway `jellyfin/jellyfin:12.2` and `:12.4` containers on Colima on Jackson's Mac, each with fresh config and cache volumes, all removed afterwards. Image digests are in `<version>/image.txt`.
- Setup by API: the startup wizard with an admin user and a random password made for the run, then one movie library over a read-only bind mount, with realtime monitoring and internet providers off.
- Media made with ffmpeg, 180 s each:
  - `Twoversion (2020)`, the 2160p (HEVC 10-bit, AAC stereo) and 1080p (H.264, AAC stereo) MKVs. Jellyfin groups them into one item.
  - `Remux (2020)`, H.264 and AAC in MKV.
  - `Audiotranscode (2020)`, H.264 with two DTS 5.1 tracks, English then French, in MKV. A first try with AC-3 played its audio direct, since Safari's profile takes AC-3.
  - `Burnin (2020)`, H.264 and AAC in MKV with one PGS image subtitle track.
  - `Direct (2020)`, H.264 and AAC in MP4.
- The player is Jellyfin Web, served by the server, in Playwright WebKit on macOS. It signs in as the admin user and uses the details page's version, audio and subtitle pickers and the Play button. Quality is the user's "In network" setting under Playback.
- `GET /Sessions` every 2 s while each play runs, about 12 polls per play. The item's `GET /Users/{u}/Items/{id}` and `POST /Items/{id}/PlaybackInfo` come before each play, and the player's own stream requests are logged with `"source":"player"`.

## Cases

| Case | 12.2.0 and 12.4.0 |
|---|---|
| A. Two versions, 1080p picked | `NowPlayingItem.Id` is the 1080p item, not the displayed 2160p one. `PlayState.MediaSourceId` is the 1080p source, and its path is the 1080p file. `Transcode`: video and audio direct, `ContainerNotSupported` |
| B. Direct at 1.5 Mbps | `Transcode`: video transcoded to H.264 720p at 1,628,002 b/s, audio direct, `ContainerBitrateExceedsLimit` |
| B. 2160p HEVC at full quality | `Transcode`: video and audio direct, container `mp4`, `ContainerNotSupported` and `VideoCodecTagNotSupported` |
| C. Remux | `Transcode`, never `DirectStream`: video and audio direct, container `mp4`, `ContainerNotSupported` |
| D. Audio only, French track | `Transcode`: video direct, audio DTS 5.1 to AAC 6 channels, `AudioCodecNotSupported`. `AudioStreamIndex` 2 is the French track |
| E. Burn-in | `Transcode`: video transcoded, `SubtitleCodecNotSupported`, `SubtitleStreamIndex` 2. The player asked for `SubtitleMethod=Encode` |
| F. Identity | One `DeviceId` over every play and across a rename through `POST /Devices/Options`. `DeviceName` follows the rename. No `DeviceType`. Direct play is `DirectPlay` from `stream.mp4?Static=true` |

## Predictions

| Prediction | Verdict | Body |
|---|---|---|
| `PlayState.MediaSourceId` names the version played | Holds | `"MediaSourceId":"2161007640e112b99e7d94026e47c250"`, the source named `1080p` at `Twoversion (2020) - 1080p.mkv` |
| An alternate's `MediaSourceId` differs from the displayed item | Holds | Displayed `66e05312…`, `MediaSourceId` and `NowPlayingItem.Id` both `2161007640…` |
| Only the primary item's `MediaSources` lists the alternate | **Contradicted** | `GET /Users/{u}/Items/2161007640…` lists both: 1080p first, then 2160p. The primary lists 2160p first. Either item resolves the source |
| The primary's source id equals its item id | Holds | Item `66e05312…`, first source `66e05312…` |
| `DirectStream` is a remux | **Contradicted** | The MKV remux reads `"PlayMethod":"Transcode"` with `"IsVideoDirect":true,"IsAudioDirect":true`. This client never reported `DirectStream`, because it plays every container change through HLS |
| Container changed when `PlayMethod` is `DirectStream` or `Transcode` | Holds | With both streams direct, the spec's rules give `remux` |
| `IsVideoDirect` and `IsAudioDirect` tell an audio-only transcode | Holds | `"IsVideoDirect":true,"IsAudioDirect":false` |
| Burned image subtitles give `SubtitleCodecNotSupported` | Holds | `"TranscodeReasons":["ContainerNotSupported","SubtitleCodecNotSupported"]`, `"IsVideoDirect":false` |
| `TranscodingInfo` carries `Container`, `VideoCodec`, `Height`, `Bitrate`, `AudioCodec`, `AudioChannels` | Holds | `{"Container":"mp4","VideoCodec":"h264","Height":720,"Bitrate":1628002,"AudioCodec":"aac","AudioChannels":2}` |
| `AudioStreamIndex` names the chosen track | Holds | 2, the `fra` DTS stream at `MediaStreams` index 2 |
| `DeviceId` is a stable player key | Holds | One id over every play and across a rename |
| `DeviceType` names the platform | **Contradicted** | Jellyfin sessions have no `DeviceType`. They carry `"DeviceName":"Safari"` and `"Client":"Jellyfin Web"` |

## What this means for the build

- **Read the version from `MediaSourceId`** as the spec says. The session's `NowPlayingItem.Id` is the alternate's own item, and that item lists every version too, so the lookup can read `NowPlayingItem.Id`'s sources. It need not find the primary.
- **Decide the container change from the streams, not from `DirectStream`.** A browser remux reads `Transcode` with both streams direct. The spec's rule gives `remux` for it. A test or fake that expects `DirectStream` for a remux doesn't match this client, and the fake should play `Transcode` with both direct for a browser.
- **Platform needs another field.** Jellyfin has no `DeviceType`. `Client` (`Jellyfin Web`) or `DeviceName` (`Safari`) are what the session gives.
- **Hardware** reads `"HardwareAccelerationType":"none"` on every transcode. No run had a GPU.

## Files

`<version>/` beside this file holds, for 12.2.0 and 12.4.0:

- `exchanges.jsonl`, one line per call: invocation, t, case, step, source, method, path, query, request, status and response. The access token and password are redacted, and the `Authorization` header is not recorded. User, device and server ids and the server name are replaced by stable placeholders such as `<user-id-1>` and `<device-id-1>`, so a repeated id still reads as one. Each run of the recorder is one invocation, and E and F were recorded a second time to capture the player's requests. Repeated range requests for one URL are kept once.
- `results.json`, per case what the spec predicts, what happened and pass or contradiction, and every prediction as holds or contradicted with its evidence.
- `image.txt`, the image digest.

The recorder (`media.sh`, `pgs.py`, `mb-setup.mjs`, `mb-play.mjs`, `check-mb.mjs`, `export.mjs`) and the raw logs stay in the project folder.
