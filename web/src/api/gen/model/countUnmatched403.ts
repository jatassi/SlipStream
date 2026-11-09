import type { CountUnmatched403Code } from "./countUnmatched403Code.ts";
import type { Problem } from "./problem.ts";

export type CountUnmatched403 = Problem & {
  code?: CountUnmatched403Code;
};
