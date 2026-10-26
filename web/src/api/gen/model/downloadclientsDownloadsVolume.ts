export interface DownloadclientsDownloadsVolume {
  /**
   * Bytes
   * @minimum 0
   */
  free: number;
  /** As the Node names it, the same as Storage names the volume */
  mount: string;
  /** Such as D: (Media) */
  name: string;
  sampledAt: string;
  /**
   * Bytes
   * @minimum 0
   */
  total: number;
}
