import type { FinishReconciling500Code } from "./finishReconciling500Code.ts";
import type { Problem } from "./problem.ts";

export type FinishReconciling500 = Problem & {
  code?: FinishReconciling500Code;
};
