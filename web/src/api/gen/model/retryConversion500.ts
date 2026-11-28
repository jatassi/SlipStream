import type { Problem } from "./problem.ts";
import type { RetryConversion500Code } from "./retryConversion500Code.ts";

export type RetryConversion500 = Problem & {
  code?: RetryConversion500Code;
};
