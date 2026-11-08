import type { Problem } from "./problem.ts";
import type { SetTrackerGoal401Code } from "./setTrackerGoal401Code.ts";

export type SetTrackerGoal401 = Problem & {
  code?: SetTrackerGoal401Code;
};
