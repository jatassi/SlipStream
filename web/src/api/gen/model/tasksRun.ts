import type { TasksRunResult } from "./tasksRunResult.ts";
import type { TasksRunTrigger } from "./tasksRunTrigger.ts";
import type { TasksRunUnit } from "./tasksRunUnit.ts";

export interface TasksRun {
  /**
   * The first failure's cause, null when none
   * @nullable
   */
  cause: string | null;
  /**
   * Null while it runs
   * @nullable
   */
  endedAt: string | null;
  id: number;
  /** The Nodes its units ran on */
  nodes: string[];
  /**
   * Null while it runs
   * @nullable
   */
  result: TasksRunResult;
  startedAt: string;
  /** Such as 5 of 6 steps, seedbox offline; empty while it runs */
  summary: string;
  trigger: TasksRunTrigger;
  /** Each unit's outcome as the run ended, kept after the unit is gone; empty while it runs */
  units: TasksRunUnit[];
}
