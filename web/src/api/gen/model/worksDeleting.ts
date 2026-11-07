import type { WorksDeletingFiles } from "./worksDeletingFiles.ts";

export interface WorksDeleting {
  /** Delete the Works' library files from disk, or leave them where they are */
  files: WorksDeletingFiles;
  /** Return the plan and the outcomes it predicts, and delete nothing */
  preview?: boolean;
  /**
   * A repeated id counts once
   * @minItems 1
   * @maxItems 20000
   */
  workIds: string[];
}
