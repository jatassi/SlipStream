export interface DownloadclientsClientFolder {
  /**
   * The folder as the client reports it, such as /downloads
   * @minLength 1
   * @maxLength 4096
   */
  client: string;
  /**
   * The same folder on the client's Node, such as D:\Torrents
   * @minLength 1
   * @maxLength 4096
   */
  node: string;
}
