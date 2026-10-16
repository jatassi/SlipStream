import type { LiveEventTasksChangedEvent } from "./liveEventTasksChangedEvent.ts";
import type { TasksChanged } from "./tasksChanged.ts";

export interface LiveEventTasksChanged {
  event: LiveEventTasksChangedEvent;
  data: TasksChanged;
}
