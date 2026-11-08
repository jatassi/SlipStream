import type { Problem } from "./problem.ts";
import type { SetTrackerRoute403Code } from "./setTrackerRoute403Code.ts";

export type SetTrackerRoute403 = Problem & {
  code?: SetTrackerRoute403Code;
};
