export interface FilesDolbyVision {
  /**
   * The base layer's signal: 1 HDR10, 2 SDR, 4 HLG, 6 Blu-ray HDR10, 0 none
   * @minimum 0
   */
  compatibility: number;
  /** @minimum 0 */
  level: number;
  /** @minimum 0 */
  profile: number;
}
