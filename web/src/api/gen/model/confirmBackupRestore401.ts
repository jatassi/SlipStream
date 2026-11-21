import type { ConfirmBackupRestore401Code } from "./confirmBackupRestore401Code.ts";
import type { Problem } from "./problem.ts";

export type ConfirmBackupRestore401 = Problem & {
  code?: ConfirmBackupRestore401Code;
};
