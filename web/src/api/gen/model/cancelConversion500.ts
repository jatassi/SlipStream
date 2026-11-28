import type { CancelConversion500Code } from "./cancelConversion500Code.ts";
import type { Problem } from "./problem.ts";

export type CancelConversion500 = Problem & {
  code?: CancelConversion500Code;
};
