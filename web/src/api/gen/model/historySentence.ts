export interface HistorySentence {
  /**
   * The failed Grabs that need attention, at most 100; 0 when the words count the week
   * @minimum 0
   * @maximum 100
   */
  needsYou: number;
  words: string;
}
