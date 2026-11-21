import type { ConfirmBackupRestore404Code } from "./confirmBackupRestore404Code.ts";
import type { Problem } from "./problem.ts";

export type ConfirmBackupRestore404 = Problem & {
  code?: ConfirmBackupRestore404Code;
};
