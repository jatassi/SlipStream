export interface DownloadclientsNodeTorrents {
  /** @minimum 0 */
  downloading: number;
  /** @minimum 0 */
  failures: number;
  nodeId: string;
  /** @minimum 0 */
  seeding: number;
  /**
   * Every Torrent it holds, paused, queued and complete ones too
   * @minimum 0
   */
  torrents: number;
  /** @minimum 0 */
  warnings: number;
}
