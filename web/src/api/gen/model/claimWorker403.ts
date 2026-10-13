import type { ClaimWorker403Code } from "./claimWorker403Code.ts";
import type { Problem } from "./problem.ts";

export type ClaimWorker403 = Problem & {
  code?: ClaimWorker403Code;
};
