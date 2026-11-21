import type { Problem } from "./problem.ts";
import type { UploadBackupRestore409Code } from "./uploadBackupRestore409Code.ts";

export type UploadBackupRestore409 = Problem & {
  code?: UploadBackupRestore409Code;
};
