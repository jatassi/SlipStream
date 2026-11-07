import type { MergeTracker500Code } from "./mergeTracker500Code.ts";
import type { Problem } from "./problem.ts";

export type MergeTracker500 = Problem & {
  code?: MergeTracker500Code;
};
