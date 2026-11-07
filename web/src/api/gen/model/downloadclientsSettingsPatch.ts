export interface DownloadclientsSettingsPatch {
  /**
   * null resets it to its default
   * @nullable
   */
  activeChecks?: number | null;
  /**
   * 0 for no limit; null resets it to its default
   * @nullable
   */
  activeDownloads?: number | null;
  /**
   * null resets it to its default
   * @nullable
   */
  connections?: number | null;
  /**
   * Whether the client runs the DHT, which public torrents use to find peers; null resets it to its default
   * @nullable
   */
  dht?: boolean | null;
  /**
   * Bytes a second, 0 for no limit; null resets it to its default
   * @nullable
   */
  downloadLimit?: number | null;
  /**
   * One of the Node's on external clients, which Grabs for Works whose library root is on the Node go to; null for its built-in client
   * @nullable
   */
  grabsTo?: string | null;
  /**
   * Seconds; null resets it to its default
   * @nullable
   */
  stalledAfter?: number | null;
  /**
   * Bytes a second, 0 for no limit; null resets it to its default
   * @nullable
   */
  uploadLimit?: number | null;
  /**
   * null resets it to its default
   * @nullable
   */
  utp?: boolean | null;
}
