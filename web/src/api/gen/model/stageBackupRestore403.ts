import type { Problem } from "./problem.ts";
import type { StageBackupRestore403Code } from "./stageBackupRestore403Code.ts";

export type StageBackupRestore403 = Problem & {
  code?: StageBackupRestore403Code;
};
