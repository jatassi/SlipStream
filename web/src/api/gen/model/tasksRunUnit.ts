import type { TasksRunUnitOutcome } from "./tasksRunUnitOutcome.ts";

export interface TasksRunUnit {
  /**
   * Why it failed or was skipped, null when it succeeded
   * @nullable
   */
  cause: string | null;
  key: string;
  label: string;
  /** The Node it ran on, still named once removed; empty when no Node has its id */
  node: string;
  /** The Node it ran on */
  nodeId: string;
  /** The Node it ran on was removed since */
  nodeRemoved: boolean;
  outcome: TasksRunUnitOutcome;
}
