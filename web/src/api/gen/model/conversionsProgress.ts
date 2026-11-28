export interface ConversionsProgress {
  /**
   * @minimum 0
   * @maximum 100
   */
  percent: number;
  /** @minimum 0 */
  secondsLeft: number;
  /**
   * Times real time
   * @minimum 0
   */
  speed: number;
}
