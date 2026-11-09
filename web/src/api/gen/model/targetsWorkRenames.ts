import type { TargetsRenamedFile } from "./targetsRenamedFile.ts";

export interface TargetsWorkRenames {
  /** In the order the files were added */
  files: TargetsRenamedFile[];
  title: string;
  workId: string;
}
