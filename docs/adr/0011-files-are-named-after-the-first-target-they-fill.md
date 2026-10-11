# Files are named after the first Target they fill

All of a Work's files share one folder, because Plex, Jellyfin and Emby group versions only when they sit together. So every file of one Item and Edition needs a name no other file there can take. A file **fills** a Target when it is the one file that Target holds for its Edition, and no two files fill the same Target. So the version label in each file name is the first Target in Profile order that the file fills, through a `{Target}` token, and names can't clash by construction. A non-Standard cut adds `{edition-<Cut>}` before the label for Plex. A template with renaming on can't be saved without `{Target}` and an Edition token. Decided in [Import pipeline under Editions and Targets](https://github.com/jatassi/SlipStream/issues/42#issuecomment-6104383476), Q2, Q3, Q5, Q9 and Q15.

## Considered Options

- **List every Target the file fills, as `Heat (1995) - 1080p + 1080p small.mkv`.** Rejected. Names grow with every overlapping Target, and the file is renamed whenever any one of them moves.
- **Label from file facts, as Radarr's `{Quality Title}` and v1's slot naming do.** Rejected. Two files with the same facts, such as two 1080p WEB-DLs filling different Targets, get the same name and need a tiebreak. The version picker shows quality strings instead of the Targets the admin defined.
- **A separate library root per Target.** Rejected. Media Servers stop grouping the versions.

## Consequences

- A file is renamed when what it fills changes, so an import can rename files it did not bring. Its Sidecars rename with it, and each Media Server gets a refresh.
- Renaming a Target or reordering a Profile can rename thousands of files. Those renames wait for the admin to confirm them in the mass rename preview.
- Whether Jellyfin still groups names carrying `{edition-…}` is unverified. The Media Server conformance suite checks it.
