import type { GetBackupSettings500Code } from "./getBackupSettings500Code.ts";
import type { Problem } from "./problem.ts";

export type GetBackupSettings500 = Problem & {
  code?: GetBackupSettings500Code;
};
