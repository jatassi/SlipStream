import type { TasksGroup } from "./tasksGroup.ts";
import type { TasksPart } from "./tasksPart.ts";
import type { TasksSchedule } from "./tasksSchedule.ts";

export interface TasksOverview {
  /** Each group with a shown task, in the page's fixed order */
  groups: TasksGroup[];
  /** The install has more than one Node, so the Tasks page shows a Node column */
  multinode: boolean;
  /**
   * The status sentence's parts, in order
   * @minItems 1
   */
  sentence: TasksPart[];
  settings: TasksSchedule;
}
