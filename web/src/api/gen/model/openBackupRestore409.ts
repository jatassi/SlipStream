import type { OpenBackupRestore409Code } from "./openBackupRestore409Code.ts";
import type { Problem } from "./problem.ts";

export type OpenBackupRestore409 = Problem & {
  code?: OpenBackupRestore409Code;
};
