import type { WorksFieldSpecKind } from "./worksFieldSpecKind.ts";
import type { WorksFieldSpecTier } from "./worksFieldSpecTier.ts";

export interface WorksFieldSpec {
  /**
   * The level's depth, 1 for the outermost; present only on tier level
   * @minimum 1
   */
  depth?: number;
  key: string;
  kind: WorksFieldSpecKind;
  label: string;
  tier: WorksFieldSpecTier;
}
