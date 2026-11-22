import type { OpenBackupRestore401Code } from "./openBackupRestore401Code.ts";
import type { Problem } from "./problem.ts";

export type OpenBackupRestore401 = Problem & {
  code?: OpenBackupRestore401Code;
};
