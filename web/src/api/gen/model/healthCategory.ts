import type { HealthCategoryID } from "./healthCategoryID.ts";
import type { HealthCategoryLevel } from "./healthCategoryLevel.ts";
import type { HealthItem } from "./healthItem.ts";

export interface HealthCategory {
  /** It has a Check now */
  canCheck: boolean;
  id: HealthCategoryID;
  /** Errors first, then the oldest */
  items: HealthItem[];
  label: string;
  /**
   * Its worst item's level, null while it has none
   * @nullable
   */
  level: HealthCategoryLevel;
  /** The label inside a sentence, such as storage */
  phrase: string;
  /** Why its last check failed. Its items then show their last known state. Absent while checks succeed */
  unread?: string;
}
