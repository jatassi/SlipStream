import type { Problem } from "./problem.ts";
import type { RehomeRoot422Code } from "./rehomeRoot422Code.ts";
import type { RootsFolderProblem } from "./rootsFolderProblem.ts";
import type { RootsRefusal } from "./rootsRefusal.ts";

export type RehomeRoot422 = Problem & {
  code?: RehomeRoot422Code;
  problem?: RootsFolderProblem;
  refusal?: RootsRefusal;
};
