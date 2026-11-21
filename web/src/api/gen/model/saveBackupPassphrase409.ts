import type { Problem } from "./problem.ts";
import type { SaveBackupPassphrase409Code } from "./saveBackupPassphrase409Code.ts";

export type SaveBackupPassphrase409 = Problem & {
  code?: SaveBackupPassphrase409Code;
};
