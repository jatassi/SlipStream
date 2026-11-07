import type { MergeTracker422Code } from "./mergeTracker422Code.ts";
import type { Problem } from "./problem.ts";

export type MergeTracker422 = Problem & {
  code?: MergeTracker422Code;
};
