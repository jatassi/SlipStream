export interface BackupsFailure {
  /** When the newest failed */
  at: string;
  /** When the first of the failed Backups since the newest good one was taken */
  since: string;
}
