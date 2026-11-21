import type { Problem } from "./problem.ts";
import type { RehomeReconciling500Code } from "./rehomeReconciling500Code.ts";

export type RehomeReconciling500 = Problem & {
  code?: RehomeReconciling500Code;
};
