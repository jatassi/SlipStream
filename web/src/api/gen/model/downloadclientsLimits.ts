export interface DownloadclientsLimits {
  /** @minimum 1 */
  activeChecks: number;
  /**
   * 0 for no limit
   * @minimum 0
   */
  activeDownloads: number;
  /** @minimum 1 */
  connections: number;
  /**
   * Bytes a second, 0 for no limit
   * @minimum 0
   */
  downloadLimit: number;
  /** @minimum 1 */
  stalledAfterSeconds: number;
  /**
   * Bytes a second, 0 for no limit
   * @minimum 0
   */
  uploadLimit: number;
  utp: boolean;
}
