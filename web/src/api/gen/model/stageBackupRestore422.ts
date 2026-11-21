import type { Problem } from "./problem.ts";
import type { StageBackupRestore422Code } from "./stageBackupRestore422Code.ts";

export type StageBackupRestore422 = Problem & {
  code?: StageBackupRestore422Code;
};
