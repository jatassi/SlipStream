import type { TrackersRuleClears } from "./trackersRuleClears.ts";
import type { TrackersRuleSource } from "./trackersRuleSource.ts";

export interface TrackersRule {
  /** What meets the rule: its seeding time, its ratio, either of them or both */
  clears: TrackersRuleClears;
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
  /** releases is the latest rule the Tracker's Releases stated; admin is the rule an Admin set, which Releases never replace */
  source: TrackersRuleSource;
}
