import type { GetDefaultGoal500Code } from "./getDefaultGoal500Code.ts";
import type { Problem } from "./problem.ts";

export type GetDefaultGoal500 = Problem & {
  code?: GetDefaultGoal500Code;
};
