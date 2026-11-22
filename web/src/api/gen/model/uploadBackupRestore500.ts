import type { Problem } from "./problem.ts";
import type { UploadBackupRestore500Code } from "./uploadBackupRestore500Code.ts";

export type UploadBackupRestore500 = Problem & {
  code?: UploadBackupRestore500Code;
};
