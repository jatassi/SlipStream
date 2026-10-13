import type { ClaimWorker422Code } from "./claimWorker422Code.ts";
import type { Problem } from "./problem.ts";

export type ClaimWorker422 = Problem & {
  code?: ClaimWorker422Code;
};
