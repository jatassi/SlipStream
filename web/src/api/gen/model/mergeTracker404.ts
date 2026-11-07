import type { MergeTracker404Code } from "./mergeTracker404Code.ts";
import type { Problem } from "./problem.ts";

export type MergeTracker404 = Problem & {
  code?: MergeTracker404Code;
};
