export interface MediaserversMappingInput {
  rootId: string;
  /**
   * Required when shown: the root's folder as the server sees it, such as /data/movies or D:\Movies
   * @maxLength 4096
   */
  serverFolder?: string;
  shown: boolean;
}
