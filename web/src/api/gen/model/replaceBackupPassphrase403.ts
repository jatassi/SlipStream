import type { Problem } from "./problem.ts";
import type { ReplaceBackupPassphrase403Code } from "./replaceBackupPassphrase403Code.ts";

export type ReplaceBackupPassphrase403 = Problem & {
  code?: ReplaceBackupPassphrase403Code;
};
