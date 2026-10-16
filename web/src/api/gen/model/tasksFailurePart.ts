import type { TasksFailurePartKind } from "./tasksFailurePartKind.ts";

export interface TasksFailurePart {
  /** The task's page */
  href: string;
  kind: TasksFailurePartKind;
  text: string;
}
