import type { HomeNeed } from "./homeNeed.ts";

export interface HomeNeedsYou {
  /**
   * Failed Grabs that need attention, counted up to 100; 100 means 100 or more
   * @minimum 0
   * @maximum 100
   */
  failed: number;
  /** Newest first, at most Rows on Home */
  rows: HomeNeed[];
  /**
   * Grabs that wait on an Admin
   * @minimum 0
   */
  waiting: number;
}
