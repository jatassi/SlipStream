# A Move deletes its source only once each Media Server that showed it is safe, within a bound

This amends ADR 0016 and ADR 0024 where building Moves showed them silent on Media Servers. A Work Move's switch still commits in one Write, and the Work plays from the source until then. Jellyfin and Emby snapshot each user's Watch data before the switch, while the source still exists, and the switch waits for them. After the switch, the source's files are deleted once every Plex server that showed the Work has found the item at each new path, or once "Delete the old files after Plex finds the new ones, or at most after" (30 minutes by default) has passed since the switch. SlipStream never refreshes a vacated Plex folder before Plex has confirmed the new item. Decided in [Moves](https://github.com/jatassi/SlipStream/issues/173), building on [Moving torrents and library files between Nodes](https://github.com/jatassi/SlipStream/issues/90).

What it amends in ADR 0016:

- **"No hold" stays true for undo, and only for undo.** A Move that is canceled before its switch removes what it placed and keeps the source. A Move that has switched is undone by a Move back. The source's delete after the switch is held for Plex, within the bound, and never for undo.
- **A Move is one Carry.** The moved change carries a source root, so a server that shows both roots gets one Carry from the source path to the destination path, and a server that shows only the destination gets an add.
- **Jellyfin and Emby snapshot before the switch.** The switch waits until every Jellyfin and Emby server that shows the source root has written its snapshot. "Switch now" passes that wait and names each server whose Watch data and collections will be lost. Their Carry then waits until the source is gone, so a report of the old path finds no file there.

What it amends in ADR 0024:

- **Plex writes collection membership back when a Move changes the item.** A Move between two Plex sections finds or creates each snapshot collection by title in the destination section and tags the item into it. ADR 0024 rejected writing collections back on Plex because a rename within a section carries them. A Move doesn't: the recorded run of 2026-11-23 (`conformance/plex/recorded/2026-11-23-plex-moves.md`) measured that a copy in another section is a new item whose watch state and rating follow its guid at once, while collections never follow a new ratingKey. A collection write adds titles and never removes one, so sending the item's own titles with the snapshot's is safe.
- **A same-section Plex Move refreshes the whole section after the delete, never the vacated folder.** In its own section a copy joins the item as a second Part. A path refresh of the vacated folder then destroyed the whole item with its surviving Part, and Plex made it again a minute later under a new ratingKey without collections. A whole-section refresh kept the item, its ratingKey and its collections. Across sections a path refresh of the vacated folder removes only the old item.
- **A Plex confirm on a Move never passes on its timeout.** It refreshes the destination folders again, and only the bound lets the source go without it.

## Considered Options

- **Delete the source as soon as the switch commits, as ADR 0016 said.** Rejected. Plex would scan a vacated folder before finding the new one and drop the item with its Watch data.
- **Hold the source until every Plex server confirms, with no bound.** Rejected. A Plex server that is off for a week would keep a full disk full for a week, and most Moves exist to free one.
- **Snapshot Jellyfin and Emby after the switch.** Rejected. Their snapshot reads the item at the source path, which the switch makes stale.

## Consequences

- A Plex server that hasn't found the new item by the bound loses that item's Watch data, and the unit notes the server by name.
- A server whose old path still has its file at the bound shows both items until cleanup runs, and its Carry lists that.
- A Move copies at Low priority and holds its pair of Nodes for one Work's copy at a time. A 60 GB Work on a slow link can hold a new episode's Import on that pair for that long.
- A torrent Move adds the destination in seed mode and trusts the source Worker's bytes. libtorrent checks each piece the first time a peer asks for it, so a bad piece is caught then.
- The torrent handoff's destination pauses itself when its local window ends with the ask still open. The window always ends before the Controller's rollback, and `Timings.moves`'s Check refuses an override that would reverse that order.
- Every Node time that reaches the Controller is converted to the Controller's clock when it arrives, so a Worker's clock that runs minutes ahead or behind moves no guard.
