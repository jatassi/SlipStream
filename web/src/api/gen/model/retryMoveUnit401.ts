import type { Problem } from "./problem.ts";
import type { RetryMoveUnit401Code } from "./retryMoveUnit401Code.ts";

export type RetryMoveUnit401 = Problem & {
  code?: RetryMoveUnit401Code;
};
