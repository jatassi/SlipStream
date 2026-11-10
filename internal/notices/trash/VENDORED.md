# Ported: TRaSH Guides

Source: https://github.com/TRaSH-Guides/Guides
Commit: 98853590599f8583c19768fa469cfbf856de2045
License: MIT (`LICENSE` beside this file)

The release parser reads HDR, video codec, audio codec and channels from a release name with the conditions of TRaSH Guides' Custom Formats, copied as data into `internal/release/internal/upstream`. The formats tests run every TRaSH Custom Format from a snapshot in `internal/formats/testdata/trash`. The naming tests parse every TRaSH naming template from a snapshot in `internal/naming/testdata/trash`.

`make arr-sync` writes this file. To change it, edit cmd/arr-sync/pins.json and run it again.
