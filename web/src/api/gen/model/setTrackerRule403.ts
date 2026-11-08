import type { Problem } from "./problem.ts";
import type { SetTrackerRule403Code } from "./setTrackerRule403Code.ts";

export type SetTrackerRule403 = Problem & {
  code?: SetTrackerRule403Code;
};
