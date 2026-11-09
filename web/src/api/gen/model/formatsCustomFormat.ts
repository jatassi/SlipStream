import type { FormatsCondition } from "./formatsCondition.ts";
import type { FormatsCustomFormatDialect } from "./formatsCustomFormatDialect.ts";
import type { FormatsTargetRef } from "./formatsTargetRef.ts";

export interface FormatsCustomFormat {
  conditions: FormatsCondition[];
  /** The app whose values its conditions hold: radarr or sonarr for a pasted format, slipstream for one made here */
  dialect: FormatsCustomFormatDialect;
  id: string;
  /** The Custom Formats naming token lists it */
  includeWhenRenaming: boolean;
  name: string;
  /** Every Target that scores it, by name */
  scoredBy: FormatsTargetRef[];
  /**
   * Releases one of its patterns ran past the match limit on since its last save; they count as not matching it
   * @minimum 0
   */
  slowReleases: number;
}
