export interface DownloadclientsFileDetail {
  /**
   * Bytes done, absent while unknown
   * @minimum 0
   */
  done?: number;
  /** @minimum 0 */
  index: number;
  /** The Items it filled once a Grab imported it, in order; empty before */
  items: string[];
  /** Under the save folder, with / between folders */
  path: string;
  /** @minimum 0 */
  size: number;
}
