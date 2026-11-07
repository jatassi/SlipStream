export interface WorksMonitoredChange {
  /** Apply the All preset to each Work, keeping its own aside choice, or None with asides off */
  on: boolean;
  /**
   * A repeated id counts once
   * @minItems 1
   * @maxItems 20000
   */
  workIds: string[];
}
