import type { TasksIdleStateKind } from "./tasksIdleStateKind.ts";

export interface TasksIdleState {
  kind: TasksIdleStateKind;
  /** When the next run is due; now when it is due already */
  next: string;
}
