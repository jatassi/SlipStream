import type { WorksFieldSpecInItem } from "./worksFieldSpecInItem.ts";
import type { WorksFieldSpecKind } from "./worksFieldSpecKind.ts";
import type { WorksFieldSpecTier } from "./worksFieldSpecTier.ts";
import type { WorksListing } from "./worksListing.ts";

export interface WorksFieldSpec {
  /**
   * The key of a Work artwork kind shown beside it, such as a studio's logo; null when none
   * @nullable
   */
  art: string | null;
  /**
   * The level's depth, 1 for the outermost; present only on tier level
   * @minimum 1
   */
  depth?: number;
  /** Every place the Work page shows it */
  in: WorksFieldSpecInItem[];
  key: string;
  kind: WorksFieldSpecKind;
  label: string;
  /** Where the Library uses it; all false for a field of a level or an Item */
  list: WorksListing;
  tier: WorksFieldSpecTier;
  /**
   * The key of a text field of the same tier shown in brackets after it; null when none
   * @nullable
   */
  with: string | null;
}
