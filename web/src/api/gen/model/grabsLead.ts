import type { GrabsWaiting } from "./grabsWaiting.ts";

export interface GrabsLead {
  /**
   * The group the line is about; null for a waiting or idle Work
   * @nullable
   */
  groupKey: string | null;
  /**
   * The Work's groups queued or searching, as in Searching season 3 of 10
   * @minimum 0
   */
  of: number;
  /** The wait the line is about; null unless the Work only waits */
  wait: GrabsWaiting | null;
}
