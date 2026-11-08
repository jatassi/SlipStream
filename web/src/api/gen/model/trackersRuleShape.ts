import type { TrackersRuleShapeClears } from "./trackersRuleShapeClears.ts";

export interface TrackersRuleShape {
  clears: TrackersRuleShapeClears;
  /**
   * null when the rule doesn't count ratio
   * @nullable
   */
  ratio: number | null;
  /**
   * Seconds; null when the rule doesn't count seeding time
   * @nullable
   */
  seedSeconds: number | null;
}
