import type { Problem } from "./problem.ts";
import type { RetryMoveUnit500Code } from "./retryMoveUnit500Code.ts";

export type RetryMoveUnit500 = Problem & {
  code?: RetryMoveUnit500Code;
};
