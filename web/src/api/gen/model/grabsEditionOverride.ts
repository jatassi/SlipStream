export interface GrabsEditionOverride {
  /**
   * Send it back as the pick's edition
   * @minLength 1
   */
  edition: string;
  /** Such as Its name gives the Extended Edition, which the Work doesn't want. It's grabbed as the Standard Edition. */
  words: string;
}
