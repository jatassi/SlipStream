import type { CheckRoot500Code } from "./checkRoot500Code.ts";
import type { Problem } from "./problem.ts";

export type CheckRoot500 = Problem & {
  code?: CheckRoot500Code;
};
