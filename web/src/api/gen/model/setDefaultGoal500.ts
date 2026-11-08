import type { Problem } from "./problem.ts";
import type { SetDefaultGoal500Code } from "./setDefaultGoal500Code.ts";

export type SetDefaultGoal500 = Problem & {
  code?: SetDefaultGoal500Code;
};
