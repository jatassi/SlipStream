import type { CountUnmatched401Code } from "./countUnmatched401Code.ts";
import type { Problem } from "./problem.ts";

export type CountUnmatched401 = Problem & {
  code?: CountUnmatched401Code;
};
