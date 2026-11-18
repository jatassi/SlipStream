export interface SubtitlesFit {
  /**
   * The timing match, in percent
   * @minimum 0
   * @maximum 100
   */
  match: number;
  offsetMs: number;
  /** The framerate change, 1 when none */
  ratio: number;
  /**
   * Stretches with one timing each, more than 1 where the timing jumps, such as at an ad break
   * @minimum 1
   */
  segments: number;
}
