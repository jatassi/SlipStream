export interface TransfersDetails {
  /**
   * Connections that carry its chunks now
   * @minimum 0
   */
  lanes: number;
  /**
   * Chunks asked for again since the receiving Node last started it
   * @minimum 0
   */
  retried: number;
  /**
   * The lanes' round trip
   * @minimum 0
   */
  rttMs: number;
}
