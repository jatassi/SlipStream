import type { ClaimController500Code } from "./claimController500Code.ts";
import type { Problem } from "./problem.ts";

export type ClaimController500 = Problem & {
  code?: ClaimController500Code;
};
