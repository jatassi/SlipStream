export interface SubtitlesLastSearch {
  at: string;
  /**
   * The best timing match among them, in percent
   * @nullable
   */
  bestMatch: number | null;
  /** @minimum 0 */
  found: number;
}
