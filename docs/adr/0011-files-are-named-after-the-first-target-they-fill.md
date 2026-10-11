# Files are named after the first Target they fill

All of a Work's files share one folder, because Plex, Jellyfin and Emby group versions only when they sit together. So every file of one Item and Edition needs a name no other file there can take. A file **fills** a Target when it is the one file that Target holds for its Edition, and no two files fill the same Target. So the version label in each file name is the first Target in Profile order that the file fills, through a `{Target}` token, and names can't clash by construction. A non-Standard movie cut puts `{edition-<Cut>}` right after the dash, before the label, as `Heat (1995) - {edition-Director's Cut} 4K HDR.mkv`, whatever Media Servers are linked. Episodes carry the plain cut in the label, as `The Wire - S01E01 - Extended 1080p.mkv`. A template with renaming on can't be saved without `{Target}` and an Edition token, nor when a rendered Standard or non-Standard sample isn't the Work folder name (or `SxxEyy` for an episode) followed by ` - `. Decided in [Import pipeline under Editions and Targets](https://github.com/jatassi/SlipStream/issues/42#issuecomment-6104383476), Q2, Q3, Q5, Q9 and Q15, and [Edition file names every Media Server groups](https://github.com/jatassi/SlipStream/issues/85).

## Considered Options

- **List every Target the file fills, as `Heat (1995) - 1080p + 1080p small.mkv`.** Rejected. Names grow with every overlapping Target, and the file is renamed whenever any one of them moves.
- **Label from file facts, as Radarr's `{Quality Title}` and v1's slot naming do.** Rejected. Two files with the same facts, such as two 1080p WEB-DLs filling different Targets, get the same name and need a tiebreak. The version picker shows quality strings instead of the Targets the admin defined.
- **A separate library root per Target.** Rejected. Media Servers stop grouping the versions.
- **`{edition-<Cut>}` before the dash, as `Heat (1995) {edition-Director's Cut} - 4K HDR.mkv`.** Rejected. Jellyfin and Emby group a folder only when every file name is the folder name followed by ` - ` (or `_`, `.`, a `[…]` block), so one such file ungroups every version in the folder. Measured on Jellyfin 12.2 and Emby 4.10.
- **The plain cut in the label, as `Heat (1995) - Director's Cut 4K HDR.mkv`.** Rejected. Labels read cleanly on Jellyfin and Emby, but Plex loses the edition.
- **A folder per Edition.** Rejected. Jellyfin and Emby show one movie per Edition with clean labels, but Plex without Plex Pass merges the folders back into one item, and every move, rename and refresh gains a folder.
- **Render the plain cut when no Plex server is linked.** Rejected. Linking or unlinking Plex would rename every cut's files.

## Consequences

- A file is renamed when what it fills changes, so an import can rename files it did not bring. Its Sidecars rename with it, and each Media Server gets a refresh.
- Renaming a Target or reordering a Profile can rename thousands of files. Those renames wait for the admin to confirm them in the mass rename preview.
- Jellyfin and Emby show the tag in the version label, as `{edition-Director's Cut} 4K HDR`. An admin who wants clean labels and no Plex edition uses `{Edition}` instead of `{Edition Tag}`.
- Plex offers every Sidecar on every version of one Plex item, by Plex's design. Under Plex Pass each Edition is its own item, and Sidecars stay on their own version. Measured in the Media Server conformance suite, https://github.com/jatassi/SlipStream/pull/83 (README, Edition layouts).
