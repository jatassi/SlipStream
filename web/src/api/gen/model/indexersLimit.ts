import type { IndexersLimitPer } from "./indexersLimitPer.ts";

export interface IndexersLimit {
  /**
   * 0 is no limit
   * @minimum 0
   * @maximum 100000
   */
  count: number;
  /** The sliding window it counts in */
  per: IndexersLimitPer;
}
