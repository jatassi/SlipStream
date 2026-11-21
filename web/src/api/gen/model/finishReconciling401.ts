import type { FinishReconciling401Code } from "./finishReconciling401Code.ts";
import type { Problem } from "./problem.ts";

export type FinishReconciling401 = Problem & {
  code?: FinishReconciling401Code;
};
