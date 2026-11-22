import type { CheckBackupPassphrase422Code } from "./checkBackupPassphrase422Code.ts";
import type { Problem } from "./problem.ts";

export type CheckBackupPassphrase422 = Problem & {
  code?: CheckBackupPassphrase422Code;
};
