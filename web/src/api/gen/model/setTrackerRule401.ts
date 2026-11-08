import type { Problem } from "./problem.ts";
import type { SetTrackerRule401Code } from "./setTrackerRule401Code.ts";

export type SetTrackerRule401 = Problem & {
  code?: SetTrackerRule401Code;
};
