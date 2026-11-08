import type { Problem } from "./problem.ts";
import type { SetTrackerRoute401Code } from "./setTrackerRoute401Code.ts";

export type SetTrackerRoute401 = Problem & {
  code?: SetTrackerRoute401Code;
};
