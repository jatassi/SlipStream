import type { GetBackupPassphrase401Code } from "./getBackupPassphrase401Code.ts";
import type { Problem } from "./problem.ts";

export type GetBackupPassphrase401 = Problem & {
  code?: GetBackupPassphrase401Code;
};
