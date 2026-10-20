import type { WorksArt } from "./worksArt.ts";
import type { WorksAvailability } from "./worksAvailability.ts";
import type { WorksFieldValue } from "./worksFieldValue.ts";
import type { WorksRating } from "./worksRating.ts";

export interface WorksItem {
  /** By kind */
  artwork: WorksArt[];
  /** The day the Item counts as released and the rule that chose it; null while no date is known */
  availability: WorksAvailability;
  fields: WorksFieldValue[];
  id: string;
  key: string;
  label: string;
  /**
   * Null when the Content Type declares no levels
   * @nullable
   */
  levelId: string | null;
  monitored: boolean;
  /** @minimum 1 */
  position: number;
  /** By source */
  ratings: WorksRating[];
  /**
   * When the source stopped listing it; null while it lists it
   * @nullable
   */
  withdrawnAt: string | null;
}
