export interface DownloadclientsRetried {
  at: string;
  /**
   * The bytes its check found, absent until its Node checked; all of size means it seeds again
   * @minimum 0
   */
  found?: number;
}
