export interface GrabsProgress {
  /**
   * Bytes downloaded
   * @minimum 0
   */
  done: number;
  /**
   * Bytes wanted
   * @minimum 0
   */
  total: number;
}
