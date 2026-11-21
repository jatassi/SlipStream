import type { Problem } from "./problem.ts";
import type { RehomeReconciling404Code } from "./rehomeReconciling404Code.ts";

export type RehomeReconciling404 = Problem & {
  code?: RehomeReconciling404Code;
};
