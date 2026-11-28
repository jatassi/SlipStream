import type { Problem } from "./problem.ts";
import type { RetryConversion401Code } from "./retryConversion401Code.ts";

export type RetryConversion401 = Problem & {
  code?: RetryConversion401Code;
};
