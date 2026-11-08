import type { Problem } from "./problem.ts";
import type { SetTrackerGoal404Code } from "./setTrackerGoal404Code.ts";

export type SetTrackerGoal404 = Problem & {
  code?: SetTrackerGoal404Code;
};
