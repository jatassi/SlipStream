export interface ConversionsCounts {
  /**
   * When the last unfinished Conversion is forecast to finish; null when none is forecast
   * @nullable
   */
  clearsAt: string | null;
  /** @minimum 0 */
  converting: number;
  /** @minimum 0 */
  failed: number;
  /** @minimum 0 */
  paused: number;
  /** @minimum 0 */
  waiting: number;
}
