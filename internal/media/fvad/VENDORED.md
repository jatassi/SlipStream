# Vendored: libfvad

Source: https://github.com/dpirch/libfvad
Commit: 532ab666c20d3cfda38bca63abbb0f152706c369
License: BSD-3-Clause (`LICENSE` beside this file), with WebRTC's patent grant (`PATENTS`)

libfvad is WebRTC's voice activity detector as a standalone C library, copyright the WebRTC project authors and Daniel Pirch. `libfvad/` holds upstream's `include/` and `src/` at the commit above, byte for byte, without its build files. Local changes: none.

cgo compiles only the C files in a Go package's own folder, so each `libfvad_*.c` beside this file is one line that includes one upstream source file. To update, copy `include/` and `src/` from the new commit over `libfvad/`, add or remove a `libfvad_*.c` line for each source file added or removed, change the commit above and `Version` in `errors.go`, and run the fvad tests with `-update` only if the golden maps change on purpose.
