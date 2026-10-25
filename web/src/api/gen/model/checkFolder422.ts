import type { CheckFolder422Code } from "./checkFolder422Code.ts";
import type { Problem } from "./problem.ts";
import type { RootsFolderProblem } from "./rootsFolderProblem.ts";

export type CheckFolder422 = Problem & {
  code?: CheckFolder422Code;
  problem?: RootsFolderProblem;
};
