import type { PrepareBackupDownload401Code } from "./prepareBackupDownload401Code.ts";
import type { Problem } from "./problem.ts";

export type PrepareBackupDownload401 = Problem & {
  code?: PrepareBackupDownload401Code;
};
