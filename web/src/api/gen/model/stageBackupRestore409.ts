import type { Problem } from "./problem.ts";
import type { StageBackupRestore409Code } from "./stageBackupRestore409Code.ts";

export type StageBackupRestore409 = Problem & {
  code?: StageBackupRestore409Code;
};
