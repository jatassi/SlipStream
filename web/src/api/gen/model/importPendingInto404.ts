import type { ImportPendingInto404Code } from "./importPendingInto404Code.ts";
import type { Problem } from "./problem.ts";

export type ImportPendingInto404 = Problem & {
  code?: ImportPendingInto404Code;
};
