import type { MergeTracker409Code } from "./mergeTracker409Code.ts";
import type { Problem } from "./problem.ts";

export type MergeTracker409 = Problem & {
  code?: MergeTracker409Code;
};
