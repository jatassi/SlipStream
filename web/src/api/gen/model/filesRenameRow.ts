import type { FilesRowBlock } from "./filesRowBlock.ts";

export interface FilesRenameRow {
  /** Why it can't be renamed now; absent when it can */
  blocked?: FilesRowBlock;
  /** Files beside it that move with it, such as its subtitles */
  companions: string[];
  fileId: string;
  /** Where the file is now, relative to its root */
  from: string;
  /**
   * 0 for a row joined to no other; rows with one number trade or chain paths, so they are renamed or left together
   * @minimum 0
   */
  group: number;
  /** Where the naming templates put it; empty when its name doesn't fit */
  to: string;
}
