import type { WorksMonitoringSkippedOutcomeOutcome } from "./worksMonitoringSkippedOutcomeOutcome.ts";
import type { WorksMonitoringSkippedOutcomeReason } from "./worksMonitoringSkippedOutcomeReason.ts";
import type { WorksPresetCounts } from "./worksPresetCounts.ts";

export interface WorksMonitoringSkippedOutcome {
  /** What each preset would monitor of the Work; null unless a preview */
  counts: WorksPresetCounts | null;
  outcome: WorksMonitoringSkippedOutcomeOutcome;
  /** Why the change left the Work as it was */
  reason: WorksMonitoringSkippedOutcomeReason;
  workId: string;
}
