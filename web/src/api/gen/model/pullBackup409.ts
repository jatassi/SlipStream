import type { Problem } from "./problem.ts";
import type { PullBackup409Code } from "./pullBackup409Code.ts";

export type PullBackup409 = Problem & {
  code?: PullBackup409Code;
};
