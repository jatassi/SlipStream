import type { Problem } from "./problem.ts";
import type { RetryConversion404Code } from "./retryConversion404Code.ts";

export type RetryConversion404 = Problem & {
  code?: RetryConversion404Code;
};
