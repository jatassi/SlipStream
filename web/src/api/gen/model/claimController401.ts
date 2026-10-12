import type { ClaimController401Code } from "./claimController401Code.ts";
import type { Problem } from "./problem.ts";

export type ClaimController401 = Problem & {
  code?: ClaimController401Code;
};
