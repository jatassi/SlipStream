import type { WorksSkippedOutcomeOutcome } from "./worksSkippedOutcomeOutcome.ts";
import type { WorksSkippedOutcomeReason } from "./worksSkippedOutcomeReason.ts";

export interface WorksSkippedOutcome {
  outcome: WorksSkippedOutcomeOutcome;
  /** Why the action left the Work as it was */
  reason: WorksSkippedOutcomeReason;
  workId: string;
}
