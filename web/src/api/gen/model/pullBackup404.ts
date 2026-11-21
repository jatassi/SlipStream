import type { Problem } from "./problem.ts";
import type { PullBackup404Code } from "./pullBackup404Code.ts";

export type PullBackup404 = Problem & {
  code?: PullBackup404Code;
};
