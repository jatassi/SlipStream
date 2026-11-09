import type { FilesWorkLine } from "./filesWorkLine.ts";

export interface FilesRenamePlan {
  /**
   * Files that can't be renamed now, in every Work
   * @minimum 0
   */
  blocked: number;
  /**
   * Files that can be renamed now, in every Work
   * @minimum 0
   */
  files: number;
  /** Each Work with a file to rename, a blocked file or a file still moving */
  works: FilesWorkLine[];
}
