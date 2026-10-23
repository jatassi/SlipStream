import type { CheckHealthCategory500Code } from "./checkHealthCategory500Code.ts";
import type { Problem } from "./problem.ts";

export type CheckHealthCategory500 = Problem & {
  code?: CheckHealthCategory500Code;
};
