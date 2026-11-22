import type { Problem } from "./problem.ts";
import type { StageBackupRestore500Code } from "./stageBackupRestore500Code.ts";

export type StageBackupRestore500 = Problem & {
  code?: StageBackupRestore500Code;
};
