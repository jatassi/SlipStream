export interface MediaserversReceivedView {
  /**
   * null when unreported
   * @nullable
   */
  audioCodec: string | null;
  /**
   * Bits a second; null when unreported
   * @nullable
   */
  bitrate: number | null;
  /**
   * null when unreported
   * @nullable
   */
  channels: number | null;
  /**
   * null when unreported
   * @nullable
   */
  container: string | null;
  /**
   * null when unreported
   * @nullable
   */
  height: number | null;
  /**
   * null when unreported
   * @nullable
   */
  videoCodec: string | null;
}
