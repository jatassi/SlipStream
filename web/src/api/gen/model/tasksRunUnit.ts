import type { TasksRunUnitOutcome } from "./tasksRunUnitOutcome.ts";

export interface TasksRunUnit {
  /**
   * Why it failed or was skipped, null when it succeeded
   * @nullable
   */
  cause: string | null;
  key: string;
  label: string;
  /** Empty when the Node is no longer paired */
  node: string;
  /** The Node it ran on */
  nodeId: string;
  outcome: TasksRunUnitOutcome;
}
