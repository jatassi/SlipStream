export interface TrackersStats {
  /**
   * Bytes, counting each torrent once, live or removed
   * @minimum 0
   */
  downloaded: number;
  /**
   * Its Grabs in the last 30 days
   * @minimum 0
   */
  grabs30d: number;
  /**
   * Its torrents that still owe it seeding
   * @minimum 0
   */
  owing: number;
  /**
   * Uploaded over downloaded, or over size where a torrent's size is larger
   * @minimum 0
   */
  ratio: number;
  /**
   * The size of its torrents that are seeding
   * @minimum 0
   */
  seedingBytes: number;
  /**
   * Its torrents, live or being removed
   * @minimum 0
   */
  torrents: number;
  /**
   * Bytes, counting each torrent once, live or removed
   * @minimum 0
   */
  uploaded: number;
}
