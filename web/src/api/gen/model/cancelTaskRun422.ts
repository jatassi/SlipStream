import type { CancelTaskRun422Code } from "./cancelTaskRun422Code.ts";
import type { Problem } from "./problem.ts";

export type CancelTaskRun422 = Problem & {
  code?: CancelTaskRun422Code;
};
