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
  /** What explains the sentence: the volume, its free space, its error level and what to do */
  hint: string;
  sampledAt: string;
  /** What the guard says, as Health says it */
  sentence: string;
  /** The downloads folder's volume, as the Node names it */
  volume: string;
}
