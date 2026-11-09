export interface FilesWorkLine {
  /**
   * Files that can't be renamed now
   * @minimum 0
   */
  blocked: number;
  contentType: string;
  /** A hash of the Work's rows, which a confirm sends back */
  digest: string;
  /**
   * Files the disk queue still moves, which the plan leaves out
   * @minimum 0
   */
  moving: number;
  /**
   * Files that can be renamed now
   * @minimum 0
   */
  rename: number;
  title: string;
  workId: string;
}
