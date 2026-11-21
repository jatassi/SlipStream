import type { GetBackupPassphrase500Code } from "./getBackupPassphrase500Code.ts";
import type { Problem } from "./problem.ts";

export type GetBackupPassphrase500 = Problem & {
  code?: GetBackupPassphrase500Code;
};
