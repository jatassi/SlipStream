import type { WorksFailedOutcome } from "./worksFailedOutcome.ts";
import type { WorksMonitoringDoneOutcome } from "./worksMonitoringDoneOutcome.ts";
import type { WorksMonitoringSkippedOutcome } from "./worksMonitoringSkippedOutcome.ts";

export type WorksMonitoringOutcome =
  | WorksMonitoringDoneOutcome
  | WorksMonitoringSkippedOutcome
  | WorksFailedOutcome;
