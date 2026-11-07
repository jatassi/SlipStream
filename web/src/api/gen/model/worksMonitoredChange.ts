export interface WorksMonitoredChange {
  /** Monitor every Item of each Work, or none */
  on: boolean;
  /**
   * A repeated id counts once
   * @minItems 1
   * @maxItems 20000
   */
  workIds: string[];
}
