export interface DownloadclientsTotals {
  /** @minimum 0 */
  activeSeconds: number;
  /**
   * Bytes
   * @minimum 0
   */
  downloaded: number;
  /** @minimum 0 */
  seedingSeconds: number;
  /**
   * Bytes
   * @minimum 0
   */
  uploaded: number;
}
