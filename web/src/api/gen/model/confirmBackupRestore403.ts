import type { ConfirmBackupRestore403Code } from "./confirmBackupRestore403Code.ts";
import type { Problem } from "./problem.ts";

export type ConfirmBackupRestore403 = Problem & {
  code?: ConfirmBackupRestore403Code;
};
