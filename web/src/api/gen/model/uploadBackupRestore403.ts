import type { Problem } from "./problem.ts";
import type { UploadBackupRestore403Code } from "./uploadBackupRestore403Code.ts";

export type UploadBackupRestore403 = Problem & {
  code?: UploadBackupRestore403Code;
};
