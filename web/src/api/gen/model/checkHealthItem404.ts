import type { CheckHealthItem404Code } from "./checkHealthItem404Code.ts";
import type { Problem } from "./problem.ts";

export type CheckHealthItem404 = Problem & {
  code?: CheckHealthItem404Code;
};
