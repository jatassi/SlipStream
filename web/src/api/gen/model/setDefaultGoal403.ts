import type { Problem } from "./problem.ts";
import type { SetDefaultGoal403Code } from "./setDefaultGoal403Code.ts";

export type SetDefaultGoal403 = Problem & {
  code?: SetDefaultGoal403Code;
};
