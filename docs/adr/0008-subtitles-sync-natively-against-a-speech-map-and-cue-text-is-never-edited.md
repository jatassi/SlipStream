# Subtitles sync natively against a Speech Map, and cue text is never edited

The Node that imports a library file builds that file's Speech Map during import, using the bundled jellyfin-ffmpeg (ADR 0007) and libfvad. The Subtitles Node then times each candidate subtitle against the Speech Map, so it never needs the video. Timing sync is SlipStream's own code. It runs an FFT offset search, a framerate ratio search and a split-penalty alignment, ported from ffsubsync's MIT code and the alass thesis. SlipStream changes only a subtitle's timing, text encoding and format, and never edits, adds or removes cue text. That rule keeps every Subtitle Provider's terms satisfied, and it keeps OpenSubtitles.com out until its files can come without injected ads. Decided in [Subtitle pipeline design](https://github.com/jatassi/SlipStream/issues/20), Q7, Q13, Q14, Q22 and Q23. Research in [Subtitle providers and timing sync](https://github.com/jatassi/SlipStream/issues/19).

## Considered Options

- **ffsubsync as a subprocess.** Rejected. It puts a Python runtime in every package and Docker image, and its optional auditok detector is GPL-3.0.
- **alass as a subprocess.** Rejected. It is GPL-3.0, and it has had no release since 2019.
- **Sync on whichever Node holds the Subtitles Role, fetching the video by Transfer.** Rejected. It copies whole files to align a subtitle. Requiring Subtitles to sit with Storage was rejected too, because it ties two Roles together.
- **OpenSubtitles.com on a free app key with the ad cue stripped.** Rejected. Stripping edits cue text, and Bazarr already refused it. A paid plan was rejected because its daily download cap is probably shared by every install (inferred).

## Consequences

- Import takes a little longer for files whose Profile wants subtitles. A Profile that gains a language makes the storage Node build the missing Speech Maps.
- A subtitle that fails the sync quality gate is dropped, never kept untimed. A language can stay missing even when providers have candidates.
- The largest subtitle catalog is missing at launch. No outreach is planned. [OpenSubtitles.com without ads](https://github.com/jatassi/SlipStream/issues/68) closed as not planned, and reopens if OpenSubtitles publishes an ad-free tier for free apps.
- The offline suite needs test media with known speech timings, plus subtitles shifted by known offsets and framerate ratios.
