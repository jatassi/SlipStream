import type { FirstRunStepInputBodyOutcome } from "./firstRunStepInputBodyOutcome.ts";

export interface FirstRunStepInputBody {
  /** done for Continue, skipped for Skip */
  outcome: FirstRunStepInputBodyOutcome;
}
