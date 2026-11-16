import type { TasksIdleStateKind } from "./tasksIdleStateKind.ts";

export interface TasksIdleState {
  kind: TasksIdleStateKind;
  /** When the next run is due; now when it is due already */
  next: string;
  /** A setting has each run skip, so the task is off until it changes */
  off: boolean;
}
