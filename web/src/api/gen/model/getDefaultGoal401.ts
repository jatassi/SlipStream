import type { GetDefaultGoal401Code } from "./getDefaultGoal401Code.ts";
import type { Problem } from "./problem.ts";

export type GetDefaultGoal401 = Problem & {
  code?: GetDefaultGoal401Code;
};
