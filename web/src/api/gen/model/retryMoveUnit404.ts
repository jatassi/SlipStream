import type { Problem } from "./problem.ts";
import type { RetryMoveUnit404Code } from "./retryMoveUnit404Code.ts";

export type RetryMoveUnit404 = Problem & {
  code?: RetryMoveUnit404Code;
};
