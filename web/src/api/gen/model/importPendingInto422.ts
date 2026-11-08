import type { ImportPendingInto422Code } from "./importPendingInto422Code.ts";
import type { Problem } from "./problem.ts";

export type ImportPendingInto422 = Problem & {
  code?: ImportPendingInto422Code;
};
