import type { CheckHealthItem401Code } from "./checkHealthItem401Code.ts";
import type { Problem } from "./problem.ts";

export type CheckHealthItem401 = Problem & {
  code?: CheckHealthItem401Code;
};
