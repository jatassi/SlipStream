export interface TrackersCounts {
  /**
   * The ones that owe it seeding and aren't seeding
   * @minimum 0
   */
  atRisk: number;
  /**
   * The ones whose seeding goal is met
   * @minimum 0
   */
  goalMet: number;
  /**
   * The ones that still owe it seeding
   * @minimum 0
   */
  owing: number;
  /** @minimum 0 */
  seeding: number;
  /** @minimum 0 */
  torrents: number;
}
