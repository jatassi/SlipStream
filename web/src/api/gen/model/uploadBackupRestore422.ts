import type { Problem } from "./problem.ts";
import type { UploadBackupRestore422Code } from "./uploadBackupRestore422Code.ts";

export type UploadBackupRestore422 = Problem & {
  code?: UploadBackupRestore422Code;
};
