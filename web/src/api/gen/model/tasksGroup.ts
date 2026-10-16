import type { TasksGroupGroup } from "./tasksGroupGroup.ts";
import type { TasksTask } from "./tasksTask.ts";

export interface TasksGroup {
  group: TasksGroupGroup;
  /** @minItems 1 */
  tasks: TasksTask[];
}
