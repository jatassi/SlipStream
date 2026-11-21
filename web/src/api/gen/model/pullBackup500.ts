import type { Problem } from "./problem.ts";
import type { PullBackup500Code } from "./pullBackup500Code.ts";

export type PullBackup500 = Problem & {
  code?: PullBackup500Code;
};
