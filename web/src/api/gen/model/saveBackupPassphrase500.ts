import type { Problem } from "./problem.ts";
import type { SaveBackupPassphrase500Code } from "./saveBackupPassphrase500Code.ts";

export type SaveBackupPassphrase500 = Problem & {
  code?: SaveBackupPassphrase500Code;
};
