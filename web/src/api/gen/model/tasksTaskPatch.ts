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
}
