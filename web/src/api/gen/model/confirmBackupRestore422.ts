import type { ConfirmBackupRestore422Code } from "./confirmBackupRestore422Code.ts";
import type { Problem } from "./problem.ts";

export type ConfirmBackupRestore422 = Problem & {
  code?: ConfirmBackupRestore422Code;
};
