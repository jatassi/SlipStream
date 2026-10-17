import type { EndSession422Code } from "./endSession422Code.ts";
import type { Problem } from "./problem.ts";

export type EndSession422 = Problem & {
  code?: EndSession422Code;
};
