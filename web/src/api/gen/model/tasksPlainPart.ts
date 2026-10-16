import type { TasksPlainPartKind } from "./tasksPlainPartKind.ts";

export interface TasksPlainPart {
  kind: TasksPlainPartKind;
  text: string;
}
