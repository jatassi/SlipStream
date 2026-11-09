import type { CountUnmatched500Code } from "./countUnmatched500Code.ts";
import type { Problem } from "./problem.ts";

export type CountUnmatched500 = Problem & {
  code?: CountUnmatched500Code;
};
