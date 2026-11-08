import type { Problem } from "./problem.ts";
import type { SetDefaultGoal401Code } from "./setDefaultGoal401Code.ts";

export type SetDefaultGoal401 = Problem & {
  code?: SetDefaultGoal401Code;
};
