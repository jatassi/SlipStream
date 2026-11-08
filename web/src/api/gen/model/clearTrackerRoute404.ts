import type { ClearTrackerRoute404Code } from "./clearTrackerRoute404Code.ts";
import type { Problem } from "./problem.ts";

export type ClearTrackerRoute404 = Problem & {
  code?: ClearTrackerRoute404Code;
};
