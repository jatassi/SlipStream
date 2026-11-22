import type { Problem } from "./problem.ts";
import type { UploadBackupRestore415Code } from "./uploadBackupRestore415Code.ts";

export type UploadBackupRestore415 = Problem & {
  code?: UploadBackupRestore415Code;
};
