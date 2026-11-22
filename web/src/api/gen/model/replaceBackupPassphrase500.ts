import type { Problem } from "./problem.ts";
import type { ReplaceBackupPassphrase500Code } from "./replaceBackupPassphrase500Code.ts";

export type ReplaceBackupPassphrase500 = Problem & {
  code?: ReplaceBackupPassphrase500Code;
};
