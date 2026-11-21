import type { Problem } from "./problem.ts";
import type { RehomeReconciling403Code } from "./rehomeReconciling403Code.ts";

export type RehomeReconciling403 = Problem & {
  code?: RehomeReconciling403Code;
};
