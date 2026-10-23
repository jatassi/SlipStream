import type { CheckHealthCategory401Code } from "./checkHealthCategory401Code.ts";
import type { Problem } from "./problem.ts";

export type CheckHealthCategory401 = Problem & {
  code?: CheckHealthCategory401Code;
};
