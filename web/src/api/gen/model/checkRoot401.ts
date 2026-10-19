import type { CheckRoot401Code } from "./checkRoot401Code.ts";
import type { Problem } from "./problem.ts";

export type CheckRoot401 = Problem & {
  code?: CheckRoot401Code;
};
