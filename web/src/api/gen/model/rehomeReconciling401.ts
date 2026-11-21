import type { Problem } from "./problem.ts";
import type { RehomeReconciling401Code } from "./rehomeReconciling401Code.ts";

export type RehomeReconciling401 = Problem & {
  code?: RehomeReconciling401Code;
};
