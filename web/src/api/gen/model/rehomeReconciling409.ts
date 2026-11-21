import type { Problem } from "./problem.ts";
import type { RehomeReconciling409Code } from "./rehomeReconciling409Code.ts";

export type RehomeReconciling409 = Problem & {
  code?: RehomeReconciling409Code;
};
