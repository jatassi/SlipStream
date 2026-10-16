import type { TasksRunningStateKind } from "./tasksRunningStateKind.ts";
import type { TasksRunningStateTrigger } from "./tasksRunningStateTrigger.ts";

export interface TasksRunningState {
  /** @minimum 0 */
  done: number;
  kind: TasksRunningStateKind;
  runId: number;
  since: string;
  /** @minimum 0 */
  total: number;
  trigger: TasksRunningStateTrigger;
}
