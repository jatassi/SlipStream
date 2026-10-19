import type { SettingField } from "./settingField.ts";
import type { TasksRun } from "./tasksRun.ts";
import type { TasksTask } from "./tasksTask.ts";
import type { TasksUnit } from "./tasksUnit.ts";

export interface TasksDetail {
  /** Its runs, newest first, up to the runs kept */
  runs: TasksRun[];
  task: TasksTask;
  /** Settings its owner stores that tune what it does; PATCH sets them under tuning by key */
  tuning: SettingField[];
  /** Each unit it lists now, in run order */
  units: TasksUnit[];
}
