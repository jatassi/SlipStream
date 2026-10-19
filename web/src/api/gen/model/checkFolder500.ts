import type { CheckFolder500Code } from "./checkFolder500Code.ts";
import type { Problem } from "./problem.ts";

export type CheckFolder500 = Problem & {
  code?: CheckFolder500Code;
};
