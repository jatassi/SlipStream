import type { Problem } from "./problem.ts";
import type { RetryConversion409Code } from "./retryConversion409Code.ts";

export type RetryConversion409 = Problem & {
  code?: RetryConversion409Code;
};
