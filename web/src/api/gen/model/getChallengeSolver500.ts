import type { GetChallengeSolver500Code } from "./getChallengeSolver500Code.ts";
import type { Problem } from "./problem.ts";

export type GetChallengeSolver500 = Problem & {
  code?: GetChallengeSolver500Code;
};
