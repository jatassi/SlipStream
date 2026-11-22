import type { ConfirmBackupRestore409Code } from "./confirmBackupRestore409Code.ts";
import type { Problem } from "./problem.ts";

export type ConfirmBackupRestore409 = Problem & {
  code?: ConfirmBackupRestore409Code;
};
