import type { MergeTracker401Code } from "./mergeTracker401Code.ts";
import type { Problem } from "./problem.ts";

export type MergeTracker401 = Problem & {
  code?: MergeTracker401Code;
};
