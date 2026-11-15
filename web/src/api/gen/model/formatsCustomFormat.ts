import type { FormatsCondition } from "./formatsCondition.ts";
import type { FormatsCustomFormatDialect } from "./formatsCustomFormatDialect.ts";
import type { FormatsTargetRef } from "./formatsTargetRef.ts";

export interface FormatsCustomFormat {
  conditions: FormatsCondition[];
  /** The app whose values its conditions hold: radarr or sonarr for a pasted format, slipstream for one made here */
  dialect: FormatsCustomFormatDialect;
  id: string;
  /** A file that matches it gets its name in the {Custom Formats} naming token */
  includeWhenRenaming: boolean;
  name: string;
  /** Every Target that scores it, by name */
  scoredBy: FormatsTargetRef[];
  /**
   * How many releases a pattern of this format timed out on since it was last saved; those releases count as not matching
   * @minimum 0
   */
  slowReleases: number;
}
