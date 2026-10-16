import type { LiveEventTasksProgressEvent } from "./liveEventTasksProgressEvent.ts";
import type { TasksProgressed } from "./tasksProgressed.ts";

export interface LiveEventTasksProgress {
  event: LiveEventTasksProgressEvent;
  data: TasksProgressed;
}
