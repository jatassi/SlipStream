# Moves switch a whole Work over and hand torrents off one side at a time

An admin relocates files with a Move: one Work to another library root, or one torrent to another Download Client. A bulk selection, a whole root or a drain makes one Move per Work and per torrent. Files that cross Nodes go as Transfers, and the destination stages a Work's files until every one of them is verified. The Work then switches roots in one step, so it never sits in two folders, and it plays from the source until then. A torrent's whole payload is copied while the source keeps seeding. The destination adds it paused and complete. Once the source's last announce is at least 5 minutes old, the source sends `stopped` and the destination sends `started`. If any tracker rejects the destination, the source resumes. The source is deleted as soon as the switch commits, with no hold. Decided in [Moving torrents and library files between Nodes](https://github.com/jatassi/SlipStream/issues/90), Q2, Q5, Q8 and Q9.

## Considered Options

- **Switch a Work over Item by Item.** Rejected by the maintainer. A Work would sit in two roots for hours. Everything that asks for a Work's folder would need two answers, and Media Servers would have to merge one series across two paths.
- **Seed from both Nodes, then stop the source.** Rejected. Stock UNIT3D marks the user's history inactive on the source's `stopped`, and the destination's next announce is then held back by the min interval for about 25 to 28 minutes. Seed time counts per user and torrent, so a sequential handoff loses only seconds.
- **Hand off with no 5-minute guard.** Rejected. UNIT3D keeps a peer's min-interval timer when it stops within 5 minutes of its last announce, so the source couldn't resume if the destination were rejected.
- **Hold the source for a while so undo is instant.** Rejected. Most Moves exist to free a full disk. A held copy goes stale on the next upgrade, and a copy left in a Media Server's folder brings the old item back. Undo is a Move back.
- **Move a whole library root as one unit.** Rejected. A failure would block the whole root, and both copies would stay on disk until the last byte. When a disk was carried to another machine, the admin edits the root's Node and path instead, and nothing is copied.

## Consequences

- A Work's folder is stored relative to its root's id, so re-homing a root is a one-row edit. [Config backup and restore](https://github.com/jatassi/SlipStream/issues/91) can use the same step to re-home roots after a restore.
- The Media Server contract gains a "moved" change. Plex adds the new path first. Jellyfin and Emby remove the old path first, and SlipStream snapshots and restores their per-user watch data.
- Seed time, upload and download belong to the torrent in the Controller's intent, so a hit and run countdown survives a Move.
- How each server handles a path change is unverified until [How Media Servers handle a file moving to a new path](https://github.com/jatassi/SlipStream/issues/101) runs it in the conformance suite.
