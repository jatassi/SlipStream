import type { GrabsWorkOutcomeOutcome } from "./grabsWorkOutcomeOutcome.ts";
import type { GrabsWorkOutcomeReason } from "./grabsWorkOutcomeReason.ts";

export interface GrabsWorkOutcome {
  outcome: GrabsWorkOutcomeOutcome;
  /**
   * Why it was skipped; null unless skipped
   * @nullable
   */
  reason: GrabsWorkOutcomeReason;
  workId: string;
}
