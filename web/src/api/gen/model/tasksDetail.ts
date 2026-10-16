import type { TasksRun } from "./tasksRun.ts";
import type { TasksTask } from "./tasksTask.ts";
import type { TasksUnit } from "./tasksUnit.ts";

export interface TasksDetail {
  /** Its runs, newest first, up to the runs kept */
  runs: TasksRun[];
  task: TasksTask;
  /** Each unit it lists now, in run order */
  units: TasksUnit[];
}
