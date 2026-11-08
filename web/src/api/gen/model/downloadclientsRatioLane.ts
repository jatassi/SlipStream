export interface DownloadclientsRatioLane {
  /** The goal's ratio, absent when the goal doesn't count ratio */
  goal?: number;
  /** The rule's ratio, absent when the rule doesn't count ratio */
  rule?: number;
  /** @minimum 0 */
  value: number;
}
