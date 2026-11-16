import type { GrabsQueueStatus } from "./grabsQueueStatus.ts";
import type { GrabsWantedRow } from "./grabsWantedRow.ts";

export interface GrabsWantedPage {
  /**
   * Works with only gone files, left out unless the filter asks for gone
   * @minimum 0
   */
  leftGone: number;
  /**
   * The cursor of the next page; null on the last
   * @nullable
   */
  next: string | null;
  /** In the sort's order, ties by Work id */
  rows: GrabsWantedRow[];
  status: GrabsQueueStatus;
  /**
   * The Works the filter lists
   * @minimum 0
   */
  total: number;
}
