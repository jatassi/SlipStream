import type { ClaimWorker500Code } from "./claimWorker500Code.ts";
import type { Problem } from "./problem.ts";

export type ClaimWorker500 = Problem & {
  code?: ClaimWorker500Code;
};
