import type { TasksWaitingStateKind } from "./tasksWaitingStateKind.ts";
import type { TasksWaitingStateLink } from "./tasksWaitingStateLink.ts";

export interface TasksWaitingState {
  kind: TasksWaitingStateKind;
  link: TasksWaitingStateLink;
  /** The Node every unit waits for */
  node: string;
  /** When the Node's link entered that state */
  since: string;
}
