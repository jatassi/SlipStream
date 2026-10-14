export interface GrabsProgress {
  /**
   * Bytes downloaded, or received and verified while transferring
   * @minimum 0
   */
  done: number;
  /**
   * Bytes wanted
   * @minimum 0
   */
  total: number;
}
