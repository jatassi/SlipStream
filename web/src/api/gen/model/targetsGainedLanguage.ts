export interface TargetsGainedLanguage {
  forced: boolean;
  language: string;
  /**
   * Of those, files whose audio is scanned first
   * @minimum 0
   */
  needScan: number;
  /**
   * Files SlipStream searches subtitles for
   * @minimum 0
   */
  searches: number;
}
