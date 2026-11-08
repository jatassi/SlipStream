import type { TrackersRuleBodyClears } from "./trackersRuleBodyClears.ts";

export interface TrackersRuleBody {
  /** What meets the rule: its seeding time, its ratio, either of them or both */
  clears: TrackersRuleBodyClears;
  /**
   * At most two decimals; null when the rule doesn't count ratio
   * @nullable
   */
  ratio: number | null;
  /**
   * Seconds; null when the rule doesn't count seeding time
   * @nullable
   */
  seedSeconds: number | null;
}
