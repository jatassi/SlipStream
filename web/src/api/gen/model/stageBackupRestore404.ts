import type { Problem } from "./problem.ts";
import type { StageBackupRestore404Code } from "./stageBackupRestore404Code.ts";

export type StageBackupRestore404 = Problem & {
  code?: StageBackupRestore404Code;
};
