import type { CheckBackupPassphrase403Code } from "./checkBackupPassphrase403Code.ts";
import type { Problem } from "./problem.ts";

export type CheckBackupPassphrase403 = Problem & {
  code?: CheckBackupPassphrase403Code;
};
