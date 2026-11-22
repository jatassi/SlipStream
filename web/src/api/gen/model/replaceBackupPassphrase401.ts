import type { Problem } from "./problem.ts";
import type { ReplaceBackupPassphrase401Code } from "./replaceBackupPassphrase401Code.ts";

export type ReplaceBackupPassphrase401 = Problem & {
  code?: ReplaceBackupPassphrase401Code;
};
