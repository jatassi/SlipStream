export interface DownloadclientsOwes {
  /**
   * Upload still needed, 0 when the rule doesn't count ratio or it is reached
   * @minimum 0
   */
  bytes: number;
  /**
   * Seeding time still needed, 0 when the rule doesn't count it or it is reached
   * @minimum 0
   */
  seconds: number;
}
