import type { BrowseFolders422Code } from "./browseFolders422Code.ts";
import type { Problem } from "./problem.ts";
import type { RootsFolderProblem } from "./rootsFolderProblem.ts";

export type BrowseFolders422 = Problem & {
  code?: BrowseFolders422Code;
  problem?: RootsFolderProblem;
};
