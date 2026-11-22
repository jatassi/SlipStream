import type { OpenBackupRestore404Code } from "./openBackupRestore404Code.ts";
import type { Problem } from "./problem.ts";

export type OpenBackupRestore404 = Problem & {
  code?: OpenBackupRestore404Code;
};
