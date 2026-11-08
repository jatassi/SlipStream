import type { ListTrackerWindowsState } from "./listTrackerWindowsState.ts";

export type ListTrackerWindowsParams = {
  /**
   * Only windows in this state; every state when absent
   */
  state?: ListTrackerWindowsState;
  /**
   * How many of the newest to skip
   * @minimum 0
   */
  offset?: number;
};
