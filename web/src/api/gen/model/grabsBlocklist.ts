import type { GrabsBlockEntry } from "./grabsBlockEntry.ts";

export interface GrabsBlocklist {
  /** Up to 50, newest first */
  entries: GrabsBlockEntry[];
  /**
   * How many entries match in all
   * @minimum 0
   */
  total: number;
}
