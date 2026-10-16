import type { TasksRunNowReason } from "./tasksRunNowReason.ts";

export interface TasksRunNow {
  available: boolean;
  /** Why Run now is unavailable; absent when it is available */
  reason?: TasksRunNowReason;
}
