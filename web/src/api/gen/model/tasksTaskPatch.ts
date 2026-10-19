import type { TasksTaskPatchTuning } from "./tasksTaskPatchTuning.ts";

export interface TasksTaskPatch {
  /**
   * Seconds; null resets it to its default
   * @nullable
   */
  cadence?: number | null;
  /**
   * catch-up or wait; null resets it to its default
   * @nullable
   */
  overdue?: string | null;
  /** Each tuning setting's new value by key, all or none; null resets one to its default */
  tuning?: TasksTaskPatchTuning;
}
