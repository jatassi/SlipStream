import type { Problem } from "./problem.ts";
import type { RetryConversion501Code } from "./retryConversion501Code.ts";

export type RetryConversion501 = Problem & {
  code?: RetryConversion501Code;
};
