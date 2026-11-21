import type { CheckBackupPassphrase500Code } from "./checkBackupPassphrase500Code.ts";
import type { Problem } from "./problem.ts";

export type CheckBackupPassphrase500 = Problem & {
  code?: CheckBackupPassphrase500Code;
};
