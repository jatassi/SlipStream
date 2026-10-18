import type { TasksWaitingStateKind } from "./tasksWaitingStateKind.ts";
import type { TasksWaitingStateStanding } from "./tasksWaitingStateStanding.ts";

export interface TasksWaitingState {
  kind: TasksWaitingStateKind;
  /** The Node every unit waits for */
  node: string;
  /** The Node's last Sync frame, or the Controller's start when it sent none since */
  since: string;
  standing: TasksWaitingStateStanding;
}
