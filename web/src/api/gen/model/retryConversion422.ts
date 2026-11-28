import type { Problem } from "./problem.ts";
import type { RetryConversion422Code } from "./retryConversion422Code.ts";

export type RetryConversion422 = Problem & {
  code?: RetryConversion422Code;
};
