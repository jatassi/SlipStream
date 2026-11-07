# Ported: Prowlarr

Source: https://github.com/Prowlarr/Prowlarr
Commit: ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9
License: GPL-3.0 (`LICENSE` beside this file)

SlipStream's ported site indexers (`internal/ports/`), their shared parsing (`internal/prowlarr/`) and the Cardigann runner (`internal/cardigann/`) are Go rewrites of Prowlarr's C#, copyright the Prowlarr contributors and modified for SlipStream. The commit above is the one the port list was measured at. Each Go file a port, a family or an `internal/prowlarr` package maps opens with a header naming its Prowlarr files and the commit it was last synced to, and the unit's `upstream.json` records the same.

The release notices read the license and this commit from here. Never port code from Jackett, which is GPL-2.0-only.
