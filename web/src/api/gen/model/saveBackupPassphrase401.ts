import type { Problem } from "./problem.ts";
import type { SaveBackupPassphrase401Code } from "./saveBackupPassphrase401Code.ts";

export type SaveBackupPassphrase401 = Problem & {
  code?: SaveBackupPassphrase401Code;
};
