import type { AddBackupDestination422Code } from "./addBackupDestination422Code.ts";
import type { BackupsRefusal } from "./backupsRefusal.ts";
import type { Problem } from "./problem.ts";

export type AddBackupDestination422 = Problem & {
  code?: AddBackupDestination422Code;
  refusal?: BackupsRefusal;
};
