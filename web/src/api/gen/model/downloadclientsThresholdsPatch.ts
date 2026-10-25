export interface DownloadclientsThresholdsPatch {
  /**
   * Seconds; null resets it to its default
   * @nullable
   */
  faultStall?: number | null;
  /**
   * Seconds; null resets it to its default
   * @nullable
   */
  unregisteredGap?: number | null;
}
