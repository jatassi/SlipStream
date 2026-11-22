import type { Problem } from "./problem.ts";
import type { PullBackup403Code } from "./pullBackup403Code.ts";

export type PullBackup403 = Problem & {
  code?: PullBackup403Code;
};
