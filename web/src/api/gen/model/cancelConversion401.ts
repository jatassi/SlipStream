import type { CancelConversion401Code } from "./cancelConversion401Code.ts";
import type { Problem } from "./problem.ts";

export type CancelConversion401 = Problem & {
  code?: CancelConversion401Code;
};
