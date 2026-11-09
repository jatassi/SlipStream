import type { MarkRootGone404Code } from "./markRootGone404Code.ts";
import type { Problem } from "./problem.ts";

export type MarkRootGone404 = Problem & {
  code?: MarkRootGone404Code;
};
