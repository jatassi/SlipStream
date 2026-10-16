import type { TasksUnitOutcome } from "./tasksUnitOutcome.ts";

export interface TasksUnit {
  /**
   * Why it last failed or was skipped, null otherwise
   * @nullable
   */
  cause: string | null;
  /** It runs once when its Node returns */
  dueOnReturn: boolean;
  /**
   * When it last ended, null before it has run
   * @nullable
   */
  endedAt: string | null;
  key: string;
  label: string;
  /**
   * Null before it has succeeded
   * @nullable
   */
  lastSuccessAt: string | null;
  /** The Node it runs on */
  node: string;
  /**
   * Its last outcome, null before it has run
   * @nullable
   */
  outcome: TasksUnitOutcome;
}
