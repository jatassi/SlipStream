import type { GetBackupSettings401Code } from "./getBackupSettings401Code.ts";
import type { Problem } from "./problem.ts";

export type GetBackupSettings401 = Problem & {
  code?: GetBackupSettings401Code;
};
