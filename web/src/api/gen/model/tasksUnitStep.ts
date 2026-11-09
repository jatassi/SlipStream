export interface TasksUnitStep {
  /** @minimum 0 */
  done: number;
  key: string;
  phase: string;
  /**
   * @minimum 0
   * @nullable
   */
  total: number | null;
}
