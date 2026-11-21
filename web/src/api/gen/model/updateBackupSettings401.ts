import type { Problem } from "./problem.ts";
import type { UpdateBackupSettings401Code } from "./updateBackupSettings401Code.ts";

export type UpdateBackupSettings401 = Problem & {
  code?: UpdateBackupSettings401Code;
};
