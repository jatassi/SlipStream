export interface MovesDropped {
  name: string;
  torrentId?: string;
  /** Why it stays */
  words: string;
  /** A Work unit's; the zero id for a torrent */
  workId: string;
}
