export interface DownloadclientsDaemon {
  /**
   * @minLength 1
   * @maxLength 253
   */
  host: string;
  /**
   * @minimum 1
   * @maximum 65535
   */
  port: number;
}
