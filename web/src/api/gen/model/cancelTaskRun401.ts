import type { CancelTaskRun401Code } from "./cancelTaskRun401Code.ts";
import type { Problem } from "./problem.ts";

export type CancelTaskRun401 = Problem & {
  code?: CancelTaskRun401Code;
};
