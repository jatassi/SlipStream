import type { GetBackupPassphrase403Code } from "./getBackupPassphrase403Code.ts";
import type { Problem } from "./problem.ts";

export type GetBackupPassphrase403 = Problem & {
  code?: GetBackupPassphrase403Code;
};
