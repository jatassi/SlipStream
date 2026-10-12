import type { ClaimController409Code } from "./claimController409Code.ts";
import type { Problem } from "./problem.ts";

export type ClaimController409 = Problem & {
  code?: ClaimController409Code;
};
