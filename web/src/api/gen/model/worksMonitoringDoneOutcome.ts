import type { WorksMonitoringDoneOutcomeOutcome } from "./worksMonitoringDoneOutcomeOutcome.ts";
import type { WorksPresetCounts } from "./worksPresetCounts.ts";

export interface WorksMonitoringDoneOutcome {
  /** What each preset would monitor of the Work; null unless a preview */
  counts: WorksPresetCounts | null;
  outcome: WorksMonitoringDoneOutcomeOutcome;
  /** Items the change starts monitoring */
  start: number;
  /** Items the change stops monitoring */
  stop: number;
  workId: string;
}
