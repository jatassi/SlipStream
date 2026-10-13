import type { ClaimWorker401Code } from "./claimWorker401Code.ts";
import type { Problem } from "./problem.ts";

export type ClaimWorker401 = Problem & {
  code?: ClaimWorker401Code;
};
