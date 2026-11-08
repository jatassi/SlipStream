export interface DownloadclientsTimeLane {
  /**
   * Seconds the goal needs, absent when the goal doesn't count seeding time
   * @minimum 0
   */
  goal?: number;
  /**
   * Seconds of the margin the rule's mark includes, absent when the rule doesn't count seeding time
   * @minimum 0
   */
  margin?: number;
  /**
   * Seconds the rule needs plus the margin, absent when the rule doesn't count seeding time
   * @minimum 0
   */
  rule?: number;
  /**
   * Seconds of seeding the goal counts: the Tracker's count for a private torrent, the torrent client's for a public one
   * @minimum 0
   */
  value: number;
}
