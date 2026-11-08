import type { Problem } from "./problem.ts";
import type { SetDefaultGoal409Code } from "./setDefaultGoal409Code.ts";

export type SetDefaultGoal409 = Problem & {
  code?: SetDefaultGoal409Code;
};
