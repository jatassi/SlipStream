import type { Problem } from "./problem.ts";
import type { SetTrackerRule500Code } from "./setTrackerRule500Code.ts";

export type SetTrackerRule500 = Problem & {
  code?: SetTrackerRule500Code;
};
