import type { Problem } from "./problem.ts";
import type { StageBackupRestore401Code } from "./stageBackupRestore401Code.ts";

export type StageBackupRestore401 = Problem & {
  code?: StageBackupRestore401Code;
};
