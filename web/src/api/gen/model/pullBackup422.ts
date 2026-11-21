import type { Problem } from "./problem.ts";
import type { PullBackup422Code } from "./pullBackup422Code.ts";

export type PullBackup422 = Problem & {
  code?: PullBackup422Code;
};
