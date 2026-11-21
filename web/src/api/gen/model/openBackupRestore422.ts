import type { OpenBackupRestore422Code } from "./openBackupRestore422Code.ts";
import type { Problem } from "./problem.ts";

export type OpenBackupRestore422 = Problem & {
  code?: OpenBackupRestore422Code;
};
