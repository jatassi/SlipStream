import type { FinishFirstRun401Code } from "./finishFirstRun401Code.ts";
import type { Problem } from "./problem.ts";

export type FinishFirstRun401 = Problem & {
  code?: FinishFirstRun401Code;
};
