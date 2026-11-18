# Ported: alass

Source: https://github.com/kaegi/alass
Commit: 874f02d9577182752a0f969b6d6b98fd65bdf1fc
License: GPL-3.0-or-later (`LICENSE` beside this file; `alass-core/src/lib.rs` grants "version 3 of the License, or (at your option) any later version")

Timing sync's split-penalty search and its guard-band scoring (`internal/subtitles/timing/split.go`) port alass's split alignment and standard scoring (`alass-core/src/lib.rs`), copyright 2017 kaegi, by way of ffsubsync's port, and modified for SlipStream. Its target accuracy is alass's measured one from `README.md`.
