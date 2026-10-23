import type { CheckHealthItem409Code } from "./checkHealthItem409Code.ts";
import type { Problem } from "./problem.ts";

export type CheckHealthItem409 = Problem & {
  code?: CheckHealthItem409Code;
};
