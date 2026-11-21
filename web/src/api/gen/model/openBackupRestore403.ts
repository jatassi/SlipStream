import type { OpenBackupRestore403Code } from "./openBackupRestore403Code.ts";
import type { Problem } from "./problem.ts";

export type OpenBackupRestore403 = Problem & {
  code?: OpenBackupRestore403Code;
};
