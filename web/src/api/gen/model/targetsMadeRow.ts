export interface TargetsMadeRow {
  addedAt: string;
  /** @minimum 0 */
  bytes: number;
  /** @minimum 0 */
  files: number;
  /** @minimum 0 */
  seconds: number;
  title: string;
  workId: string;
}
