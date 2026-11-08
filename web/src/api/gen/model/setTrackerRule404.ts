import type { Problem } from "./problem.ts";
import type { SetTrackerRule404Code } from "./setTrackerRule404Code.ts";

export type SetTrackerRule404 = Problem & {
  code?: SetTrackerRule404Code;
};
