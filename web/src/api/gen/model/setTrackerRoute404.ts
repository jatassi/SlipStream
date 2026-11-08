import type { Problem } from "./problem.ts";
import type { SetTrackerRoute404Code } from "./setTrackerRoute404Code.ts";

export type SetTrackerRoute404 = Problem & {
  code?: SetTrackerRoute404Code;
};
