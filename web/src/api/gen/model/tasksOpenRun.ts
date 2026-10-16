import type { TasksOpenRunTrigger } from "./tasksOpenRunTrigger.ts";

export interface TasksOpenRun {
  /** @minimum 0 */
  done: number;
  /** The task's name */
  name: string;
  /** The Nodes its units run on */
  nodes: string[];
  runId: number;
  startedAt: string;
  taskId: string;
  /** @minimum 0 */
  total: number;
  trigger: TasksOpenRunTrigger;
}
