import type { ImportPendingInto401Code } from "./importPendingInto401Code.ts";
import type { Problem } from "./problem.ts";

export type ImportPendingInto401 = Problem & {
  code?: ImportPendingInto401Code;
};
