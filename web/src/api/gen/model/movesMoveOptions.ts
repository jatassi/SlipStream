export interface MovesMoveOptions {
  /** Delete each root the Move empties, once every unit ends */
  deleteEmptiedRoots?: boolean;
  /** Also move the seeding torrent a Work's files are linked to */
  includeLinkedTorrents?: boolean;
  /** Have each Tracker's new Grabs go to the client its torrents move to */
  repointRouting?: boolean;
}
