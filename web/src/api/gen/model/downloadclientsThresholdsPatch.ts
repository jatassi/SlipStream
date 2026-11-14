export interface DownloadclientsThresholdsPatch {
  /**
   * Seconds an external client reports an error on a Torrent before its Grab fails; null resets it to its default
   * @nullable
   */
  clientErrorFail?: number | null;
  /**
   * Seconds; null resets it to its default
   * @nullable
   */
  faultStall?: number | null;
  /**
   * Seconds a magnet link's metadata search runs before its Grab fails; null resets it to its default
   * @nullable
   */
  metadataWait?: number | null;
  /**
   * Seconds; null resets it to its default
   * @nullable
   */
  unregisteredGap?: number | null;
}
