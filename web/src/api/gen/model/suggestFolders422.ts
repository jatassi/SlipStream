import type { Problem } from "./problem.ts";
import type { RootsFolderProblem } from "./rootsFolderProblem.ts";
import type { SuggestFolders422Code } from "./suggestFolders422Code.ts";

export type SuggestFolders422 = Problem & {
  code?: SuggestFolders422Code;
  problem?: RootsFolderProblem;
};
