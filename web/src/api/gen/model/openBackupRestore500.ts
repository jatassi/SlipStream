import type { OpenBackupRestore500Code } from "./openBackupRestore500Code.ts";
import type { Problem } from "./problem.ts";

export type OpenBackupRestore500 = Problem & {
  code?: OpenBackupRestore500Code;
};
