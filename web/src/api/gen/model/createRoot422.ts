import type { CreateRoot422Code } from "./createRoot422Code.ts";
import type { Problem } from "./problem.ts";
import type { RootsFolderProblem } from "./rootsFolderProblem.ts";
import type { RootsRefusal } from "./rootsRefusal.ts";

export type CreateRoot422 = Problem & {
  code?: CreateRoot422Code;
  problem?: RootsFolderProblem;
  refusal?: RootsRefusal;
};
