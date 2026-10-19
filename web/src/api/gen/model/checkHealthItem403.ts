import type { CheckHealthItem403Code } from "./checkHealthItem403Code.ts";
import type { Problem } from "./problem.ts";

export type CheckHealthItem403 = Problem & {
  code?: CheckHealthItem403Code;
};
