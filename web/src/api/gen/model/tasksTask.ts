import type { SettingChoiceField } from "./settingChoiceField.ts";
import type { SettingNoun } from "./settingNoun.ts";
import type { TasksCadence } from "./tasksCadence.ts";
import type { TasksRun } from "./tasksRun.ts";
import type { TasksRunNow } from "./tasksRunNow.ts";
import type { TasksState } from "./tasksState.ts";
import type { TasksTaskGroup } from "./tasksTaskGroup.ts";

export interface TasksTask {
  about: string;
  cadence: TasksCadence;
  group: TasksTaskGroup;
  id: string;
  /** The latest finished run, absent before one has */
  lastRun?: TasksRun;
  name: string;
  /** The Nodes its units run on */
  nodes: string[];
  /** What its progress counts: 3 of 12 steps */
  noun: SettingNoun;
  overdue: SettingChoiceField;
  runNow: TasksRunNow;
  state: TasksState;
}
