import type { CheckBackupPassphrase401Code } from "./checkBackupPassphrase401Code.ts";
import type { Problem } from "./problem.ts";

export type CheckBackupPassphrase401 = Problem & {
  code?: CheckBackupPassphrase401Code;
};
