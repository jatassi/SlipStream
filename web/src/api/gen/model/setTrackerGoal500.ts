import type { Problem } from "./problem.ts";
import type { SetTrackerGoal500Code } from "./setTrackerGoal500Code.ts";

export type SetTrackerGoal500 = Problem & {
  code?: SetTrackerGoal500Code;
};
