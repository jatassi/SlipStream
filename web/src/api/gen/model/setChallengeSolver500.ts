import type { Problem } from "./problem.ts";
import type { SetChallengeSolver500Code } from "./setChallengeSolver500Code.ts";

export type SetChallengeSolver500 = Problem & {
  code?: SetChallengeSolver500Code;
};
