import type { HealthCategory } from "./healthCategory.ts";
import type { HealthPointer } from "./healthPointer.ts";
import type { HealthReportLevel } from "./healthReportLevel.ts";

export interface HealthReport {
  /** In page order */
  categories: HealthCategory[];
  /** The item the status sentence leads with: the worst, then the oldest, then in page order; null while nothing is wrong */
  first: HealthPointer | null;
  /**
   * The worst level over every category, null while nothing is wrong
   * @nullable
   */
  level: HealthReportLevel;
}
