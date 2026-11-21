import type { Problem } from "./problem.ts";
import type { PullBackup401Code } from "./pullBackup401Code.ts";

export type PullBackup401 = Problem & {
  code?: PullBackup401Code;
};
