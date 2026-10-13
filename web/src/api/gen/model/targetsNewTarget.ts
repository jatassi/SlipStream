import type { TargetsNewTargetResolutionsItem } from "./targetsNewTargetResolutionsItem.ts";

export interface TargetsNewTarget {
  /**
   * @minLength 1
   * @maxLength 100
   */
  name: string;
  /**
   * Highest ranked first
   * @minItems 1
   */
  resolutions: TargetsNewTargetResolutionsItem[];
}
