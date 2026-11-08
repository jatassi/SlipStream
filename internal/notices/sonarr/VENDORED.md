# Ported: Sonarr

Source: https://github.com/Sonarr/Sonarr
Commit: c018578e4b79ee048575c02d7a8aa2b929e4b837
License: GPL-3.0 (`LICENSE` beside this file)

SlipStream's release parser (`internal/release`) and the series Content Type's title and numbering rules (`internal/contenttypes/series`) are Go ports of Sonarr's C# parser, copyright the Sonarr contributors and modified for SlipStream. Their regular expressions are Sonarr's, copied as data into each package's `internal/upstream`, and each package's `upstream.json` maps its Go files to the Sonarr files they port. The parser's golden tests read Sonarr's parser test cases at the commit above, and the formats package ports Sonarr's Custom Format conditions and enums.

`make arr-sync` writes this file. To change it, edit cmd/arr-sync/pins.json and run it again.
