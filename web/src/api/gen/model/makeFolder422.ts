import type { MakeFolder422Code } from "./makeFolder422Code.ts";
import type { Problem } from "./problem.ts";
import type { RootsFolderProblem } from "./rootsFolderProblem.ts";

export type MakeFolder422 = Problem & {
  code?: MakeFolder422Code;
  problem?: RootsFolderProblem;
};
