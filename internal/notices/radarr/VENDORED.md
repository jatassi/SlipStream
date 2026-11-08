# Ported: Radarr

Source: https://github.com/Radarr/Radarr
Commit: 928e8871ccad6a899e9d99781a54fcf17ab7c3d9
License: GPL-3.0 (`LICENSE` beside this file)

SlipStream's release parser (`internal/release`) and the movie Content Type's title rules (`internal/contenttypes/movie`) are Go ports of Radarr's C# parser, copyright the Radarr contributors and modified for SlipStream. Their regular expressions are Radarr's, copied as data into each package's `internal/upstream`, and each package's `upstream.json` maps its Go files to the Radarr files they port. The parser's golden tests read Radarr's parser test cases at the commit above, and the formats package ports Radarr's Custom Format conditions and enums.

`make arr-sync` writes this file. To change it, edit cmd/arr-sync/pins.json and run it again.
