import type { FinishReconciling409Code } from "./finishReconciling409Code.ts";
import type { Problem } from "./problem.ts";

export type FinishReconciling409 = Problem & {
  code?: FinishReconciling409Code;
};
