import type { CheckHealthItem500Code } from "./checkHealthItem500Code.ts";
import type { Problem } from "./problem.ts";

export type CheckHealthItem500 = Problem & {
  code?: CheckHealthItem500Code;
};
