import type { ListTrackerWindowsList } from "./listTrackerWindowsList.ts";

export type ListTrackerWindowsParams = {
  /**
   * open lists the windows that owe. history lists those met, missed or released, a missed one grabbed again included. Every window when absent.
   */
  list?: ListTrackerWindowsList;
  /**
   * How many of the newest to skip
   * @minimum 0
   */
  offset?: number;
};
