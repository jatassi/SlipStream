import type { Problem } from "./problem.ts";
import type { SaveBackupPassphrase403Code } from "./saveBackupPassphrase403Code.ts";

export type SaveBackupPassphrase403 = Problem & {
  code?: SaveBackupPassphrase403Code;
};
