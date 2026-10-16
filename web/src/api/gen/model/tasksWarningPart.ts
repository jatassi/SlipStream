import type { TasksWarningPartKind } from "./tasksWarningPartKind.ts";

export interface TasksWarningPart {
  /** The task's page */
  href: string;
  kind: TasksWarningPartKind;
  text: string;
}
