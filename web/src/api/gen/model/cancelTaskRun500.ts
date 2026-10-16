import type { CancelTaskRun500Code } from "./cancelTaskRun500Code.ts";
import type { Problem } from "./problem.ts";

export type CancelTaskRun500 = Problem & {
  code?: CancelTaskRun500Code;
};
