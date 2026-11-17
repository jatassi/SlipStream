import type { CalendarItem } from "./calendarItem.ts";
import type { CalendarWork } from "./calendarWork.ts";
import type { FilesState } from "./filesState.ts";

export interface CalendarRow {
  /** In tree order */
  items: CalendarItem[];
  /** The Items with their level, such as Season 2 · Episodes 1 to 10; empty for one Item without a level */
  itemsWords: string;
  /** The dated kind's key */
  kind: string;
  kindLabel: string;
  /**
   * Its Items with a Target that are met
   * @minimum 0
   */
  met: number;
  /** false only for a row of unmonitored Items, which unmonitored asks for */
  monitored: boolean;
  /**
   * Its Items with a Target
   * @minimum 0
   */
  of: number;
  /** The most urgent state among its Items, in the order failed, missing, downloading, upgradable, unreleased, met; null while none of them has a Target */
  state: FilesState | null;
  work: CalendarWork;
}
