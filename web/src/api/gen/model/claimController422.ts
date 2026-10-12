import type { ClaimController422Code } from "./claimController422Code.ts";
import type { Problem } from "./problem.ts";

export type ClaimController422 = Problem & {
  code?: ClaimController422Code;
};
