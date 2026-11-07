import type { WorksMonitoringOutcome } from "./worksMonitoringOutcome.ts";

export interface WorksMonitoringList {
  /** One per Work, in request order */
  outcomes: WorksMonitoringOutcome[];
}
