import type { Problem } from "./problem.ts";
import type { UpdateBackupSettings500Code } from "./updateBackupSettings500Code.ts";

export type UpdateBackupSettings500 = Problem & {
  code?: UpdateBackupSettings500Code;
};
