import type { Problem } from "./problem.ts";
import type { SetTrackerRoute500Code } from "./setTrackerRoute500Code.ts";

export type SetTrackerRoute500 = Problem & {
  code?: SetTrackerRoute500Code;
};
