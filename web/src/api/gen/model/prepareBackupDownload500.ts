import type { PrepareBackupDownload500Code } from "./prepareBackupDownload500Code.ts";
import type { Problem } from "./problem.ts";

export type PrepareBackupDownload500 = Problem & {
  code?: PrepareBackupDownload500Code;
};
