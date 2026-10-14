import type { TargetsTargetResolutionsItem } from "./targetsTargetResolutionsItem.ts";

export interface TargetsTarget {
  createdAt: string;
  id: string;
  name: string;
  /** Highest ranked first */
  resolutions: TargetsTargetResolutionsItem[];
}
