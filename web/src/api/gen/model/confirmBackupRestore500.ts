import type { ConfirmBackupRestore500Code } from "./confirmBackupRestore500Code.ts";
import type { Problem } from "./problem.ts";

export type ConfirmBackupRestore500 = Problem & {
  code?: ConfirmBackupRestore500Code;
};
