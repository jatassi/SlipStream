export interface MediaserversReceivedView {
  /**
   * null when unreported
   * @nullable
   */
  audioCodec: string | null;
  /**
   * null when unreported, in bits a second
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
