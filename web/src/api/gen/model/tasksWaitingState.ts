import type { TasksWaitingStateKind } from "./tasksWaitingStateKind.ts";
import type { TasksWaitingStateStanding } from "./tasksWaitingStateStanding.ts";

export interface TasksWaitingState {
  kind: TasksWaitingStateKind;
  /** The Node every unit waits for */
  node: string;
  /** When the Controller last heard from the Node, or when the Controller started if it hasn't heard from it since */
  since: string;
  standing: TasksWaitingStateStanding;
}
