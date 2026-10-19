import type { CheckRoot422Code } from "./checkRoot422Code.ts";
import type { Problem } from "./problem.ts";
import type { RootsFolderProblem } from "./rootsFolderProblem.ts";

export type CheckRoot422 = Problem & {
  code?: CheckRoot422Code;
  problem?: RootsFolderProblem;
};
