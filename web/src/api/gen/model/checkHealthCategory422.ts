import type { CheckHealthCategory422Code } from "./checkHealthCategory422Code.ts";
import type { Problem } from "./problem.ts";

export type CheckHealthCategory422 = Problem & {
  code?: CheckHealthCategory422Code;
};
