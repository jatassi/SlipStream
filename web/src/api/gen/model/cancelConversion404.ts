import type { CancelConversion404Code } from "./cancelConversion404Code.ts";
import type { Problem } from "./problem.ts";

export type CancelConversion404 = Problem & {
  code?: CancelConversion404Code;
};
