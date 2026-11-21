export interface TargetsDroppedLanguage {
  /**
   * Subtitles SlipStream downloaded that the save deletes. Uploads and subtitles found beside the files stay
   * @minimum 0
   */
  deleted: number;
  forced: boolean;
  language: string;
}
