import type { Problem } from "./problem.ts";
import type { RetryConversion403Code } from "./retryConversion403Code.ts";

export type RetryConversion403 = Problem & {
  code?: RetryConversion403Code;
};
