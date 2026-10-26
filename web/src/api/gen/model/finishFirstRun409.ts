import type { FinishFirstRun409Code } from "./finishFirstRun409Code.ts";
import type { Problem } from "./problem.ts";

export type FinishFirstRun409 = Problem & {
  code?: FinishFirstRun409Code;
};
