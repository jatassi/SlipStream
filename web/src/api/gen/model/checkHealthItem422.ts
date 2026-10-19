import type { CheckHealthItem422Code } from "./checkHealthItem422Code.ts";
import type { Problem } from "./problem.ts";

export type CheckHealthItem422 = Problem & {
  code?: CheckHealthItem422Code;
};
