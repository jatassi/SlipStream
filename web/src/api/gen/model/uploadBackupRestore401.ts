import type { Problem } from "./problem.ts";
import type { UploadBackupRestore401Code } from "./uploadBackupRestore401Code.ts";

export type UploadBackupRestore401 = Problem & {
  code?: UploadBackupRestore401Code;
};
