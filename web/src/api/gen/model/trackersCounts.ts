export interface TrackersCounts {
  /**
   * The ones that haven't met its seeding rule and aren't seeding
   * @minimum 0
   */
  atRisk: number;
  /**
   * The ones whose seeding goal is met
   * @minimum 0
   */
  goalMet: number;
  /**
   * The ones that haven't met its seeding rule yet
   * @minimum 0
   */
  owing: number;
  /** @minimum 0 */
  seeding: number;
  /** @minimum 0 */
  torrents: number;
}
