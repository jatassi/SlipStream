export interface DownloadclientsDiskGuard {
  /**
   * The volume's error level, in bytes
   * @minimum 0
   */
  errorBelow: number;
  /**
   * Bytes
   * @minimum 0
   */
  free: number;
  sampledAt: string;
  /** The downloads folder's volume, as the Node names it */
  volume: string;
}
