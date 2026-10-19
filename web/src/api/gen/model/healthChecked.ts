import type { HealthReport } from "./healthReport.ts";
import type { HealthUnreached } from "./healthUnreached.ts";

export interface HealthChecked {
  /** What is wrong right after the check */
  report: HealthReport;
  /** What the check couldn't reach; those items stay as they were */
  unreached: HealthUnreached[];
}
