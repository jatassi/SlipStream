import type { FilesRenameRow } from "./filesRenameRow.ts";

export interface FilesWorkRenamePlan {
  contentType: string;
  /** A hash of the Work's rows, which a confirm sends back */
  digest: string;
  /**
   * Files the disk queue still moves, which the plan leaves out
   * @minimum 0
   */
  moving: number;
  /** Each file whose path differs from what the naming templates make */
  rows: FilesRenameRow[];
  title: string;
  workId: string;
}
