import type { HomeJob } from "./homeJob.ts";

export interface HomeJobs {
  /**
   * Every moving Grab
   * @minimum 0
   */
  moving: number;
  /** The moving Grabs, nearest Available first, at most Rows on Home */
  rows: HomeJob[];
}
