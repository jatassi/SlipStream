import type { ClaimController403Code } from "./claimController403Code.ts";
import type { Problem } from "./problem.ts";

export type ClaimController403 = Problem & {
  code?: ClaimController403Code;
};
