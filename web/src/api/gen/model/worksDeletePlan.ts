export interface WorksDeletePlan {
  /**
   * The size of their library files
   * @minimum 0
   */
  bytes: number;
  /**
   * Their unfinished downloads, whose torrents keep downloading and seeding in the client
   * @minimum 0
   */
  downloading: number;
  /**
   * Their library files, Kept files included
   * @minimum 0
   */
  files: number;
  /**
   * Their Kept files
   * @minimum 0
   */
  kept: number;
  /**
   * The Works it removes
   * @minimum 0
   */
  works: number;
}
